package repository

import (
	"database/sql"
	"errors"

	"github.com/chiamck/hotel-booking/internal/models"
)

type RoomRepository interface {
	List() ([]models.Room, error)
	Exists(id int) (bool, error)
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

func (r *roomRepository) Exists(id int) (bool, error) {
	var one int
	err := r.db.QueryRow(`SELECT 1 FROM rooms WHERE id = $1`, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
