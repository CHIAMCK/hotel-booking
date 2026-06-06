package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
)

type availabilityRepoStub struct {
	bookings []models.Booking
	err      error
}

func (s *availabilityRepoStub) Create(repository.CreateBookingParams) (models.Booking, error) {
	return models.Booking{}, errors.New("not implemented")
}

func (s *availabilityRepoStub) ListBlockingByRoomOverlap(int, time.Time, time.Time) ([]models.Booking, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.bookings, nil
}

func TestOccupiedNightsUTCHalfOpenRange(t *testing.T) {
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)

	got := occupiedNightsUTC(start, end)
	want := []string{"2026-07-01", "2026-07-02", "2026-07-03", "2026-07-04", "2026-07-05"}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, d := range want {
		if got[i] != d {
			t.Fatalf("index %d: got %q want %q", i, got[i], d)
		}
	}
}

func TestOccupiedNightsUTCSingleNight(t *testing.T) {
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)

	got := occupiedNightsUTC(start, end)
	if len(got) != 1 || got[0] != "2026-08-10" {
		t.Fatalf("got %v, want [2026-08-10]", got)
	}
}

func TestOccupiedNightsUTCEmptyWhenCheckOutOnCheckIn(t *testing.T) {
	day := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	got := occupiedNightsUTC(day, day)
	if len(got) != 0 {
		t.Fatalf("expected no nights, got %v", got)
	}
}

func TestGetRoomAvailabilityEmpty(t *testing.T) {
	svc := NewBookingService(&availabilityRepoStub{}, nil, nil)

	res, err := svc.GetRoomAvailability(context.Background(), 2, "2026-07-01", "2026-07-10")
	if err != nil {
		t.Fatalf("GetRoomAvailability: %v", err)
	}
	if res.RoomID != 2 || res.From != "2026-07-01" || res.To != "2026-07-10" {
		t.Fatalf("unexpected metadata: %+v", res)
	}
	if len(res.UnavailableDates) != 0 {
		t.Fatalf("expected no unavailable dates, got %v", res.UnavailableDates)
	}
}

func TestGetRoomAvailabilityClipsToWindow(t *testing.T) {
	checkIn := time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)
	svc := NewBookingService(&availabilityRepoStub{
		bookings: []models.Booking{
			{ID: 1, RoomID: 2, StartTime: checkIn, EndTime: checkOut, Status: "confirmed"},
		},
	}, nil, nil)

	res, err := svc.GetRoomAvailability(context.Background(), 2, "2026-07-01", "2026-07-03")
	if err != nil {
		t.Fatalf("GetRoomAvailability: %v", err)
	}
	want := []string{"2026-07-01", "2026-07-02", "2026-07-03"}
	if len(res.UnavailableDates) != len(want) {
		t.Fatalf("got %v, want %v", res.UnavailableDates, want)
	}
	for i, d := range want {
		if res.UnavailableDates[i] != d {
			t.Fatalf("index %d: got %q want %q", i, res.UnavailableDates[i], d)
		}
	}
}

func TestGetRoomAvailabilityInvalidRoomID(t *testing.T) {
	svc := NewBookingService(&availabilityRepoStub{}, nil, nil)

	_, err := svc.GetRoomAvailability(context.Background(), 0, "2026-07-01", "2026-07-10")
	if err != ErrInvalidRoomID {
		t.Fatalf("expected ErrInvalidRoomID, got %v", err)
	}
}

func TestGetRoomAvailabilityInvalidFromDate(t *testing.T) {
	svc := NewBookingService(&availabilityRepoStub{}, nil, nil)

	_, err := svc.GetRoomAvailability(context.Background(), 2, "not-a-date", "2026-07-10")
	if err != ErrInvalidAvailabilityFrom {
		t.Fatalf("expected ErrInvalidAvailabilityFrom, got %v", err)
	}
}

func TestGetRoomAvailabilityInvalidToDate(t *testing.T) {
	svc := NewBookingService(&availabilityRepoStub{}, nil, nil)

	_, err := svc.GetRoomAvailability(context.Background(), 2, "2026-07-01", "bad")
	if err != ErrInvalidAvailabilityTo {
		t.Fatalf("expected ErrInvalidAvailabilityTo, got %v", err)
	}
}

func TestGetRoomAvailabilityWindowTooLarge(t *testing.T) {
	svc := NewBookingService(&availabilityRepoStub{}, nil, nil)

	_, err := svc.GetRoomAvailability(context.Background(), 2, "2026-01-01", "2028-01-02")
	if err != ErrAvailabilityWindowTooLarge {
		t.Fatalf("expected ErrAvailabilityWindowTooLarge, got %v", err)
	}
}

func TestGetRoomAvailabilityMergesMultipleBookings(t *testing.T) {
	svc := NewBookingService(&availabilityRepoStub{
		bookings: []models.Booking{
			{
				ID:        1,
				StartTime: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
				Status:    "confirmed",
			},
			{
				ID:        2,
				StartTime: time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC),
				Status:    "pending",
			},
		},
	}, nil, nil)

	res, err := svc.GetRoomAvailability(context.Background(), 2, "2026-07-01", "2026-07-10")
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

func TestGetRoomAvailabilityPropagatesRepoError(t *testing.T) {
	svc := NewBookingService(&availabilityRepoStub{err: errors.New("db down")}, nil, nil)

	_, err := svc.GetRoomAvailability(context.Background(), 2, "2026-07-01", "2026-07-10")
	if err == nil {
		t.Fatal("expected repo error")
	}
}

func TestParseAvailabilityWindowExplicitRange(t *testing.T) {
	from, to, err := parseAvailabilityWindow("2026-07-01", "2026-07-10")
	if err != nil {
		t.Fatalf("parseAvailabilityWindow: %v", err)
	}
	if from.Format("2006-01-02") != "2026-07-01" || to.Format("2006-01-02") != "2026-07-10" {
		t.Fatalf("unexpected window: %s to %s", from, to)
	}
}

func TestGenerateBookingIdempotencyKey(t *testing.T) {
	checkIn := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)

	got := generateBookingIdempotencyKey(2, 1, checkIn, checkOut)
	want := "booking:auto:2:1:2026-07-01:2026-07-06"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
