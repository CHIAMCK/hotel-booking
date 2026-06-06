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

func (s *createBookingRepoStub) ListBlockingByRoomOverlap(int, time.Time, time.Time) ([]models.Booking, error) {
	return nil, nil
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

func (s *createLockStub) TryLock(context.Context, string, time.Duration) (func(), bool, error) {
	if !s.acquired {
		return func() {}, false, nil
	}
	return func() {}, true, nil
}

func TestParseCreateBookingInputValid(t *testing.T) {
	params, key, err := parseCreateBookingInput(CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
	if err != nil {
		t.Fatalf("parseCreateBookingInput: %v", err)
	}
	if params.RoomID != 2 || params.CustomerID != 1 {
		t.Fatalf("unexpected params: %+v", params)
	}
	if !params.CheckOut.After(params.CheckIn) {
		t.Fatal("expected check-out after check-in")
	}
	if key != "booking:auto:2:1:2026-07-01:2026-07-06" {
		t.Fatalf("unexpected idempotency key: %q", key)
	}
}

func TestParseCreateBookingInputRejectsInvalidCustomerID(t *testing.T) {
	_, _, err := parseCreateBookingInput(CreateBookingInput{
		RoomID:     2,
		CustomerID: 0,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
	if err != ErrInvalidCustomerID {
		t.Fatalf("expected ErrInvalidCustomerID, got %v", err)
	}
}

func TestParseCreateBookingInputRejectsSameDayStay(t *testing.T) {
	_, _, err := parseCreateBookingInput(CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-01",
	})
	if err != ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestParseCreateBookingInputRejectsInvalidCheckOut(t *testing.T) {
	_, _, err := parseCreateBookingInput(CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "07-06-2026",
	})
	if err != ErrInvalidCheckOut {
		t.Fatalf("expected ErrInvalidCheckOut, got %v", err)
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
		validate bool
		conflict bool
		notFound bool
		cache    bool
	}{
		{ErrInvalidRoomID, "room_id must be a positive integer", true, false, false, false},
		{repository.ErrRoomNotFound, "room not found", false, false, true, false},
		{repository.ErrRoomNotAvailable, "room is not available", false, true, false, false},
		{repository.ErrBookingOverlap, "room is already booked for the selected dates", false, true, false, false},
		{ErrBookingLockNotAcquired, "another booking is in progress for this room, please retry", false, true, false, false},
		{ErrDuplicateBookingRequest, "this booking request was already processed", false, true, false, false},
		{ErrIdempotencyCache, "idempotency cache unavailable", false, false, false, true},
	}

	for _, tc := range cases {
		if got := BookingErrorMessage(tc.err); got != tc.msg {
			t.Errorf("%v: message got %q want %q", tc.err, got, tc.msg)
		}
		if IsBookingValidationError(tc.err) != tc.validate {
			t.Errorf("%v: validate got %v want %v", tc.err, IsBookingValidationError(tc.err), tc.validate)
		}
		if IsBookingConflictError(tc.err) != tc.conflict {
			t.Errorf("%v: conflict got %v want %v", tc.err, IsBookingConflictError(tc.err), tc.conflict)
		}
		if IsBookingNotFoundError(tc.err) != tc.notFound {
			t.Errorf("%v: notFound got %v want %v", tc.err, IsBookingNotFoundError(tc.err), tc.notFound)
		}
		if IsIdempotencyCacheError(tc.err) != tc.cache {
			t.Errorf("%v: cache got %v want %v", tc.err, IsIdempotencyCacheError(tc.err), tc.cache)
		}
	}
}

func TestAvailabilityErrorMessageAndClassifiers(t *testing.T) {
	if got := AvailabilityErrorMessage(ErrInvalidAvailabilityRange); got != "to must be on or after from" {
		t.Fatalf("got %q", got)
	}
	if !IsAvailabilityValidationError(ErrAvailabilityWindowTooLarge) {
		t.Fatal("expected availability validation error")
	}
	if IsAvailabilityValidationError(repository.ErrRoomNotFound) {
		t.Fatal("room not found is not an availability validation error")
	}
}

func TestCreateRejectsDuplicateIdempotencyKey(t *testing.T) {
	repo := &createBookingRepoStub{}
	idem := &createIdempotencyStub{used: true}
	svc := NewBookingService(repo, &createLockStub{acquired: true}, idem)

	_, err := svc.Create(context.Background(), CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
	if err != ErrDuplicateBookingRequest {
		t.Fatalf("expected duplicate booking error, got %v", err)
	}
	if repo.calls != 0 {
		t.Fatalf("expected repo not called, calls=%d", repo.calls)
	}
}

func TestCreatePropagatesRepoErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"room not found", repository.ErrRoomNotFound},
		{"room not available", repository.ErrRoomNotAvailable},
		{"overlap", repository.ErrBookingOverlap},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &createBookingRepoStub{err: tc.err}
			svc := NewBookingService(repo, &createLockStub{acquired: true}, &createIdempotencyStub{})

			_, err := svc.Create(context.Background(), CreateBookingInput{
				RoomID:     2,
				CustomerID: 1,
				CheckIn:    "2026-07-01",
				CheckOut:   "2026-07-06",
			})
			if !errors.Is(err, tc.err) {
				t.Fatalf("expected %v, got %v", tc.err, err)
			}
			if repo.calls != 1 {
				t.Fatalf("expected one repo call, got %d", repo.calls)
			}
		})
	}
}

func TestCreateCachesBookingAfterInsert(t *testing.T) {
	repo := &createBookingRepoStub{}
	idem := &createIdempotencyStub{}
	svc := NewBookingService(repo, &createLockStub{acquired: true}, idem)

	_, err := svc.Create(context.Background(), CreateBookingInput{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if repo.calls != 1 || idem.sets != 1 {
		t.Fatalf("expected create + mark used, calls=%d sets=%d", repo.calls, idem.sets)
	}
}
