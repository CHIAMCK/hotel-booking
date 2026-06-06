package repository

import (
	"database/sql"
	"time"

	"github.com/chiamck/hotel-booking/internal/models"
)

type RoomCategorySearchParams struct {
	HotelID  int
	Guests   int
	CheckIn  time.Time
	CheckOut time.Time
	Limit    int
}

type RoomCategoryRepository interface {
	Search(params RoomCategorySearchParams) ([]models.RoomCategorySearchResult, error)
}

type roomCategoryRepository struct {
	db *sql.DB
}

func NewRoomCategoryRepository(db *sql.DB) RoomCategoryRepository {
	return &roomCategoryRepository{db: db}
}

func (r *roomCategoryRepository) Search(params RoomCategorySearchParams) ([]models.RoomCategorySearchResult, error) {
	rows, err := r.db.Query(`
		SELECT
			rc.id,
			rc.name,
			rc.max_person,
			rc.base_price,
			COUNT(r.id) AS available_count
		FROM room_categories rc
		JOIN rooms r
			ON r.category_id = rc.id
		   AND r.hotel_id = rc.hotel_id
		   AND r.status = 'available'
		   AND NOT EXISTS (
			   SELECT 1
			   FROM bookings b
			   WHERE b.room_id = r.id
				 AND b.status IN ('pending', 'confirmed')
				 AND b.start_time < $4
				 AND b.end_time > $3
		   )
		WHERE rc.hotel_id = $1
		  AND rc.max_person >= $2
		GROUP BY rc.id, rc.name, rc.max_person, rc.base_price
		HAVING COUNT(r.id) > 0
		ORDER BY rc.base_price ASC, rc.name ASC
		LIMIT $5`,
		params.HotelID,
		params.Guests,
		params.CheckIn,
		params.CheckOut,
		params.Limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.RoomCategorySearchResult
	for rows.Next() {
		var result models.RoomCategorySearchResult
		if err := rows.Scan(
			&result.ID,
			&result.Name,
			&result.MaxPerson,
			&result.BasePrice,
			&result.AvailableCount,
		); err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
