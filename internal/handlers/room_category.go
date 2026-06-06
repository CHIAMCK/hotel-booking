package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/chiamck/hotel-booking/internal/service"

	"github.com/gin-gonic/gin"
)

type RoomCategoryHandler struct {
	service *service.RoomCategoryService
}

func NewRoomCategoryHandler(service *service.RoomCategoryService) *RoomCategoryHandler {
	return &RoomCategoryHandler{service: service}
}

type roomCategorySearchQuery struct {
	HotelID  string `form:"hotel_id" binding:"required"`
	CheckIn  string `form:"check_in" binding:"required"`
	CheckOut string `form:"check_out" binding:"required"`
	Guests   string `form:"guests" binding:"required"`
	Page     string `form:"page"`
	Limit    string `form:"limit"`
}

func (h *RoomCategoryHandler) Search(c *gin.Context) {
	var query roomCategorySearchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hotel_id, check_in, check_out, and guests are required"})
		return
	}

	hotelID, err := strconv.Atoi(query.HotelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hotel_id must be a positive integer"})
		return
	}

	guests, err := strconv.Atoi(query.Guests)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "guests must be a positive integer"})
		return
	}

	page := 1
	if query.Page != "" {
		page, err = strconv.Atoi(query.Page)
		if err != nil || page < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "page must be a positive integer"})
			return
		}
	}

	limit := 10
	if query.Limit != "" {
		limit, err = strconv.Atoi(query.Limit)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 10"})
			return
		}
	}

	result, err := h.service.SearchCategories(service.RoomCategorySearchInput{
		HotelID:  hotelID,
		Guests:   guests,
		CheckIn:  query.CheckIn,
		CheckOut: query.CheckOut,
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		var validationErr error
		switch {
		case errors.Is(err, service.ErrInvalidHotelID),
			errors.Is(err, service.ErrInvalidGuests),
			errors.Is(err, service.ErrInvalidCheckIn),
			errors.Is(err, service.ErrInvalidCheckOut),
			errors.Is(err, service.ErrInvalidDateRange),
			errors.Is(err, service.ErrInvalidLimit),
			errors.Is(err, service.ErrInvalidPage):
			validationErr = err
		}
		if validationErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": service.ValidationErrorMessage(err)})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search room categories"})
		return
	}

	c.JSON(http.StatusOK, result)
}
