package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/chiamck/hotel-booking/internal/lock"
	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/service"
)

type stubBookingRepository struct {
	bookings map[string]models.Booking
	created  int
}

func newStubBookingRepository() *stubBookingRepository {
	return &stubBookingRepository{bookings: make(map[string]models.Booking)}
}

func (s *stubBookingRepository) FindByIdempotencyKey(key string) (*models.Booking, error) {
	booking, ok := s.bookings[key]
	if !ok {
		return nil, nil
	}
	return &booking, nil
}

func (s *stubBookingRepository) Create(params repository.CreateBookingParams) (models.Booking, error) {
	if existing, ok := s.bookings[params.IdempotencyKey]; ok {
		return existing, repository.ErrIdempotencyConflict
	}

	s.created++
	booking := models.Booking{
		ID:             s.created,
		RoomID:         params.RoomID,
		CustomerID:     params.CustomerID,
		StartTime:      params.CheckIn,
		EndTime:        params.CheckOut,
		Status:         "confirmed",
		TotalAmount:    750,
		PricePerNight:  150,
		IdempotencyKey: params.IdempotencyKey,
	}
	s.bookings[params.IdempotencyKey] = booking
	return booking, nil
}

type stubLocker struct {
	acquired bool
}

func (s *stubLocker) TryLock(ctx context.Context, key string, ttl time.Duration) (func(), bool, error) {
	if !s.acquired {
		return func() {}, false, nil
	}
	return func() {}, true, nil
}

func TestBookingServiceCreate(t *testing.T) {
	repo := newStubBookingRepository()
	svc := service.NewBookingService(repo, &stubLocker{acquired: true})

	result, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:         2,
		CustomerID:     1,
		CheckIn:        "2026-07-01",
		CheckOut:       "2026-07-06",
		IdempotencyKey: "key-1",
	})
	if err != nil {
		t.Fatalf("create booking: %v", err)
	}
	if !result.Created || result.Booking.ID != 1 {
		t.Fatalf("expected created booking, got %+v", result)
	}

	replay, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:         2,
		CustomerID:     1,
		CheckIn:        "2026-07-01",
		CheckOut:       "2026-07-06",
		IdempotencyKey: "key-1",
	})
	if err != nil {
		t.Fatalf("replay booking: %v", err)
	}
	if replay.Created || replay.Booking.ID != 1 {
		t.Fatalf("expected idempotent replay, got %+v", replay)
	}
}

func TestBookingServiceRequiresIdempotencyKey(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: true})

	_, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
	if err != service.ErrIdempotencyKeyRequired {
		t.Fatalf("expected ErrIdempotencyKeyRequired, got %v", err)
	}
}

func TestBookingServiceLockNotAcquired(t *testing.T) {
	svc := service.NewBookingService(newStubBookingRepository(), &stubLocker{acquired: false})

	_, err := svc.Create(context.Background(), service.CreateBookingInput{
		RoomID:         2,
		CustomerID:     1,
		CheckIn:        "2026-07-01",
		CheckOut:       "2026-07-06",
		IdempotencyKey: "key-2",
	})
	if err != service.ErrBookingLockNotAcquired {
		t.Fatalf("expected ErrBookingLockNotAcquired, got %v", err)
	}
}

var _ lock.DistributedLock = (*stubLocker)(nil)
