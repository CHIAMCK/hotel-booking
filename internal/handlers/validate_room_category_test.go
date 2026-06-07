package handlers

import (
	"errors"
	"testing"
	"time"
)

func TestParseRoomCategorySearchQueryDefaults(t *testing.T) {
	params, err := parseRoomCategorySearchQuery(roomCategorySearchQuery{
		HotelID:  "1",
		Guests:   "2",
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
	})
	if err != nil {
		t.Fatalf("parseRoomCategorySearchQuery: %v", err)
	}
	if params.Page != 1 {
		t.Fatalf("expected default page 1, got %d", params.Page)
	}
	if params.Limit != maxCategorySearchLimit {
		t.Fatalf("expected default limit %d, got %d", maxCategorySearchLimit, params.Limit)
	}
	if !params.CheckOut.After(params.CheckIn) {
		t.Fatal("expected parsed check-out after check-in")
	}
}

func TestParseRoomCategorySearchQueryRejectsInvalidHotelID(t *testing.T) {
	_, err := parseRoomCategorySearchQuery(roomCategorySearchQuery{
		HotelID:  "0",
		Guests:   "2",
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
	})
	if !errors.Is(err, errInvalidHotelID) {
		t.Fatalf("expected errInvalidHotelID, got %v", err)
	}
}

func TestParseRoomCategorySearchQueryRejectsInvalidGuests(t *testing.T) {
	_, err := parseRoomCategorySearchQuery(roomCategorySearchQuery{
		HotelID:  "1",
		Guests:   "0",
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
	})
	if !errors.Is(err, errInvalidGuests) {
		t.Fatalf("expected errInvalidGuests, got %v", err)
	}
}

func TestParseRoomCategorySearchQueryRejectsInvalidCheckOut(t *testing.T) {
	_, err := parseRoomCategorySearchQuery(roomCategorySearchQuery{
		HotelID:  "1",
		Guests:   "2",
		CheckIn:  "2026-06-10",
		CheckOut: "June 15",
	})
	if !errors.Is(err, errInvalidCheckOut) {
		t.Fatalf("expected errInvalidCheckOut, got %v", err)
	}
}

func TestParseRoomCategorySearchQueryRejectsInvalidDateRange(t *testing.T) {
	_, err := parseRoomCategorySearchQuery(roomCategorySearchQuery{
		HotelID:  "1",
		Guests:   "2",
		CheckIn:  "2026-06-15",
		CheckOut: "2026-06-10",
	})
	if !errors.Is(err, errInvalidDateRange) {
		t.Fatalf("expected errInvalidDateRange, got %v", err)
	}
}

func TestParseRoomCategorySearchQueryRejectsInvalidLimit(t *testing.T) {
	_, err := parseRoomCategorySearchQuery(roomCategorySearchQuery{
		HotelID:  "1",
		Guests:   "2",
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
		Limit:    "11",
	})
	if !errors.Is(err, errInvalidLimit) {
		t.Fatalf("expected errInvalidLimit, got %v", err)
	}
}

func TestParseRoomCategorySearchQueryRejectsInvalidPage(t *testing.T) {
	_, err := parseRoomCategorySearchQuery(roomCategorySearchQuery{
		HotelID:  "1",
		Guests:   "2",
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
		Page:     "-1",
	})
	if !errors.Is(err, errInvalidPage) {
		t.Fatalf("expected errInvalidPage, got %v", err)
	}
}

func TestParseRoomCategorySearchQueryParsesDatesAsUTCMidnight(t *testing.T) {
	params, err := parseRoomCategorySearchQuery(roomCategorySearchQuery{
		HotelID:  "1",
		Guests:   "2",
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
		Page:     "2",
		Limit:    "5",
	})
	if err != nil {
		t.Fatalf("parseRoomCategorySearchQuery: %v", err)
	}
	wantIn := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	wantOut := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	if !params.CheckIn.Equal(wantIn) || !params.CheckOut.Equal(wantOut) {
		t.Fatalf("unexpected dates: in=%v out=%v", params.CheckIn, params.CheckOut)
	}
	if params.Page != 2 || params.Limit != 5 {
		t.Fatalf("unexpected pagination: page=%d limit=%d", params.Page, params.Limit)
	}
}
