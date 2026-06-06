package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/chiamck/hotel-booking/internal/lock"
	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
)

const bookingLockTTL = 30 * time.Second

var (
	ErrIdempotencyKeyRequired = errors.New("idempotency-key header is required")
	ErrInvalidRoomID          = errors.New("room_id must be a positive integer")
	ErrInvalidCustomerID      = errors.New("customer_id must be a positive integer")
	ErrBookingLockNotAcquired = errors.New("could not acquire booking lock")
)

type CreateBookingInput struct {
	RoomID         int
	CustomerID     int
	CheckIn        string
	CheckOut       string
	IdempotencyKey string
}

type CreateBookingResult struct {
	Booking models.Booking `json:"booking"`
	Created bool           `json:"created"`
}

type BookingService struct {
	repo repository.BookingRepository
	lock lock.DistributedLock
}

func NewBookingService(repo repository.BookingRepository, locker lock.DistributedLock) *BookingService {
	return &BookingService{repo: repo, lock: locker}
}

func (s *BookingService) Create(ctx context.Context, input CreateBookingInput) (CreateBookingResult, error) {
	params, err := parseCreateBookingInput(input)
	if err != nil {
		return CreateBookingResult{}, err
	}

	if existing, err := s.repo.FindByIdempotencyKey(params.IdempotencyKey); err != nil {
		return CreateBookingResult{}, err
	} else if existing != nil {
		return CreateBookingResult{Booking: *existing, Created: false}, nil
	}

	lockKey := bookingLockKey(params.RoomID)
	unlock, acquired, err := s.lock.TryLock(ctx, lockKey, bookingLockTTL)
	if err != nil {
		return CreateBookingResult{}, err
	}
	if !acquired {
		return CreateBookingResult{}, ErrBookingLockNotAcquired
	}
	defer unlock()

	if existing, err := s.repo.FindByIdempotencyKey(params.IdempotencyKey); err != nil {
		return CreateBookingResult{}, err
	} else if existing != nil {
		return CreateBookingResult{Booking: *existing, Created: false}, nil
	}

	booking, err := s.repo.Create(params)
	if errors.Is(err, repository.ErrIdempotencyConflict) {
		existing, findErr := s.repo.FindByIdempotencyKey(params.IdempotencyKey)
		if findErr != nil {
			return CreateBookingResult{}, findErr
		}
		if existing == nil {
			return CreateBookingResult{}, err
		}
		return CreateBookingResult{Booking: *existing, Created: false}, nil
	}
	if err != nil {
		return CreateBookingResult{}, err
	}

	return CreateBookingResult{Booking: booking, Created: true}, nil
}

func parseCreateBookingInput(input CreateBookingInput) (repository.CreateBookingParams, error) {
	if input.IdempotencyKey == "" {
		return repository.CreateBookingParams{}, ErrIdempotencyKeyRequired
	}
	if input.RoomID <= 0 {
		return repository.CreateBookingParams{}, ErrInvalidRoomID
	}
	if input.CustomerID <= 0 {
		return repository.CreateBookingParams{}, ErrInvalidCustomerID
	}

	checkIn, err := time.Parse("2006-01-02", input.CheckIn)
	if err != nil {
		return repository.CreateBookingParams{}, ErrInvalidCheckIn
	}

	checkOut, err := time.Parse("2006-01-02", input.CheckOut)
	if err != nil {
		return repository.CreateBookingParams{}, ErrInvalidCheckOut
	}

	if !checkOut.After(checkIn) {
		return repository.CreateBookingParams{}, ErrInvalidDateRange
	}

	return repository.CreateBookingParams{
		RoomID:         input.RoomID,
		CustomerID:     input.CustomerID,
		CheckIn:        checkIn,
		CheckOut:       checkOut,
		IdempotencyKey: input.IdempotencyKey,
	}, nil
}

func bookingLockKey(roomID int) string {
	return "booking:lock:room:" + strconv.Itoa(roomID)
}

func BookingErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrIdempotencyKeyRequired):
		return "Idempotency-Key header is required"
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
	default:
		return "invalid request"
	}
}

func IsBookingValidationError(err error) bool {
	switch {
	case errors.Is(err, ErrIdempotencyKeyRequired),
		errors.Is(err, ErrInvalidRoomID),
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
