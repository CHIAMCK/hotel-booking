package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/chiamck/hotel-booking/internal/models"
	"github.com/lib/pq"
)

var (
	ErrBookingNotFound      = errors.New("booking not found")
	ErrRoomNotFound         = errors.New("room not found")
	ErrRoomNotAvailable     = errors.New("room is not available")
	ErrBookingOverlap       = errors.New("room is already booked for the selected dates")
	ErrIdempotencyConflict  = errors.New("idempotency key conflict")
)

type CreateBookingParams struct {
	RoomID         int
	CustomerID     int
	CheckIn        time.Time
	CheckOut       time.Time
	IdempotencyKey string
}

type BookingRepository interface {
	FindByIdempotencyKey(key string) (*models.Booking, error)
	Create(params CreateBookingParams) (models.Booking, error)
}

type bookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository) FindByIdempotencyKey(key string) (*models.Booking, error) {
	row := r.db.QueryRow(`
		SELECT id, room_id, customer_id, start_time, end_time, status,
		       total_amount, price_per_night, COALESCE(idempotency_key, '')
		FROM bookings
		WHERE idempotency_key = $1`, key)

	booking, err := scanBooking(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &booking, nil
}

func (r *bookingRepository) Create(params CreateBookingParams) (models.Booking, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return models.Booking{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var roomStatus string
	var basePrice float64
	err = tx.QueryRow(`
		SELECT r.status, rc.base_price
		FROM rooms r
		JOIN room_categories rc ON rc.id = r.category_id
		WHERE r.id = $1
		FOR UPDATE`, params.RoomID).Scan(&roomStatus, &basePrice)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Booking{}, ErrRoomNotFound
	}
	if err != nil {
		return models.Booking{}, err
	}
	if roomStatus != "available" {
		return models.Booking{}, ErrRoomNotAvailable
	}

	nights := int(params.CheckOut.Sub(params.CheckIn).Hours() / 24)
	if nights < 1 {
		return models.Booking{}, ErrBookingOverlap
	}

	totalAmount := basePrice * float64(nights)

	row := tx.QueryRow(`
		INSERT INTO bookings (
			room_id, customer_id, start_time, end_time, status,
			total_amount, price_per_night, idempotency_key
		) VALUES ($1, $2, $3, $4, 'confirmed', $5, $6, $7)
		RETURNING id, room_id, customer_id, start_time, end_time, status,
		          total_amount, price_per_night, COALESCE(idempotency_key, '')`,
		params.RoomID,
		params.CustomerID,
		params.CheckIn,
		params.CheckOut,
		totalAmount,
		basePrice,
		params.IdempotencyKey,
	)

	booking, err := scanBooking(row)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code {
			case "23P01":
				return models.Booking{}, ErrBookingOverlap
			case "23505":
				if pqErr.Constraint == "idx_bookings_idempotency_key" {
					return models.Booking{}, ErrIdempotencyConflict
				}
			}
		}
		return models.Booking{}, err
	}

	if err := tx.Commit(); err != nil {
		return models.Booking{}, err
	}

	return booking, nil
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
		&booking.IdempotencyKey,
	)
	return booking, err
}
