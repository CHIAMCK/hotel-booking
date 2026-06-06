package models

type RoomCategorySearchResult struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	MaxPerson      int     `json:"max_person"`
	BasePrice      float64 `json:"base_price"`
	AvailableCount int     `json:"available_count"`
}
