package handlers

import (
	"net/http"

	"github.com/chiamck/hotel-booking/internal/service"

	"github.com/gin-gonic/gin"
)

type BookingHandler struct {
	service *service.BookingService
}

func NewBookingHandler(service *service.BookingService) *BookingHandler {
	return &BookingHandler{service: service}
}

type createBookingRequest struct {
	RoomID     int    `json:"room_id" binding:"required"`
	CustomerID int    `json:"customer_id" binding:"required"`
	CheckIn    string `json:"check_in" binding:"required"`
	CheckOut   string `json:"check_out" binding:"required"`
}

func (h *BookingHandler) Create(c *gin.Context) {
	var req createBookingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "room_id, customer_id, check_in, and check_out are required"})
		return
	}

	booking, err := h.service.Create(c.Request.Context(), service.CreateBookingInput{
		RoomID:     req.RoomID,
		CustomerID: req.CustomerID,
		CheckIn:    req.CheckIn,
		CheckOut:   req.CheckOut,
	})

	if err != nil {
		switch {
		case service.IsBookingValidationError(err):
			c.JSON(http.StatusBadRequest, gin.H{"error": service.BookingErrorMessage(err)})
		case service.IsBookingNotFoundError(err):
			c.JSON(http.StatusNotFound, gin.H{"error": service.BookingErrorMessage(err)})
		case service.IsBookingConflictError(err):
			c.JSON(http.StatusConflict, gin.H{"error": service.BookingErrorMessage(err)})
		case service.IsIdempotencyCacheError(err):
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": service.BookingErrorMessage(err)})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create booking"})
		}
		return
	}

	c.JSON(http.StatusCreated, booking)
}
