package handlers

import (
	"net/http"
	"strconv"

	"github.com/chiamck/hotel-booking/internal/service"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	bookingService *service.BookingService
}

func NewRoomHandler(bookingService *service.BookingService) *RoomHandler {
	return &RoomHandler{bookingService: bookingService}
}

func (h *RoomHandler) Availability(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room id must be a positive integer"})
		return
	}

	fromDate, toDate, err := parseAvailabilityQuery(c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.bookingService.GetRoomAvailability(c.Request.Context(), id, fromDate, toDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load availability"})
		return
	}

	c.JSON(http.StatusOK, result)
}
