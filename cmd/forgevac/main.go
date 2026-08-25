package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/wyw14/cry-127/internal/api"
	"github.com/wyw14/cry-127/internal/service"
)

func main() {
	address := flag.String("addr", envOr("FORGEVAC_ADDR", "127.0.0.1:21227"), "")
	dataDir := flag.String("data", envOr("FORGEVAC_DATA_DIR", "./data"), "")
	flag.Parse()
	if err := run(*address, *dataDir); err != nil {
		log.Fatal(err)
	}
}

func run(address, dataDir string) error {
	if address == "" || dataDir == "" {
		return errors.New("address and data directory are required")
	}
	runtime, err := service.NewRuntime(dataDir, time.Now)
	if err != nil {
		return err
	}
	server, err := api.NewServer(runtime)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:              address,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	return httpServer.ListenAndServe()
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
