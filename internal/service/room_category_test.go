package service_test

import (
	"testing"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/service"
)

type stubRoomCategoryRepository struct{}

func (stubRoomCategoryRepository) Search(repository.RoomCategorySearchParams) (models.RoomCategorySearchPage, error) {
	return models.RoomCategorySearchPage{}, nil
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

	_, err = svc.SearchCategories(service.RoomCategorySearchInput{
		HotelID:  1,
		Guests:   2,
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
		Page:     -1,
	})
	if err != service.ErrInvalidPage {
		t.Fatalf("expected ErrInvalidPage, got %v", err)
	}
}

type paginatedStubRoomCategoryRepository struct {
	categories []models.RoomCategorySearchResult
}

func (s paginatedStubRoomCategoryRepository) Search(params repository.RoomCategorySearchParams) (models.RoomCategorySearchPage, error) {
	offset := (params.Page - 1) * params.Limit
	if offset >= len(s.categories) {
		return models.RoomCategorySearchPage{
			Categories: []models.RoomCategorySearchResult{},
			Pagination: models.Pagination{
				Page:       params.Page,
				Limit:      params.Limit,
				Total:      len(s.categories),
				TotalPages: (len(s.categories) + params.Limit - 1) / params.Limit,
			},
		}, nil
	}

	end := offset + params.Limit
	if end > len(s.categories) {
		end = len(s.categories)
	}

	return models.RoomCategorySearchPage{
		Categories: s.categories[offset:end],
		Pagination: models.Pagination{
			Page:       params.Page,
			Limit:      params.Limit,
			Total:      len(s.categories),
			TotalPages: (len(s.categories) + params.Limit - 1) / params.Limit,
		},
	}, nil
}

func TestRoomCategoryServicePagination(t *testing.T) {
	svc := service.NewRoomCategoryService(paginatedStubRoomCategoryRepository{
		categories: []models.RoomCategorySearchResult{
			{ID: 1, Name: "Category 1"},
			{ID: 2, Name: "Category 2"},
			{ID: 3, Name: "Category 3"},
		},
	})

	page1, err := svc.SearchCategories(service.RoomCategorySearchInput{
		HotelID:  1,
		Guests:   2,
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
		Page:     1,
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("search page 1: %v", err)
	}
	if len(page1.Categories) != 2 || page1.Pagination.Total != 3 || page1.Pagination.TotalPages != 2 {
		t.Fatalf("unexpected page 1: %+v", page1)
	}

	page2, err := svc.SearchCategories(service.RoomCategorySearchInput{
		HotelID:  1,
		Guests:   2,
		CheckIn:  "2026-06-10",
		CheckOut: "2026-06-15",
		Page:     2,
		Limit:    2,
	})
	if err != nil {
		t.Fatalf("search page 2: %v", err)
	}
	if len(page2.Categories) != 1 || page2.Categories[0].ID != 3 {
		t.Fatalf("unexpected page 2: %+v", page2)
	}
}
