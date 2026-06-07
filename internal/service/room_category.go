package service

import (
	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
)

type RoomCategoryService struct {
	repo repository.RoomCategoryRepository
}

func NewRoomCategoryService(repo repository.RoomCategoryRepository) *RoomCategoryService {
	return &RoomCategoryService{repo: repo}
}

func (s *RoomCategoryService) SearchCategories(params repository.RoomCategorySearchParams) (models.RoomCategorySearchPage, error) {
	return s.repo.Search(params)
}
