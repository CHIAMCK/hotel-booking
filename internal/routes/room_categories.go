package routes

import (
	"github.com/chiamck/hotel-booking/internal/handlers"

	"github.com/gin-gonic/gin"
)

func registerRoomCategoryRoutes(router *gin.RouterGroup, roomCategoryHandler *handlers.RoomCategoryHandler) {
	router.GET("/room-categories", roomCategoryHandler.Search)
}
