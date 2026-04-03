package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"vps-webhook/internal/db"
	"vps-webhook/internal/webhook"
)

func main() {
	port := flag.Int("port", 9000, "webhook server port")
	dbPath := flag.String("db", "./db.sqlite", "path to sqlite database")
	logsDir := flag.String("logs", "./logs", "directory to store request logs")
	flag.Parse()

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	srv := webhook.NewServer(database, *logsDir)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("webhook server listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
