package service

import (
	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/chiamck/hotel-booking/internal/repository"
)

type RoomService struct {
	repo repository.RoomRepository
}

func NewRoomService(repo repository.RoomRepository) *RoomService {
	return &RoomService{repo: repo}
}

func (s *RoomService) ListRooms() ([]models.Room, error) {
	return s.repo.List()
}

func (s *RoomService) RoomExists(id int) (bool, error) {
	if id <= 0 {
		return false, nil
	}
	return s.repo.Exists(id)
}
