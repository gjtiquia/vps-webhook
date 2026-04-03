package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"vps-webhook/internal/admin"
	"vps-webhook/internal/db"
)

func main() {
	port := flag.Int("port", 9001, "admin dashboard port")
	dbPath := flag.String("db", "./db.sqlite", "path to sqlite database")
	logsDir := flag.String("logs", "./logs", "directory to store request logs")
	flag.Parse()

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	srv := admin.NewServer(database, *logsDir)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("admin dashboard listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
