package postgres_db

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func init() {
	db, err := sql.Open("postgres", os.Getenv("PG_URL"))
	if err != nil {
		log.Fatal("error connecting to postgres db:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal("error pinging postgres db:", err)
	}

	// TODO: set these variables in config package
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(10 * time.Minute)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(1 * time.Minute)
}
