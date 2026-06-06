package repository

import (
	"database/sql"
	"errors"
)

type RoomRepository interface {
	Exists(id int) (bool, error)
}

type roomRepository struct {
	db *sql.DB
}

func NewRoomRepository(db *sql.DB) RoomRepository {
	return &roomRepository{db: db}
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
