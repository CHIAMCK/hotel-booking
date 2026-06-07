package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/lib/pq"
)

var (
	ErrBookingNotFound = errors.New("booking not found")
	ErrBookingOverlap  = errors.New("room is already booked for the selected dates")
)

type CreateBookingParams struct {
	RoomID        int
	CustomerID    int
	CheckIn       time.Time
	CheckOut      time.Time
	Nights        int
	TotalAmount   float64
	PricePerNight float64
}

type ListBookingsParams struct {
	CustomerID int
	Page       int
	Limit      int
}

type BookingRepository interface {
	Create(params CreateBookingParams) (models.Booking, error)
	ListActiveBookingsOverlappingRange(roomID int, rangeStart, rangeEnd time.Time) ([]models.Booking, error)
	List(params ListBookingsParams) (models.BookingListPage, error)
}

type bookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository) Create(params CreateBookingParams) (models.Booking, error) {
	row := r.db.QueryRow(`
		INSERT INTO bookings (
			room_id, customer_id, start_time, end_time, status,
			total_amount, price_per_night
		) VALUES ($1, $2, $3, $4, 'confirmed', $5, $6)
		RETURNING id, room_id, customer_id, start_time, end_time, status,
		          total_amount, price_per_night`,
		params.RoomID,
		params.CustomerID,
		params.CheckIn,
		params.CheckOut,
		params.TotalAmount,
		params.PricePerNight,
	)

	booking, err := scanBooking(row)
	if err != nil {
		return models.Booking{}, mapBookingCreateError(err)
	}

	return booking, nil
}

func mapBookingCreateError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23P01" && pqErr.Constraint == "bookings_no_overlap" {
		return ErrBookingOverlap
	}
	return err
}

type bookingScanner interface {
	Scan(dest ...any) error
}

func scanBooking(row bookingScanner) (models.Booking, error) {
	var booking models.Booking
	err := row.Scan(
		&booking.ID,
		&booking.RoomID,
		&booking.CustomerID,
		&booking.StartTime,
		&booking.EndTime,
		&booking.Status,
		&booking.TotalAmount,
		&booking.PricePerNight,
	)
	return booking, err
}

func (r *bookingRepository) ListActiveBookingsOverlappingRange(roomID int, rangeStart, rangeEnd time.Time) ([]models.Booking, error) {
	if !rangeEnd.After(rangeStart) {
		return nil, nil
	}

	rows, err := r.db.Query(`
		SELECT id, room_id, customer_id, start_time, end_time, status,
		       total_amount, price_per_night
		FROM bookings
		WHERE room_id = $1
		  AND status IN ('pending', 'confirmed')
		  AND start_time < $3 AND end_time > $2
		ORDER BY start_time`,
		roomID, rangeStart, rangeEnd,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.Booking
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, booking)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *bookingRepository) List(params ListBookingsParams) (models.BookingListPage, error) {
	offset := (params.Page - 1) * params.Limit
	rows, err := r.db.Query(`
		SELECT id, room_id, customer_id, start_time, end_time, status,
		       total_amount, price_per_night
		FROM bookings
		WHERE customer_id = $1
		ORDER BY start_time DESC, id DESC
		LIMIT $2 OFFSET $3`,
		params.CustomerID, params.Limit, offset,
	)

	if err != nil {
		return models.BookingListPage{}, err
	}

	defer rows.Close()

	var bookings []models.Booking
	for rows.Next() {
		booking, err := scanBooking(rows)
		if err != nil {
			return models.BookingListPage{}, err
		}

		bookings = append(bookings, booking)
	}

	if err := rows.Err(); err != nil {
		return models.BookingListPage{}, err
	}

	if bookings == nil {
		bookings = []models.Booking{}
	}

	return models.BookingListPage{
		Bookings: bookings,
		Pagination: models.Pagination{
			Page:  params.Page,
			Limit: params.Limit,
		},
	}, nil
}
