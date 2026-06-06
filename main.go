package main

import (
	"github.com/chiamck/hotel-booking/internal/config"
	"github.com/chiamck/hotel-booking/internal/routes"
)

func main() {
	cfg := config.Load()

	router := routes.SetupRouter()
	if err := router.Run(":" + cfg.Port); err != nil {
		panic(err)
	}
}
