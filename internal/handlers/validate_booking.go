package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/timeutil"
)

const (
	defaultAvailabilityHorizonDays = 180
	maxAvailabilityHorizonDays     = 731
	defaultBookingListLimit        = 20
	maxBookingListLimit            = 100
)

var (
	errInvalidRoomID              = errors.New("room_id must be a positive integer")
	errInvalidCustomerID          = errors.New("customer_id must be a positive integer")
	errInvalidBasePrice           = errors.New("base_price must be zero or greater")
	errInvalidCheckIn             = errors.New("check_in must be a valid date in YYYY-MM-DD format")
	errInvalidCheckOut            = errors.New("check_out must be a valid date in YYYY-MM-DD format")
	errInvalidDateRange           = errors.New("check_out must be after check_in")
	errInvalidUserID              = errors.New("user_id must be a positive integer")
	errInvalidPage                = errors.New("page must be a positive integer")
	errInvalidBookingListLimit    = fmt.Errorf("limit must be between 1 and %d", maxBookingListLimit)
	errInvalidAvailabilityFrom    = errors.New("from must be a valid date in YYYY-MM-DD format")
	errInvalidAvailabilityTo      = errors.New("to must be a valid date in YYYY-MM-DD format")
	errInvalidAvailabilityRange   = errors.New("to must be on or after from")
	errAvailabilityWindowTooLarge = errors.New("date range must not exceed 731 days")
)

func parseCreateBookingRequest(req createBookingRequest) (repository.CreateBookingParams, error) {
	if req.RoomID <= 0 {
		return repository.CreateBookingParams{}, errInvalidRoomID
	}

	if req.CustomerID <= 0 {
		return repository.CreateBookingParams{}, errInvalidCustomerID
	}

	if req.BasePrice < 0 {
		return repository.CreateBookingParams{}, errInvalidBasePrice
	}

	checkIn, err := time.Parse("2006-01-02", req.CheckIn)
	if err != nil {
		return repository.CreateBookingParams{}, errInvalidCheckIn
	}

	checkOut, err := time.Parse("2006-01-02", req.CheckOut)
	if err != nil {
		return repository.CreateBookingParams{}, errInvalidCheckOut
	}

	if !checkOut.After(checkIn) {
		return repository.CreateBookingParams{}, errInvalidDateRange
	}

	nights := int(checkOut.Sub(checkIn).Hours() / 24)
	if nights < 1 {
		return repository.CreateBookingParams{}, errInvalidDateRange
	}

	return repository.CreateBookingParams{
		RoomID:        req.RoomID,
		CustomerID:    req.CustomerID,
		CheckIn:       checkIn,
		CheckOut:      checkOut,
		Nights:        nights,
		TotalAmount:   req.BasePrice * float64(nights),
		PricePerNight: req.BasePrice,
	}, nil
}

func parseListBookingsQuery(userID, page, limit string) (repository.ListBookingsParams, error) {
	if userID == "" {
		return repository.ListBookingsParams{}, errInvalidUserID
	}

	parsedUserID, err := strconv.Atoi(userID)
	if err != nil || parsedUserID < 1 {
		return repository.ListBookingsParams{}, errInvalidUserID
	}

	parsedPage := 1
	if page != "" {
		parsed, err := strconv.Atoi(page)
		if err != nil || parsed < 1 {
			return repository.ListBookingsParams{}, errInvalidPage
		}
		parsedPage = parsed
	}

	parsedLimit := defaultBookingListLimit
	if limit != "" {
		parsed, err := strconv.Atoi(limit)
		if err != nil || parsed < 1 || parsed > maxBookingListLimit {
			return repository.ListBookingsParams{}, errInvalidBookingListLimit
		}
		parsedLimit = parsed
	}

	return repository.ListBookingsParams{
		CustomerID: parsedUserID,
		Page:       parsedPage,
		Limit:      parsedLimit,
	}, nil
}

func parseAvailabilityQuery(fromQuery, toQuery string) (fromDate, toDate time.Time, err error) {
	if fromQuery == "" {
		now := time.Now().UTC()
		fromDate = timeutil.MidnightUTC(now.Year(), int(now.Month()), now.Day())
	} else {
		fromDate, err = time.Parse("2006-01-02", fromQuery)
		if err != nil {
			return time.Time{}, time.Time{}, errInvalidAvailabilityFrom
		}
		fromDate = timeutil.MidnightUTC(fromDate.Year(), int(fromDate.Month()), fromDate.Day())
	}

	if toQuery == "" {
		toDate = fromDate.AddDate(0, 0, defaultAvailabilityHorizonDays-1)
		return fromDate, toDate, nil
	}

	toDate, err = time.Parse("2006-01-02", toQuery)
	if err != nil {
		return time.Time{}, time.Time{}, errInvalidAvailabilityTo
	}

	toDate = timeutil.MidnightUTC(toDate.Year(), int(toDate.Month()), toDate.Day())

	if toDate.Before(fromDate) {
		return time.Time{}, time.Time{}, errInvalidAvailabilityRange
	}

	days := int(toDate.Sub(fromDate).Hours()/24) + 1
	if days > maxAvailabilityHorizonDays {
		return time.Time{}, time.Time{}, errAvailabilityWindowTooLarge
	}

	return fromDate, toDate, nil
}
