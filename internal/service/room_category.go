package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
)

const maxCategorySearchLimit = 10

var (
	ErrInvalidHotelID   = errors.New("hotel_id must be a positive integer")
	ErrInvalidGuests    = errors.New("guests must be a positive integer")
	ErrInvalidCheckIn   = errors.New("check_in must be a valid date in YYYY-MM-DD format")
	ErrInvalidCheckOut  = errors.New("check_out must be a valid date in YYYY-MM-DD format")
	ErrInvalidDateRange = errors.New("check_out must be after check_in")
	ErrInvalidLimit     = errors.New("limit must be between 1 and 10")
	ErrInvalidPage      = errors.New("page must be a positive integer")
)

type RoomCategorySearchInput struct {
	HotelID  int
	Guests   int
	CheckIn  string
	CheckOut string
	Page     int
	Limit    int
}

type RoomCategoryService struct {
	repo repository.RoomCategoryRepository
}

func NewRoomCategoryService(repo repository.RoomCategoryRepository) *RoomCategoryService {
	return &RoomCategoryService{repo: repo}
}

func (s *RoomCategoryService) SearchCategories(input RoomCategorySearchInput) (models.RoomCategorySearchPage, error) {
	params, err := parseSearchInput(input)
	if err != nil {
		return models.RoomCategorySearchPage{}, err
	}

	return s.repo.Search(params)
}

func parseSearchInput(input RoomCategorySearchInput) (repository.RoomCategorySearchParams, error) {
	if input.HotelID <= 0 {
		return repository.RoomCategorySearchParams{}, ErrInvalidHotelID
	}
	if input.Guests <= 0 {
		return repository.RoomCategorySearchParams{}, ErrInvalidGuests
	}

	checkIn, err := time.Parse("2006-01-02", input.CheckIn)
	if err != nil {
		return repository.RoomCategorySearchParams{}, ErrInvalidCheckIn
	}

	checkOut, err := time.Parse("2006-01-02", input.CheckOut)
	if err != nil {
		return repository.RoomCategorySearchParams{}, ErrInvalidCheckOut
	}

	if !checkOut.After(checkIn) {
		return repository.RoomCategorySearchParams{}, ErrInvalidDateRange
	}

	page := input.Page
	if page == 0 {
		page = 1
	}
	if page < 1 {
		return repository.RoomCategorySearchParams{}, ErrInvalidPage
	}

	limit := input.Limit
	if limit == 0 {
		limit = maxCategorySearchLimit
	}
	if limit < 1 || limit > maxCategorySearchLimit {
		return repository.RoomCategorySearchParams{}, ErrInvalidLimit
	}

	return repository.RoomCategorySearchParams{
		HotelID:  input.HotelID,
		Guests:   input.Guests,
		CheckIn:  checkIn,
		CheckOut: checkOut,
		Page:     page,
		Limit:    limit,
	}, nil
}

func ValidationErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrInvalidHotelID):
		return "hotel_id must be a positive integer"
	case errors.Is(err, ErrInvalidGuests):
		return "guests must be a positive integer"
	case errors.Is(err, ErrInvalidCheckIn):
		return "check_in must be a valid date in YYYY-MM-DD format"
	case errors.Is(err, ErrInvalidCheckOut):
		return "check_out must be a valid date in YYYY-MM-DD format"
	case errors.Is(err, ErrInvalidDateRange):
		return "check_out must be after check_in"
	case errors.Is(err, ErrInvalidLimit):
		return fmt.Sprintf("limit must be between 1 and %d", maxCategorySearchLimit)
	case errors.Is(err, ErrInvalidPage):
		return "page must be a positive integer"
	default:
		return "invalid request"
	}
}
