package service

import (
	"testing"
	"time"
)

func TestParseSearchInputDefaults(t *testing.T) {
	params, err := parseSearchInput(RoomCategorySearchInput{
		HotelID:  1,
		Guests:   2,
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
	})
	if err != nil {
		t.Fatalf("parseSearchInput: %v", err)
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

func TestParseSearchInputRejectsInvalidHotelID(t *testing.T) {
	_, err := parseSearchInput(RoomCategorySearchInput{
		HotelID:  0,
		Guests:   2,
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
	})
	if err != ErrInvalidHotelID {
		t.Fatalf("expected ErrInvalidHotelID, got %v", err)
	}
}

func TestParseSearchInputRejectsInvalidGuests(t *testing.T) {
	_, err := parseSearchInput(RoomCategorySearchInput{
		HotelID:  1,
		Guests:   0,
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
	})
	if err != ErrInvalidGuests {
		t.Fatalf("expected ErrInvalidGuests, got %v", err)
	}
}

func TestParseSearchInputRejectsInvalidCheckOut(t *testing.T) {
	_, err := parseSearchInput(RoomCategorySearchInput{
		HotelID:  1,
		Guests:   2,
		CheckIn:  "2026-06-10",
		CheckOut: "June 15",
	})
	if err != ErrInvalidCheckOut {
		t.Fatalf("expected ErrInvalidCheckOut, got %v", err)
	}
}

func TestParseSearchInputParsesDatesAsUTCMidnight(t *testing.T) {
	params, err := parseSearchInput(RoomCategorySearchInput{
		HotelID:  1,
		Guests:   2,
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
		Page:     2,
		Limit:    5,
	})
	if err != nil {
		t.Fatalf("parseSearchInput: %v", err)
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

func TestValidationErrorMessage(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{ErrInvalidHotelID, "hotel_id must be a positive integer"},
		{ErrInvalidGuests, "guests must be a positive integer"},
		{ErrInvalidLimit, "limit must be between 1 and 10"},
		{ErrInvalidPage, "page must be a positive integer"},
	}

	for _, tc := range cases {
		if got := ValidationErrorMessage(tc.err); got != tc.want {
			t.Errorf("%v: got %q want %q", tc.err, got, tc.want)
		}
	}
}
