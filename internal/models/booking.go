package models

import "time"

type Booking struct {
	ID             int       `json:"id"`
	RoomID         int       `json:"room_id"`
	CustomerID     int       `json:"customer_id"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Status         string    `json:"status"`
	TotalAmount    float64   `json:"total_amount"`
	PricePerNight  float64   `json:"price_per_night"`
	IdempotencyKey string    `json:"idempotency_key,omitempty"`
}
