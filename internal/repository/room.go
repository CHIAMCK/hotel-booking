package repository

import (
	"database/sql"

	"github.com/chiamck/hotel-booking/internal/models"
)

type RoomRepository interface {
	List() ([]models.Room, error)
}

type roomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) List() ([]models.Room, error) {
	rows, err := r.db.Query(`
		SELECT id, hotel_id, category_id, number, status
		FROM rooms
		ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []models.Room
	for rows.Next() {
		var room models.Room
		if err := rows.Scan(&room.ID, &room.HotelID, &room.CategoryID, &room.Number, &room.Status); err != nil {
			return nil, err
		}
		rooms = append(rooms, room)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return rooms, nil
}
