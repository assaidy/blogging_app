package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"github.com/assaidy/blogging_app/internal/repositry"
	"github.com/assaidy/blogging_app/internal/server"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
)

func main() {
	db, err := sql.Open("postgres", os.Getenv("PG_URL"))
	if err != nil {
		log.Fatal("error connecting to postgres db:", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		log.Fatal("error pinging postgres db:", err)
	}

	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(1 * time.Minute)

	app := server.NewAppServer(":"+os.Getenv("PORT"), repositry.New(db))

	if err := app.Run(); err != nil {
		log.Fatal("error running server:", err)
	}
}
