package main

import (
	"log"

	"github.com/chiamck/hotel-booking/internal/config"
	"github.com/chiamck/hotel-booking/internal/database"
	"github.com/chiamck/hotel-booking/internal/idempotency"
	"github.com/chiamck/hotel-booking/internal/lock"
	"github.com/chiamck/hotel-booking/internal/repository"
	"github.com/chiamck/hotel-booking/internal/routes"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	redisClient, err := database.ConnectRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	defer redisClient.Close()

	router := routes.SetupRouter(routes.Dependencies{
		RoomRepo:           repository.NewRoomRepository(db),
		RoomCategoryRepo:   repository.NewRoomCategoryRepository(db),
		BookingRepo:        repository.NewBookingRepository(db),
		Lock:               lock.NewRedisLock(redisClient),
		BookingIdempotency: idempotency.NewRedisBookingStore(redisClient),
	})

	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
