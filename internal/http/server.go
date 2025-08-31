package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"pulsedb/internal/store"
)

// HTTPServer represents the HTTP API server
type HTTPServer struct {
	store  *store.Store
	server *http.Server
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(store *store.Store, metrics interface{}) *HTTPServer {
	return &HTTPServer{
		store: store,
	}
}

// Start starts the HTTP server
func (h *HTTPServer) Start(ctx context.Context, addr string) error {
	mux := http.NewServeMux()

	// Key-value operations
	mux.HandleFunc("/kv/", h.handleKeyValue)

	// Health check
	mux.HandleFunc("/health", h.handleHealth)

	h.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return h.server.Shutdown(shutdownCtx)
}

// Response structures
type SetRequest struct {
	Value string `json:"value"`
	TTL   int64  `json:"ttl,omitempty"` // TTL in seconds
}

type GetResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Found bool   `json:"found"`
}

// Handler functions

func (h *HTTPServer) handleKeyValue(w http.ResponseWriter, r *http.Request) {
	// Parse the path to extract key and operation
	path := r.URL.Path[4:] // Remove "/kv/" prefix

	switch r.Method {
	case "GET":
		h.handleGet(w, r, path)
	case "POST", "PUT":
		h.handleSet(w, r, path)
	case "DELETE":
		h.handleDelete(w, r, path)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *HTTPServer) handleGet(w http.ResponseWriter, r *http.Request, key string) {
	value, found := h.store.Get(key)

	response := GetResponse{
		Key:   key,
		Value: value,
		Found: found,
	}

	w.Header().Set("Content-Type", "application/json")
	if !found {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(response)
}

func (h *HTTPServer) handleSet(w http.ResponseWriter, r *http.Request, key string) {
	var req SetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	ttlMs := req.TTL * 1000 // Convert seconds to milliseconds
	h.store.Set(key, req.Value, ttlMs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
}

func (h *HTTPServer) handleDelete(w http.ResponseWriter, r *http.Request, key string) {
	deleted := h.store.Delete(key)

	w.Header().Set("Content-Type", "application/json")
	if !deleted {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"deleted": deleted,
	})
}

func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	stats := h.store.Stats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"stats":  stats,
	})
}
