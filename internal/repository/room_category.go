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
	Page     int
	Limit    int
}

type RoomCategoryRepository interface {
	Search(params RoomCategorySearchParams) (models.RoomCategorySearchPage, error)
}

type roomCategoryRepository struct {
	db *sql.DB
}

func NewRoomCategoryRepository(db *sql.DB) RoomCategoryRepository {
	return &roomCategoryRepository{db: db}
}

const roomCategorySearchBaseQuery = `
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
	HAVING COUNT(r.id) > 0`

func (r *roomCategoryRepository) Search(params RoomCategorySearchParams) (models.RoomCategorySearchPage, error) {
	offset := (params.Page - 1) * params.Limit
	dataQuery := `
		SELECT
			rc.id,
			rc.name,
			rc.max_person,
			rc.base_price,
			COUNT(r.id) AS available_count` + roomCategorySearchBaseQuery + `
		ORDER BY rc.base_price ASC, rc.name ASC
		LIMIT $5 OFFSET $6`

	rows, err := r.db.Query(
		dataQuery,
		params.HotelID,
		params.Guests,
		params.CheckIn,
		params.CheckOut,
		params.Limit,
		offset,
	)

	if err != nil {
		return models.RoomCategorySearchPage{}, err
	}

	defer rows.Close()

	var categories []models.RoomCategorySearchResult
	for rows.Next() {
		var result models.RoomCategorySearchResult
		if err := rows.Scan(
			&result.ID,
			&result.Name,
			&result.MaxPerson,
			&result.BasePrice,
			&result.AvailableCount,
		); err != nil {
			return models.RoomCategorySearchPage{}, err
		}

		categories = append(categories, result)
	}

	if err := rows.Err(); err != nil {
		return models.RoomCategorySearchPage{}, err
	}

	if categories == nil {
		categories = []models.RoomCategorySearchResult{}
	}

	return models.RoomCategorySearchPage{
		Categories: categories,
		Pagination: models.Pagination{
			Page:  params.Page,
			Limit: params.Limit,
		},
	}, nil
}
