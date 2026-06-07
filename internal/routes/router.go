package routes

import (
	"github.com/chiamck/hotel-booking/internal/handlers"
	"github.com/chiamck/hotel-booking/internal/idempotency"
	"github.com/chiamck/hotel-booking/internal/lock"
	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/service"

	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	RoomRepo           repository.RoomRepository
	RoomCategoryRepo   repository.RoomCategoryRepository
	BookingRepo        repository.BookingRepository
	Lock               lock.DistributedLock
	BookingIdempotency idempotency.BookingStore
}

func SetupRouter(deps Dependencies) *gin.Engine {
	bookingService := service.NewBookingService(deps.BookingRepo, deps.Lock, deps.BookingIdempotency)
	roomHandler := handlers.NewRoomHandler(bookingService)

	roomCategoryService := service.NewRoomCategoryService(deps.RoomCategoryRepo)
	roomCategoryHandler := handlers.NewRoomCategoryHandler(roomCategoryService)

	bookingHandler := handlers.NewBookingHandler(bookingService)

	router := gin.Default()

	registerRootRoutes(router)
	registerV1Routes(router.Group("/api/v1"), roomHandler, roomCategoryHandler, bookingHandler)

	return router
}

func registerRootRoutes(router *gin.Engine) {
	router.GET("/", handlers.Welcome)
	router.GET("/health", handlers.Health)
}

func registerV1Routes(
	router *gin.RouterGroup,
	roomHandler *handlers.RoomHandler,
	roomCategoryHandler *handlers.RoomCategoryHandler,
	bookingHandler *handlers.BookingHandler,
) {
	registerRoomRoutes(router, roomHandler)
	registerRoomCategoryRoutes(router, roomCategoryHandler)
	registerBookingRoutes(router, bookingHandler)
}
