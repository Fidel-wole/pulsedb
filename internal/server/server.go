package server

import (
	"net"
	"time"

	"pulsedb/internal/proto"
	"pulsedb/internal/store"
)

// Server represents the TCP server
type Server struct {
	store      *store.Store
	dispatcher *CommandDispatcher
}

// NewServer creates a new server instance
func NewServer(store *store.Store, metrics interface{}) *Server {
	return &Server{
		store:      store,
		dispatcher: NewCommandDispatcher(store, metrics),
	}
}

// HandleConnection handles a client connection
func (s *Server) HandleConnection(conn net.Conn) {
	defer conn.Close()

	reader := proto.NewRESPReader(conn)
	writer := proto.NewRESPWriter(conn)

	for {
		// Set read timeout
		conn.SetReadDeadline(time.Now().Add(30 * time.Second))

		value, err := reader.Read()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout, close connection
				return
			}
			// Connection closed or other error
			return
		}

		// Process command
		response := s.dispatcher.Dispatch(value)

		// Write response
		if err := writer.WriteValue(response); err != nil {
			return
		}
	}
}
