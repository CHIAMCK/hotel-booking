package handlers

import (
	"errors"
	"testing"
	"time"
)

func TestParseCreateBookingRequestValid(t *testing.T) {
	params, err := parseCreateBookingRequest(createBookingRequest{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
		BasePrice:  150,
	})
	if err != nil {
		t.Fatalf("parseCreateBookingRequest: %v", err)
	}
	if params.RoomID != 2 || params.CustomerID != 1 {
		t.Fatalf("unexpected params: %+v", params)
	}
	if !params.CheckOut.After(params.CheckIn) {
		t.Fatal("expected check-out after check-in")
	}
	if params.Nights != 5 || params.TotalAmount != 750 || params.PricePerNight != 150 {
		t.Fatalf("unexpected pricing: %+v", params)
	}
}

func TestParseCreateBookingRequestRejectsInvalidCustomerID(t *testing.T) {
	_, err := parseCreateBookingRequest(createBookingRequest{
		RoomID:     2,
		CustomerID: 0,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-06",
		BasePrice:  150,
	})
	if !errors.Is(err, errInvalidCustomerID) {
		t.Fatalf("expected errInvalidCustomerID, got %v", err)
	}
}

func TestParseCreateBookingRequestRejectsSameDayStay(t *testing.T) {
	_, err := parseCreateBookingRequest(createBookingRequest{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "2026-07-01",
		BasePrice:  150,
	})
	if !errors.Is(err, errInvalidDateRange) {
		t.Fatalf("expected errInvalidDateRange, got %v", err)
	}
}

func TestParseCreateBookingRequestRejectsInvalidCheckOut(t *testing.T) {
	_, err := parseCreateBookingRequest(createBookingRequest{
		RoomID:     2,
		CustomerID: 1,
		CheckIn:    "2026-07-01",
		CheckOut:   "07-06-2026",
		BasePrice:  150,
	})
	if !errors.Is(err, errInvalidCheckOut) {
		t.Fatalf("expected errInvalidCheckOut, got %v", err)
	}
}

func TestParseListBookingsQueryDefaults(t *testing.T) {
	params, err := parseListBookingsQuery("1", "", "")
	if err != nil {
		t.Fatalf("parseListBookingsQuery: %v", err)
	}
	if params.CustomerID != 1 || params.Page != 1 || params.Limit != defaultBookingListLimit {
		t.Fatalf("unexpected params: %+v", params)
	}
}

func TestParseListBookingsQueryRejectsInvalidLimit(t *testing.T) {
	_, err := parseListBookingsQuery("1", "1", "101")
	if !errors.Is(err, errInvalidBookingListLimit) {
		t.Fatalf("expected errInvalidBookingListLimit, got %v", err)
	}
}

func TestParseAvailabilityQueryExplicitRange(t *testing.T) {
	from, to, err := parseAvailabilityQuery("2026-07-01", "2026-07-10")
	if err != nil {
		t.Fatalf("parseAvailabilityQuery: %v", err)
	}
	if from.Format("2006-01-02") != "2026-07-01" || to.Format("2006-01-02") != "2026-07-10" {
		t.Fatalf("unexpected window: %s to %s", from, to)
	}
}

func TestParseAvailabilityQueryRejectsInvalidRange(t *testing.T) {
	_, _, err := parseAvailabilityQuery("2026-08-10", "2026-08-01")
	if !errors.Is(err, errInvalidAvailabilityRange) {
		t.Fatalf("expected errInvalidAvailabilityRange, got %v", err)
	}
}

func TestParseAvailabilityQueryRejectsInvalidFromDate(t *testing.T) {
	_, _, err := parseAvailabilityQuery("not-a-date", "2026-07-10")
	if !errors.Is(err, errInvalidAvailabilityFrom) {
		t.Fatalf("expected errInvalidAvailabilityFrom, got %v", err)
	}
}

func TestParseAvailabilityQueryRejectsWindowTooLarge(t *testing.T) {
	_, _, err := parseAvailabilityQuery("2026-01-01", "2028-01-02")
	if !errors.Is(err, errAvailabilityWindowTooLarge) {
		t.Fatalf("expected errAvailabilityWindowTooLarge, got %v", err)
	}
}

func TestParseAvailabilityQueryDefaultsTo(t *testing.T) {
	from, to, err := parseAvailabilityQuery("2026-07-01", "")
	if err != nil {
		t.Fatalf("parseAvailabilityQuery: %v", err)
	}
	wantTo := from.AddDate(0, 0, defaultAvailabilityHorizonDays-1)
	if !to.Equal(wantTo) {
		t.Fatalf("expected to=%v, got %v", wantTo, to)
	}
}

func TestParseAvailabilityQueryParsesUTC(t *testing.T) {
	from, to, err := parseAvailabilityQuery("2026-07-01", "2026-07-10")
	if err != nil {
		t.Fatalf("parseAvailabilityQuery: %v", err)
	}
	wantFrom := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	if !from.Equal(wantFrom) || !to.Equal(wantTo) {
		t.Fatalf("unexpected dates: from=%v to=%v", from, to)
	}
}
