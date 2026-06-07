package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/chiamck/hotel-booking/internal/repository"
)

const maxCategorySearchLimit = 10

var (
	errInvalidHotelID   = errors.New("hotel_id must be a positive integer")
	errInvalidGuests    = errors.New("guests must be a positive integer")
	errInvalidLimit     = fmt.Errorf("limit must be between 1 and %d", maxCategorySearchLimit)
)

func parseRoomCategorySearchQuery(query roomCategorySearchQuery) (repository.RoomCategorySearchParams, error) {
	hotelID, err := strconv.Atoi(query.HotelID)
	if err != nil || hotelID < 1 {
		return repository.RoomCategorySearchParams{}, errInvalidHotelID
	}

	guests, err := strconv.Atoi(query.Guests)
	if err != nil || guests < 1 {
		return repository.RoomCategorySearchParams{}, errInvalidGuests
	}

	checkIn, err := time.Parse("2006-01-02", query.CheckIn)
	if err != nil {
		return repository.RoomCategorySearchParams{}, errInvalidCheckIn
	}

	checkOut, err := time.Parse("2006-01-02", query.CheckOut)
	if err != nil {
		return repository.RoomCategorySearchParams{}, errInvalidCheckOut
	}

	if !checkOut.After(checkIn) {
		return repository.RoomCategorySearchParams{}, errInvalidDateRange
	}

	page := 1
	if query.Page != "" {
		parsed, err := strconv.Atoi(query.Page)
		if err != nil || parsed < 1 {
			return repository.RoomCategorySearchParams{}, errInvalidPage
		}
		page = parsed
	}

	limit := maxCategorySearchLimit
	if query.Limit != "" {
		parsed, err := strconv.Atoi(query.Limit)
		if err != nil || parsed < 1 || parsed > maxCategorySearchLimit {
			return repository.RoomCategorySearchParams{}, errInvalidLimit
		}
		limit = parsed
	}

	return repository.RoomCategorySearchParams{
		HotelID:  hotelID,
		Guests:   guests,
		CheckIn:  checkIn,
		CheckOut: checkOut,
		Page:     page,
		Limit:    limit,
	}, nil
}
