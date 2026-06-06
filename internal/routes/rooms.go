package routes

import (
	"github.com/chiamck/hotel-booking/internal/handlers"

	"github.com/gin-gonic/gin"
)

func registerRoomRoutes(router *gin.RouterGroup, roomHandler *handlers.RoomHandler) {
	rooms := router.Group("/rooms")
	{
		rooms.GET("/:id/availability", roomHandler.Availability)
		rooms.GET("", roomHandler.List)
	}
}
