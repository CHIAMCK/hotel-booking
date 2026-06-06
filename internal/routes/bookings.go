package routes

import (
	"github.com/chiamck/hotel-booking/internal/handlers"

	"github.com/gin-gonic/gin"
)

func registerBookingRoutes(router *gin.RouterGroup, bookingHandler *handlers.BookingHandler) {
	router.POST("/bookings", bookingHandler.Create)
}
