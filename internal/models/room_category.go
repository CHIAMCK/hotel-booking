package models

type RoomCategorySearchResult struct {
	ID             int     `json:"id"`
	Name           string  `json:"name"`
	MaxPerson      int     `json:"max_person"`
	BasePrice      float64 `json:"base_price"`
	AvailableCount int     `json:"available_count"`
}

type Pagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type RoomCategorySearchPage struct {
	Categories []RoomCategorySearchResult `json:"categories"`
	Pagination Pagination                 `json:"pagination"`
}
