package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
)

type createBookingRepoStub struct {
	booking models.Booking
	err     error
	calls   int
}

func (s *createBookingRepoStub) Create(repository.CreateBookingParams) (models.Booking, error) {
	s.calls++
	if s.err != nil {
		return models.Booking{}, s.err
	}
	if s.booking.ID == 0 {
		s.booking = models.Booking{
			ID:         1,
			RoomID:     2,
			CustomerID: 1,
			Status:     "confirmed",
		}
	}
	return s.booking, nil
}

func (s *createBookingRepoStub) ListActiveBookingsOverlappingRange(int, time.Time, time.Time) ([]models.Booking, error) {
	return nil, nil
}

func (s *createBookingRepoStub) List(repository.ListBookingsParams) (models.BookingListPage, error) {
	return models.BookingListPage{}, nil
}

type createIdempotencyStub struct {
	used   bool
	setErr error
	getErr error
	sets   int
}

func (s *createIdempotencyStub) CheckIdempotent(context.Context, string) (bool, error) {
	if s.getErr != nil {
		return false, s.getErr
	}
	return s.used, nil
}

func (s *createIdempotencyStub) SetIdempotent(context.Context, string, time.Duration) error {
	s.sets++
	return s.setErr
}

type createLockStub struct {
	acquired bool
}

func (s *createLockStub) TryLock(context.Context, string, time.Duration) (bool, error) {
	if !s.acquired {
		return false, nil
	}
	return true, nil
}

func (s *createLockStub) Unlock(context.Context, string) error {
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

func TestBookingLockKey(t *testing.T) {
	if got := bookingLockKey(42); got != "booking:lock:room:42" {
		t.Fatalf("got %q", got)
	}
}

func TestBookingErrorMessageAndClassifiers(t *testing.T) {
	cases := []struct {
		err      error
		msg      string
		conflict bool
		cache    bool
	}{
		{repository.ErrBookingOverlap, "room is already booked for the selected dates", true, false},
		{ErrBookingLockNotAcquired, "another booking is in progress for this room, please retry", true, false},
		{ErrDuplicateBookingRequest, "this booking request was already processed", true, false},
		{ErrIdempotencyCache, "idempotency cache unavailable", false, true},
	}

	for _, tc := range cases {
		if got := BookingErrorMessage(tc.err); got != tc.msg {
			t.Errorf("%v: message got %q want %q", tc.err, got, tc.msg)
		}
		if IsBookingConflictError(tc.err) != tc.conflict {
			t.Errorf("%v: conflict got %v want %v", tc.err, IsBookingConflictError(tc.err), tc.conflict)
		}
		if IsIdempotencyCacheError(tc.err) != tc.cache {
			t.Errorf("%v: cache got %v want %v", tc.err, IsIdempotencyCacheError(tc.err), tc.cache)
		}
	}
}

func TestCreateRejectsDuplicateIdempotencyKey(t *testing.T) {
	repo := &createBookingRepoStub{}
	idem := &createIdempotencyStub{used: true}
	svc := NewBookingService(repo, &createLockStub{acquired: true}, idem)

	_, err := svc.Create(context.Background(), testCreateBookingParams())
	if err != ErrDuplicateBookingRequest {
		t.Fatalf("expected duplicate booking error, got %v", err)
	}
	if repo.calls != 0 {
		t.Fatalf("expected repo not called, calls=%d", repo.calls)
	}
}

func TestCreatePropagatesRepoErrors(t *testing.T) {
	repo := &createBookingRepoStub{err: repository.ErrBookingOverlap}
	svc := NewBookingService(repo, &createLockStub{acquired: true}, &createIdempotencyStub{})

	_, err := svc.Create(context.Background(), testCreateBookingParams())
	if !errors.Is(err, repository.ErrBookingOverlap) {
		t.Fatalf("expected %v, got %v", repository.ErrBookingOverlap, err)
	}
	if repo.calls != 1 {
		t.Fatalf("expected 1 create call, got %d", repo.calls)
	}
}

func TestCreateCachesBookingAfterInsert(t *testing.T) {
	repo := &createBookingRepoStub{}
	idem := &createIdempotencyStub{}
	svc := NewBookingService(repo, &createLockStub{acquired: true}, idem)

	_, err := svc.Create(context.Background(), testCreateBookingParams())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if repo.calls != 1 || idem.sets != 1 {
		t.Fatalf("expected create + mark used, calls=%d sets=%d", repo.calls, idem.sets)
	}
}
