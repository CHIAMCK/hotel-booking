package handlers

import (
	"net/http"

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

	params, err := parseRoomCategorySearchQuery(query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.SearchCategories(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search room categories"})
		return
	}

	c.JSON(http.StatusOK, result)
}
