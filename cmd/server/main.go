package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpAdapter "factory-traffic/internal/adapters/http"
	"factory-traffic/internal/adapters/simcontroller"
	"factory-traffic/internal/adapters/sqlite"
	"factory-traffic/internal/app"
	"factory-traffic/internal/domain"
	"factory-traffic/internal/ports"
)

func main() {
	port := flag.Int("port", 8080, "HTTP server listening port")
	dbPath := flag.String("db", "traffic.db", "SQLite database path (use :memory: for ephemeral)")
	webDir := flag.String("web", "web", "Directory holding static web UI files")
	junctionID := flag.String("junction", "A", "Primary factory junction ID (e.g. A, B, C, D)")
	flag.Parse()

	log.Printf("[TrafficMaster] Initializing Factory Traffic Controller on port %d...", *port)

	// 1. Storage Repository
	repo, err := sqlite.NewSQLiteRepo(*dbPath)
	if err != nil {
		log.Fatalf("Fatal: could not initialize SQLite database: %v", err)
	}
	defer repo.Close()

	// 2. Clock & Infrastructure
	clock := ports.NewRealClock()
	cfg := domain.DefaultConfig()

	// 3. Service & Simulated Hardware Controller
	var service *app.TrafficService
	simCtrl := simcontroller.NewSimController(func(ack domain.ControllerAck) {
		if service != nil {
			if err := service.ProcessControllerAck(context.Background(), ack); err != nil {
				log.Printf("[Main] Error processing hardware ACK: %v", err)
			}
		}
	})

	service = app.NewTrafficService(simCtrl, repo, clock)

	// 4. Boot-time State Recovery for Primary Junction
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	initialStateA, err := app.RecoverJunction(ctx, *junctionID, repo, cfg, clock)
	if err != nil {
		log.Fatalf("Fatal: error recovering junction %s: %v", *junctionID, err)
	}
	service.RegisterJunction(ctx, initialStateA)
	log.Printf("[TrafficMaster] Primary Junction '%s' active in mode %s, initial phase %s",
		*junctionID, initialStateA.Mode, initialStateA.CurrentPhase)

	// Also register junction-1 as an alias if junctionID is A
	if *junctionID == "A" {
		aliasState, _ := app.RecoverJunction(ctx, "junction-1", repo, cfg, clock)
		service.RegisterJunction(ctx, aliasState)
	}

	// 6. HTTP API & Dashboard Setup
	handler := httpAdapter.NewHandler(service, simCtrl)
	router := httpAdapter.NewRouter(handler, *webDir)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 7. Graceful Shutdown Setup
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[TrafficMaster] Server running at http://localhost:%d", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listen failed: %v", err)
		}
	}()

	<-stopCh
	log.Println("[TrafficMaster] Shutting down gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	service.StopAll()
	log.Println("[TrafficMaster] Server exited safely.")
}
