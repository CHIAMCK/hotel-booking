package repository

import "github.com/chiamck/hotel-booking/internal/models"

type RoomRepository interface {
	List() ([]models.Room, error)
}

type InMemoryRoomRepository struct {
	rooms []models.Room
}

func NewInMemoryRoomRepository() *InMemoryRoomRepository {
	return &InMemoryRoomRepository{
		rooms: []models.Room{
			{
				ID:          1,
				Name:        "Deluxe King",
				Description: "A spacious room with a king bed.",
				Price:       180,
			},
			{
				ID:          2,
				Name:        "Twin Suite",
				Description: "A comfortable suite with two twin beds.",
				Price:       220,
			},
		},
	}
}

func (r *InMemoryRoomRepository) List() ([]models.Room, error) {
	return r.rooms, nil
}
