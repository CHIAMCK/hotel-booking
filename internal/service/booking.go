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
	"github.com/chiamck/hotel-booking/internal/timeutil"
)

const bookingLockExp = 30 * time.Second

const bookingIdempotencyCacheTTL = 24 * time.Hour

var (
	ErrBookingLockNotAcquired  = errors.New("could not acquire booking lock")
	ErrDuplicateBookingRequest = errors.New("duplicate booking request")
	ErrIdempotencyCache        = errors.New("idempotency cache unavailable")
)

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

func (s *BookingService) Create(ctx context.Context, params repository.CreateBookingParams) (models.Booking, error) {
	lockKey := bookingLockKey(params.RoomID)
	acquired, err := s.lock.TryLock(ctx, lockKey, bookingLockExp)
	if err != nil {
		return models.Booking{}, err
	}

	if !acquired {
		return models.Booking{}, ErrBookingLockNotAcquired
	}

	defer s.lock.Unlock(ctx, lockKey)
	idemKey := generateBookingIdempotencyKey(params.RoomID, params.CustomerID, params.CheckIn, params.CheckOut)
	used, err := s.idem.CheckIdempotent(ctx, idemKey)
	if err != nil {
		return models.Booking{}, fmt.Errorf("%w: %v", ErrIdempotencyCache, err)
	}

	if used {
		return models.Booking{}, ErrDuplicateBookingRequest
	}

	booking, err := s.repo.Create(params)
	if err != nil {
		return models.Booking{}, err
	}

	if err := s.idem.SetIdempotent(ctx, idemKey, bookingIdempotencyCacheTTL); err != nil {
		return models.Booking{}, fmt.Errorf("%w: %v", ErrIdempotencyCache, err)
	}

	return booking, nil
}

func (s *BookingService) List(_ context.Context, params repository.ListBookingsParams) (models.BookingListPage, error) {
	return s.repo.List(params)
}

func (s *BookingService) GetRoomAvailability(_ context.Context, roomID int, fromDate, toDate time.Time) (RoomAvailabilityResponse, error) {
	rangeEndExclusive := toDate.AddDate(0, 0, 1)
	bookings, err := s.repo.ListActiveBookingsOverlappingRange(roomID, fromDate, rangeEndExclusive)
	if err != nil {
		return RoomAvailabilityResponse{}, err
	}

	seen := make(map[string]struct{})
	fromKey := fromDate.Format("2006-01-02")
	toKey := toDate.Format("2006-01-02")
	for _, b := range bookings {
		for _, d := range stayNightDates(b.StartTime, b.EndTime) {
			if dateInInclusiveRange(d, fromKey, toKey) {
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

func dateInInclusiveRange(date, from, to string) bool {
	return date >= from && date <= to
}

func stayNightDates(start, end time.Time) []string {
	loc := time.UTC
	s := start.In(loc)
	e := end.In(loc)
	startDay := timeutil.MidnightUTC(s.Year(), int(s.Month()), s.Day())
	endDay := timeutil.MidnightUTC(e.Year(), int(e.Month()), e.Day())

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

func bookingLockKey(roomID int) string {
	return "booking:lock:room:" + strconv.Itoa(roomID)
}

func BookingErrorMessage(err error) string {
	switch {
	case errors.Is(err, repository.ErrBookingOverlap):
		return "room is already booked for the selected dates"
	case errors.Is(err, ErrBookingLockNotAcquired):
		return "another booking is in progress for this room, please retry"
	case errors.Is(err, ErrDuplicateBookingRequest):
		return "this booking request was already processed"
	case errors.Is(err, ErrIdempotencyCache):
		return "idempotency cache unavailable"
	default:
		return "invalid request"
	}
}

func IsBookingConflictError(err error) bool {
	switch {
	case errors.Is(err, repository.ErrBookingOverlap),
		errors.Is(err, ErrBookingLockNotAcquired),
		errors.Is(err, ErrDuplicateBookingRequest):
		return true
	default:
		return false
	}
}

func IsIdempotencyCacheError(err error) bool {
	return errors.Is(err, ErrIdempotencyCache)
}
