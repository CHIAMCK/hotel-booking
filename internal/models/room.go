package models

type Room struct {
	ID         int    `json:"id"`
	HotelID    int    `json:"hotel_id"`
	CategoryID int    `json:"category_id"`
	Number     string `json:"number"`
	Status     string `json:"status"`
}
