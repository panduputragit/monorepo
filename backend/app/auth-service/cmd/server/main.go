package main

import (
	"context"
	"fmt"
	"log"

	"github.com/panduputragit/gym/backend/app/auth-service/internal/config"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/handler"
	authhttp "github.com/panduputragit/gym/backend/app/auth-service/internal/http"
	"github.com/panduputragit/gym/backend/app/auth-service/internal/token"
	"github.com/panduputragit/gym/backend/packages/database"
	"github.com/panduputragit/gym/backend/packages/httpserver"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(context.Background(), database.Config{URL: cfg.DatabaseURL})
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()
	fmt.Printf("%s connected to database\n", cfg.Name)

	tokenMaker, err := token.NewMakerWithRandomKey()
	if err != nil {
		log.Fatalf("create token maker: %v", err)
	}

	router := httpserver.NewRouter(cfg.Name, cfg.GinMode)
	authhttp.RegisterRoutes(router, handler.New(db, tokenMaker))

	addr := ":" + cfg.Port
	fmt.Printf("%s listening on %s\n", cfg.Name, addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("start server: %v", err)
	}
}
