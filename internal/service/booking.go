package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/chiamck/hotel-booking/internal/idempotency"
	"github.com/chiamck/hotel-booking/internal/lock"
	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
)

const bookingLockExp = 30 * time.Second

const bookingIdempotencyCacheTTL = 7 * 24 * time.Hour

const (
	defaultAvailabilityHorizonDays = 180
	maxAvailabilityHorizonDays     = 731
)

var (
	ErrInvalidRoomID              = errors.New("room_id must be a positive integer")
	ErrInvalidCustomerID          = errors.New("customer_id must be a positive integer")
	ErrBookingLockNotAcquired     = errors.New("could not acquire booking lock")
	ErrIdempotencyCache           = errors.New("idempotency cache unavailable")
	ErrInvalidAvailabilityFrom    = errors.New("from must be a valid date in YYYY-MM-DD format")
	ErrInvalidAvailabilityTo      = errors.New("to must be a valid date in YYYY-MM-DD format")
	ErrInvalidAvailabilityRange   = errors.New("to must be on or after from")
	ErrAvailabilityWindowTooLarge = errors.New("date range must not exceed 731 days")
)

type CreateBookingInput struct {
	RoomID     int
	CustomerID int
	CheckIn    string
	CheckOut   string
}

type CreateBookingResult struct {
	Booking models.Booking `json:"booking"`
	Created bool           `json:"created"`
}

// RoomAvailabilityResponse lists calendar dates (UTC, YYYY-MM-DD) on which the room has an active stay night.
type RoomAvailabilityResponse struct {
	RoomID           int      `json:"room_id"`
	From             string   `json:"from"`
	To               string   `json:"to"`
	UnavailableDates []string `json:"unavailable_dates"`
}

type BookingService struct {
	repo repository.BookingRepository
	lock lock.DistributedLock
	idem idempotency.BookingStore
}

func NewBookingService(repo repository.BookingRepository, locker lock.DistributedLock, idem idempotency.BookingStore) *BookingService {
	return &BookingService{repo: repo, lock: locker, idem: idem}
}

func (s *BookingService) Create(ctx context.Context, input CreateBookingInput) (CreateBookingResult, error) {
	params, idemKey, err := parseCreateBookingInput(input)
	if err != nil {
		return CreateBookingResult{}, err
	}

	lockKey := bookingLockKey(params.RoomID)
	unlock, acquired, err := s.lock.TryLock(ctx, lockKey, bookingLockExp)
	if err != nil {
		return CreateBookingResult{}, err
	}

	if !acquired {
		return CreateBookingResult{}, ErrBookingLockNotAcquired
	}
	defer unlock()

	existing, err := s.idem.GetBooking(ctx, idemKey)
	if err != nil {
		return CreateBookingResult{}, fmt.Errorf("%w: %v", ErrIdempotencyCache, err)
	}

	if existing != nil {
		return CreateBookingResult{Booking: *existing, Created: false}, nil
	}

	booking, err := s.repo.Create(params)
	if err != nil {
		return CreateBookingResult{}, err
	}

	if err := s.idem.SetBooking(ctx, idemKey, booking, bookingIdempotencyCacheTTL); err != nil {
		return CreateBookingResult{}, fmt.Errorf("%w: %v", ErrIdempotencyCache, err)
	}

	return CreateBookingResult{Booking: booking, Created: true}, nil
}

// GetRoomAvailability loads pending/confirmed bookings for the room in the window and returns occupied nights as dates.
// from/to query strings use YYYY-MM-DD (UTC). Empty from defaults to today (UTC); empty to defaults to from + 180 days inclusive.
func (s *BookingService) GetRoomAvailability(_ context.Context, roomID int, fromQuery, toQuery string) (RoomAvailabilityResponse, error) {
	if roomID <= 0 {
		return RoomAvailabilityResponse{}, ErrInvalidRoomID
	}

	fromDate, toDate, err := parseAvailabilityWindow(fromQuery, toQuery)
	if err != nil {
		return RoomAvailabilityResponse{}, err
	}

	rangeEndExclusive := toDate.AddDate(0, 0, 1)
	bookings, err := s.repo.ListBlockingByRoomOverlap(roomID, fromDate, rangeEndExclusive)
	if err != nil {
		return RoomAvailabilityResponse{}, err
	}

	seen := make(map[string]struct{})
	fromKey := fromDate.Format("2006-01-02")
	toKey := toDate.Format("2006-01-02")
	for _, b := range bookings {
		for _, d := range occupiedNightsUTC(b.StartTime, b.EndTime) {
			if d >= fromKey && d <= toKey {
				seen[d] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for d := range seen {
		out = append(out, d)
	}
	sort.Strings(out)

	return RoomAvailabilityResponse{
		RoomID:           roomID,
		From:             fromKey,
		To:               toKey,
		UnavailableDates: out,
	}, nil
}

func parseAvailabilityWindow(fromQuery, toQuery string) (fromDate, toDate time.Time, err error) {
	if fromQuery == "" {
		now := time.Now().UTC()
		fromDate = civilMidnightUTC(now.Year(), int(now.Month()), now.Day())
	} else {
		fromDate, err = time.Parse("2006-01-02", fromQuery)
		if err != nil {
			err = ErrInvalidAvailabilityFrom
			return
		}
		fromDate = civilMidnightUTC(fromDate.Year(), int(fromDate.Month()), fromDate.Day())
	}

	if toQuery == "" {
		toDate = fromDate.AddDate(0, 0, defaultAvailabilityHorizonDays-1)
		return fromDate, toDate, nil
	}

	toDate, err = time.Parse("2006-01-02", toQuery)
	if err != nil {
		err = ErrInvalidAvailabilityTo
		return
	}
	toDate = civilMidnightUTC(toDate.Year(), int(toDate.Month()), toDate.Day())

	if toDate.Before(fromDate) {
		err = ErrInvalidAvailabilityRange
		return
	}

	days := int(toDate.Sub(fromDate).Hours()/24) + 1
	if days > maxAvailabilityHorizonDays {
		err = ErrAvailabilityWindowTooLarge
		return
	}

	return fromDate, toDate, nil
}

func civilMidnightUTC(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// occupiedNightsUTC returns each calendar night in [check-in date, check-out date) in UTC (matches booking half-open range).
func occupiedNightsUTC(start, end time.Time) []string {
	loc := time.UTC
	s := start.In(loc)
	e := end.In(loc)
	startDay := civilMidnightUTC(s.Year(), int(s.Month()), s.Day())
	endDay := civilMidnightUTC(e.Year(), int(e.Month()), e.Day())

	var out []string
	for d := startDay; d.Before(endDay); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format("2006-01-02"))
	}
	return out
}

func generateBookingIdempotencyKey(roomID, customerID int, checkIn, checkOut time.Time) string {
	return fmt.Sprintf(
		"booking:auto:%d:%d:%s:%s",
		roomID,
		customerID,
		checkIn.UTC().Format("2006-01-02"),
		checkOut.UTC().Format("2006-01-02"),
	)
}

func parseCreateBookingInput(input CreateBookingInput) (repository.CreateBookingParams, string, error) {
	if input.RoomID <= 0 {
		return repository.CreateBookingParams{}, "", ErrInvalidRoomID
	}
	if input.CustomerID <= 0 {
		return repository.CreateBookingParams{}, "", ErrInvalidCustomerID
	}

	checkIn, err := time.Parse("2006-01-02", input.CheckIn)
	if err != nil {
		return repository.CreateBookingParams{}, "", ErrInvalidCheckIn
	}

	checkOut, err := time.Parse("2006-01-02", input.CheckOut)
	if err != nil {
		return repository.CreateBookingParams{}, "", ErrInvalidCheckOut
	}

	if !checkOut.After(checkIn) {
		return repository.CreateBookingParams{}, "", ErrInvalidDateRange
	}

	idemKey := generateBookingIdempotencyKey(input.RoomID, input.CustomerID, checkIn, checkOut)

	return repository.CreateBookingParams{
		RoomID:     input.RoomID,
		CustomerID: input.CustomerID,
		CheckIn:    checkIn,
		CheckOut:   checkOut,
	}, idemKey, nil
}

func bookingLockKey(roomID int) string {
	return "booking:lock:room:" + strconv.Itoa(roomID)
}

func BookingErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidRoomID):
		return "room_id must be a positive integer"
	case errors.Is(err, ErrInvalidCustomerID):
		return "customer_id must be a positive integer"
	case errors.Is(err, ErrInvalidCheckIn):
		return "check_in must be a valid date in YYYY-MM-DD format"
	case errors.Is(err, ErrInvalidCheckOut):
		return "check_out must be a valid date in YYYY-MM-DD format"
	case errors.Is(err, ErrInvalidDateRange):
		return "check_out must be after check_in"
	case errors.Is(err, repository.ErrRoomNotFound):
		return "room not found"
	case errors.Is(err, repository.ErrRoomNotAvailable):
		return "room is not available"
	case errors.Is(err, repository.ErrBookingOverlap):
		return "room is already booked for the selected dates"
	case errors.Is(err, ErrBookingLockNotAcquired):
		return "another booking is in progress for this room, please retry"
	case errors.Is(err, ErrIdempotencyCache):
		return "idempotency cache unavailable"
	default:
		return "invalid request"
	}
}

func IsBookingValidationError(err error) bool {
	switch {
	case errors.Is(err, ErrInvalidRoomID),
		errors.Is(err, ErrInvalidCustomerID),
		errors.Is(err, ErrInvalidCheckIn),
		errors.Is(err, ErrInvalidCheckOut),
		errors.Is(err, ErrInvalidDateRange):
		return true
	default:
		return false
	}
}

func IsBookingConflictError(err error) bool {
	switch {
	case errors.Is(err, repository.ErrRoomNotAvailable),
		errors.Is(err, repository.ErrBookingOverlap),
		errors.Is(err, ErrBookingLockNotAcquired):
		return true
	default:
		return false
	}
}

func IsBookingNotFoundError(err error) bool {
	return errors.Is(err, repository.ErrRoomNotFound)
}

func IsIdempotencyCacheError(err error) bool {
	return errors.Is(err, ErrIdempotencyCache)
}

func AvailabilityErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidRoomID):
		return "room id must be a positive integer"
	case errors.Is(err, ErrInvalidAvailabilityFrom):
		return "from must be a valid date in YYYY-MM-DD format"
	case errors.Is(err, ErrInvalidAvailabilityTo):
		return "to must be a valid date in YYYY-MM-DD format"
	case errors.Is(err, ErrInvalidAvailabilityRange):
		return "to must be on or after from"
	case errors.Is(err, ErrAvailabilityWindowTooLarge):
		return "date range must not exceed 731 days"
	default:
		return "invalid request"
	}
}

func IsAvailabilityValidationError(err error) bool {
	switch {
	case errors.Is(err, ErrInvalidRoomID),
		errors.Is(err, ErrInvalidAvailabilityFrom),
		errors.Is(err, ErrInvalidAvailabilityTo),
		errors.Is(err, ErrInvalidAvailabilityRange),
		errors.Is(err, ErrAvailabilityWindowTooLarge):
		return true
	default:
		return false
	}
}
