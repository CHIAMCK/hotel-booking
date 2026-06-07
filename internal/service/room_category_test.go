package service_test

import (
	"testing"
	"time"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/service"
)

type stubRoomCategoryRepository struct{}

func (stubRoomCategoryRepository) Search(repository.RoomCategorySearchParams) (models.RoomCategorySearchPage, error) {
	return models.RoomCategorySearchPage{}, nil
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
				Page:  params.Page,
				Limit: params.Limit,
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
			Page:  params.Page,
			Limit: params.Limit,
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

	checkIn := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)
	checkOut := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	page1, err := svc.SearchCategories(repository.RoomCategorySearchParams{
		HotelID: 1, Guests: 2, CheckIn: checkIn, CheckOut: checkOut, Page: 1, Limit: 2,
	})
	if err != nil {
		t.Fatalf("search page 1: %v", err)
	}
	if len(page1.Categories) != 2 || page1.Pagination.Page != 1 || page1.Pagination.Limit != 2 {
		t.Fatalf("unexpected page 1: %+v", page1)
	}

	page2, err := svc.SearchCategories(repository.RoomCategorySearchParams{
		HotelID: 1, Guests: 2, CheckIn: checkIn, CheckOut: checkOut, Page: 2, Limit: 2,
	})
	if err != nil {
		t.Fatalf("search page 2: %v", err)
	}
	if len(page2.Categories) != 1 || page2.Categories[0].ID != 3 {
		t.Fatalf("unexpected page 2: %+v", page2)
	}
}
