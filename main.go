package main

import (
	"log"

	"github.com/chiamck/hotel-booking/internal/config"
	"github.com/chiamck/hotel-booking/internal/database"
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

	roomRepo := repository.NewRoomRepository(db)
	roomCategoryRepo := repository.NewRoomCategoryRepository(db)

	router := routes.SetupRouter(roomRepo, roomCategoryRepo)
	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
