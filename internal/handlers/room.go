package handlers

import (
	"net/http"

	"github.com/chiamck/hotel-booking/internal/service"

	"github.com/gin-gonic/gin"
)

type RoomHandler struct {
	service *service.RoomService
}

func NewRoomHandler(service *service.RoomService) *RoomHandler {
	return &RoomHandler{service: service}
}

func (h *RoomHandler) List(c *gin.Context) {
	rooms, err := h.service.ListRooms()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rooms"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}
