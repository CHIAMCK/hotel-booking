package service_test

import (
	"testing"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/service"
)

type stubRoomCategoryRepository struct{}

func (stubRoomCategoryRepository) Search(repository.RoomCategorySearchParams) ([]models.RoomCategorySearchResult, error) {
	return nil, nil
}

func TestRoomCategoryServiceValidation(t *testing.T) {
	svc := service.NewRoomCategoryService(stubRoomCategoryRepository{})

	_, err := svc.SearchCategories(service.RoomCategorySearchInput{
		HotelID:  1,
		Guests:   2,
		CheckIn:  "2026-06-15",
		CheckOut: "2026-06-10",
	})
	if err != service.ErrInvalidDateRange {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}

	_, err = svc.SearchCategories(service.RoomCategorySearchInput{
		HotelID:  1,
		Guests:   2,
		CheckIn:  "invalid",
		CheckOut: "2026-06-15",
	})
	if err != service.ErrInvalidCheckIn {
		t.Fatalf("expected ErrInvalidCheckIn, got %v", err)
	}

	_, err = svc.SearchCategories(service.RoomCategorySearchInput{
		HotelID:  1,
		Guests:   2,
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
		Limit:    11,
	})
	if err != service.ErrInvalidLimit {
		t.Fatalf("expected ErrInvalidLimit, got %v", err)
	}
}
