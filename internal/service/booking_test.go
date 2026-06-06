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

func (s *stubBookingRepository) ListBlockingByRoomOverlap(roomID int, rangeStart, rangeEnd time.Time) ([]models.Booking, error) {
	return nil, nil
}

type availabilityStubRepo struct {
	list []models.Booking
}

func (a *availabilityStubRepo) Create(params repository.CreateBookingParams) (models.Booking, error) {
	return models.Booking{}, errors.New("not implemented")
}

func (a *availabilityStubRepo) ListBlockingByRoomOverlap(roomID int, rangeStart, rangeEnd time.Time) ([]models.Booking, error) {
	return a.list, nil
}

var _ repository.BookingRepository = (*availabilityStubRepo)(nil)

type memoryIdempotencyStore struct {
	mu sync.Mutex
	m  map[string]models.Booking
}

func newMemoryIdempotencyStore() *memoryIdempotencyStore {
	return &memoryIdempotencyStore{m: make(map[string]models.Booking)}
}

func (s *memoryIdempotencyStore) GetBooking(ctx context.Context, idempotencyKey string) (*models.Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.m[idempotencyKey]
	if !ok {
		return nil, nil
	}
	cp := b
	return &cp, nil
}

func (s *memoryIdempotencyStore) SetBooking(ctx context.Context, idempotencyKey string, b models.Booking, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[idempotencyKey] = b
	return nil
}

var _ idempotency.BookingStore = (*memoryIdempotencyStore)(nil)

type stubLocker struct {
	acquired bool
}

func (s *stubLocker) TryLock(ctx context.Context, key string, exp time.Duration) (func(), bool, error) {
	if !s.acquired {
		return func() {}, false, nil
	}
	return func() {}, true, nil
}

func TestBookingServiceCreate(t *testing.T) {
	repo := newStubBookingRepository()
	idem := newMemoryIdempotencyStore()
	svc := service.NewBookingService(repo, &stubLocker{acquired: true}, idem)

	result, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}
	if !result.Created || result.Booking.ID != 1 {
		t.Fatalf("expected created booking, got %+v", result)
	}

	replay, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
	if err != nil {
		t.Fatalf("replay booking: %v", err)
	}
	if replay.Created || replay.Booking.ID != 1 {
		t.Fatalf("expected idempotent replay, got %+v", replay)
	}
}

func TestBookingServiceCreateUsesDerivedIdempotencyKey(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: true}, newMemoryIdempotencyStore())

	_, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
	if err != nil {
		t.Fatalf("expected success with derived idempotency key, got %v", err)
	}
}

func TestBookingServiceLockNotAcquired(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: false}, newMemoryIdempotencyStore())

	_, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
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

	res, err := svc.GetRoomAvailability(context.Background(), 2, "2026-07-01", "2026-07-31")
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

func TestBookingServiceRoomAvailabilityRejectsInvalidRange(t *testing.T) {
	svc := service.NewBookingService(&availabilityStubRepo{}, &stubLocker{acquired: true}, newMemoryIdempotencyStore())
	_, err := svc.GetRoomAvailability(context.Background(), 2, "2026-08-10", "2026-08-01")
	if err != service.ErrInvalidAvailabilityRange {
		t.Fatalf("expected ErrInvalidAvailabilityRange, got %v", err)
	}
}

func TestBookingServiceCreateRejectsInvalidRoomID(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: true}, newMemoryIdempotencyStore())

	_, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:     0,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
	if err != service.ErrInvalidRoomID {
		t.Fatalf("expected ErrInvalidRoomID, got %v", err)
	}
}

func TestBookingServiceCreateRejectsInvalidDateRange(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: true}, newMemoryIdempotencyStore())

	_, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-06",
		CheckOut:   "2026-07-01",
	})
	if err != service.ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}

type failingIdempotencyStore struct{}

func (failingIdempotencyStore) GetBooking(context.Context, string) (*models.Booking, error) {
	return nil, errors.New("redis down")
}

func (failingIdempotencyStore) SetBooking(context.Context, string, models.Booking, time.Duration) error {
	return errors.New("redis down")
}

func TestBookingServiceCreateIdempotencyCacheError(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: true}, failingIdempotencyStore{})

	_, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
	if !service.IsIdempotencyCacheError(err) {
		t.Fatalf("expected idempotency cache error, got %v", err)
	}
}

var _ lock.DistributedLock = (*stubLocker)(nil)
