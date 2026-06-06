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

// Availability returns dates in the requested window (UTC) where the room has a pending or confirmed stay night.
// Query: optional from=YYYY-MM-DD, to=YYYY-MM-DD (defaults: from=today UTC, to=from+179 days). Responses are uncached for up-to-date availability.
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

	result, err := h.bookingService.GetRoomAvailability(c.Request.Context(), id, c.Query("from"), c.Query("to"))
	if err != nil {
		if service.IsAvailabilityValidationError(err) {
			c.JSON(http.StatusBadRequest, gin.H{"error": service.AvailabilityErrorMessage(err)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load availability"})
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, result)
}
