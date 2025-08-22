package main

import (
	"context"
	"flag"
	"log"
	"net/http"
)

func main() {
	dbPath := flag.String("db", "data.db", "path to sqlite database")
	addr := flag.String("addr", ":8080", "http listen address")
	flag.Parse()

	db, err := OpenDB(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := EnsureSchema(context.Background(), db); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	srv := NewServer(db)
	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, srv.routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
