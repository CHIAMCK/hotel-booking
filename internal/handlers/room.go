package handlers

import (
	"net/http"
	"strconv"

	"github.com/chiamck/hotel-booking/internal/service"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	roomService    *service.RoomService
	bookingService *service.BookingService
}

func NewRoomHandler(roomService *service.RoomService, bookingService *service.BookingService) *RoomHandler {
	return &RoomHandler{roomService: roomService, bookingService: bookingService}
}

func (h *RoomHandler) Availability(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room id must be a positive integer"})
		return
	}

	exists, err := h.roomService.RoomExists(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load room"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
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
