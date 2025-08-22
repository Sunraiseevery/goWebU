package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"goWebU/app"
)

func main() {
	dbPath := flag.String("db", "data.db", "path to sqlite database")
	addr := flag.String("addr", ":8080", "http listen address")
	flag.Parse()

	db, err := app.OpenDB(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()
	if err := app.EnsureSchema(context.Background(), db); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	srv := app.NewServer(db)

	go func() {
		// Allow the server a brief moment to start before opening the browser.
		time.Sleep(200 * time.Millisecond)
		openBrowser(fmt.Sprintf("http://localhost%s/", *addr))
	}()

	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, srv.Routes()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("failed to open browser: %v", err)
	}
}
