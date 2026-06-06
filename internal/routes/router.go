package routes

import (
	"github.com/chiamck/hotel-booking/internal/handlers"
	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/service"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	roomRepo := repository.NewInMemoryRoomRepository()
	roomService := service.NewRoomService(roomRepo)
	roomHandler := handlers.NewRoomHandler(roomService)

	router := gin.Default()

	registerRootRoutes(router)
	registerV1Routes(router.Group("/api/v1"), roomHandler)

	return router
}

func registerRootRoutes(router *gin.Engine) {
	router.GET("/", handlers.Welcome)
	router.GET("/health", handlers.Health)
}

func registerV1Routes(router *gin.RouterGroup, roomHandler *handlers.RoomHandler) {
	registerRoomRoutes(router, roomHandler)
}
