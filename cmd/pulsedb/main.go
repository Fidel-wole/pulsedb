package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"pulsedb/internal/http"
	"pulsedb/internal/metrics"
	"pulsedb/internal/server"
	"pulsedb/internal/store"
)

const (
	DefaultTCPPort  = "6380"
	DefaultHTTPPort = "8080"
)

func main() {
	log.Println("Starting PulseDB...")

	// Initialize store with MVCC support
	db := store.NewStore()

	// Initialize metrics
	metricsRegistry := metrics.NewMetrics()

	// Create TCP server
	tcpServer := server.NewServer(db, metricsRegistry)

	// Create HTTP server
	httpServer := http.NewHTTPServer(db, metricsRegistry)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// Start TCP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := startTCPServer(ctx, tcpServer); err != nil {
			log.Printf("TCP server error: %v", err)
		}
	}()

	// Start HTTP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.Start(ctx, ":"+DefaultHTTPPort); err != nil {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Start background processes
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.StartBackgroundProcesses(ctx)
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("PulseDB is running on TCP port %s and HTTP port %s", DefaultTCPPort, DefaultHTTPPort)
	<-sigChan

	log.Println("Shutting down PulseDB...")
	cancel()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("PulseDB shutdown complete")
	case <-time.After(30 * time.Second):
		log.Println("Shutdown timeout exceeded")
	}
}

func startTCPServer(ctx context.Context, srv *server.Server) error {
	listener, err := net.Listen("tcp", ":"+DefaultTCPPort)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", DefaultTCPPort, err)
	}
	defer listener.Close()

	// Accept connections in a separate goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					log.Printf("Failed to accept connection: %v", err)
					continue
				}
			}

			go srv.HandleConnection(conn)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	return nil
}
