package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/chiamck/hotel-booking/internal/idempotency"
	"github.com/chiamck/hotel-booking/internal/lock"
	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/service"
)

type stubBookingRepository struct {
	created int
}

func newStubBookingRepository() *stubBookingRepository {
	return &stubBookingRepository{}
}

func (s *stubBookingRepository) Create(params repository.CreateBookingParams) (models.Booking, error) {
	s.created++
	return models.Booking{
		ID:            s.created,
		RoomID:        params.RoomID,
		CustomerID:    params.CustomerID,
		StartTime:     params.CheckIn,
		EndTime:       params.CheckOut,
		Status:        "confirmed",
		TotalAmount:   750,
		PricePerNight: 150,
	}, nil
}

func (s *stubBookingRepository) ListActiveBookingsOverlappingRange(roomID int, rangeStart, rangeEnd time.Time) ([]models.Booking, error) {
	return nil, nil
}

func (s *stubBookingRepository) List(params repository.ListBookingsParams) (models.BookingListPage, error) {
	return models.BookingListPage{
		Bookings: []models.Booking{},
		Pagination: models.Pagination{
			Page:  params.Page,
			Limit: params.Limit,
		},
	}, nil
}

type availabilityStubRepo struct {
	list []models.Booking
}

func (a *availabilityStubRepo) Create(params repository.CreateBookingParams) (models.Booking, error) {
	return models.Booking{}, errors.New("not implemented")
}

func (a *availabilityStubRepo) ListActiveBookingsOverlappingRange(roomID int, rangeStart, rangeEnd time.Time) ([]models.Booking, error) {
	return a.list, nil
}

func (a *availabilityStubRepo) List(repository.ListBookingsParams) (models.BookingListPage, error) {
	return models.BookingListPage{}, errors.New("not implemented")
}

var _ repository.BookingRepository = (*availabilityStubRepo)(nil)

type memoryIdempotencyStore struct {
	mu sync.Mutex
	m  map[string]struct{}
}

func newMemoryIdempotencyStore() *memoryIdempotencyStore {
	return &memoryIdempotencyStore{m: make(map[string]struct{})}
}

func (s *memoryIdempotencyStore) CheckIdempotent(ctx context.Context, idempotencyKey string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.m[idempotencyKey]
	return ok, nil
}

func (s *memoryIdempotencyStore) SetIdempotent(ctx context.Context, idempotencyKey string, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[idempotencyKey] = struct{}{}
	return nil
}

var _ idempotency.BookingStore = (*memoryIdempotencyStore)(nil)

type stubLocker struct {
	acquired bool
}

func (s *stubLocker) TryLock(ctx context.Context, key string, exp time.Duration) (bool, error) {
	if !s.acquired {
		return false, nil
	}
	return true, nil
}

func (s *stubLocker) Unlock(context.Context, string) error {
	return nil
}

func testCreateBookingParams() repository.CreateBookingParams {
	checkIn := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	return repository.CreateBookingParams{
		RoomID:        2,
		CustomerID:    1,
		CheckIn:       checkIn,
		CheckOut:      checkOut,
		Nights:        5,
		TotalAmount:   750,
		PricePerNight: 150,
	}
}

func TestBookingServiceCreate(t *testing.T) {
	repo := newStubBookingRepository()
	idem := newMemoryIdempotencyStore()
	svc := service.NewBookingService(repo, &stubLocker{acquired: true}, idem)

	booking, err := svc.Create(context.Background(), testCreateBookingParams())
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}
	if booking.ID != 1 {
		t.Fatalf("expected created booking, got booking=%+v", booking)
	}

	_, err = svc.Create(context.Background(), testCreateBookingParams())
	if err != service.ErrDuplicateBookingRequest {
		t.Fatalf("expected duplicate booking error, got %v", err)
	}
}

func TestBookingServiceCreateUsesDerivedIdempotencyKey(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: true}, newMemoryIdempotencyStore())

	_, err := svc.Create(context.Background(), testCreateBookingParams())
	if err != nil {
		t.Fatalf("expected success with derived idempotency key, got %v", err)
	}
}

func TestBookingServiceLockNotAcquired(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: false}, newMemoryIdempotencyStore())

	_, err := svc.Create(context.Background(), testCreateBookingParams())
	if err != service.ErrBookingLockNotAcquired {
		t.Fatalf("expected ErrBookingLockNotAcquired, got %v", err)
	}
}

func TestBookingServiceRoomAvailabilityNights(t *testing.T) {
	checkIn := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	repo := &availabilityStubRepo{
		list: []models.Booking{
			{ID: 1, RoomID: 2, StartTime: checkIn, EndTime: checkOut, Status: "confirmed"},
		},
	}
	svc := service.NewBookingService(repo, &stubLocker{acquired: true}, newMemoryIdempotencyStore())

	res, err := svc.GetRoomAvailability(context.Background(), 2,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("GetRoomAvailability: %v", err)
	}
	want := []string{"2026-07-01", "2026-07-02", "2026-07-03", "2026-07-04", "2026-07-05"}
	if len(res.UnavailableDates) != len(want) {
		t.Fatalf("got %v, want %v", res.UnavailableDates, want)
	}
	for i, d := range want {
		if res.UnavailableDates[i] != d {
			t.Fatalf("index %d: got %q want %q", i, res.UnavailableDates[i], d)
		}
	}
}

type failingIdempotencyStore struct{}

func (failingIdempotencyStore) CheckIdempotent(context.Context, string) (bool, error) {
	return false, errors.New("redis down")
}

func (failingIdempotencyStore) SetIdempotent(context.Context, string, time.Duration) error {
	return errors.New("redis down")
}

func TestBookingServiceList(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: true}, newMemoryIdempotencyStore())

	page, err := svc.List(context.Background(), repository.ListBookingsParams{
		CustomerID: 1,
		Page:       1,
		Limit:      20,
	})
	if err != nil {
		t.Fatalf("list bookings: %v", err)
	}
	if page.Pagination.Page != 1 || page.Pagination.Limit != 20 {
		t.Fatalf("unexpected pagination: %+v", page.Pagination)
	}
}

func TestBookingServiceCreateIdempotencyCacheError(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: true}, failingIdempotencyStore{})

	_, err := svc.Create(context.Background(), testCreateBookingParams())
	if !service.IsIdempotencyCacheError(err) {
		t.Fatalf("expected idempotency cache error, got %v", err)
	}
}

var _ lock.DistributedLock = (*stubLocker)(nil)
