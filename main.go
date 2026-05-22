package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// TelemetryEvent represents a structured log or state change.
type TelemetryEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"` // "LOG", "STATE", "TASK", "METRIC"
	Message   string                 `json:"message"`
	State     string                 `json:"state,omitempty"`
	AgentID   string                 `json:"agent_id"`
	Component string                 `json:"component,omitempty"`
	Severity  string                 `json:"severity,omitempty"` // "INFO", "WARNING", "CRITICAL"
	Metrics   map[string]interface{} `json:"metrics,omitempty"`   // e.g. {"stabilization": 92.5, "flux_density": 4.2}
}

type TelemetryHub struct {
	SocketPath string
	clients    map[net.Conn]bool
	mu         sync.Mutex

	history    []TelemetryEvent
	maxHistory int

	sseClients map[chan TelemetryEvent]bool
	sseMu      sync.Mutex
}

func NewTelemetryHub(path string) *TelemetryHub {
	return &TelemetryHub{
		SocketPath: path,
		clients:    make(map[net.Conn]bool),
		maxHistory: 500,
		sseClients: make(map[chan TelemetryEvent]bool),
	}
}

func (h *TelemetryHub) Start() error {
	_ = os.Remove(h.SocketPath)
	l, err := net.Listen("unix", h.SocketPath)
	if err != nil {
		return err
	}

	log.Printf("Telemetry Hub listening on Unix socket: %s\n", h.SocketPath)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				continue
			}
			h.mu.Lock()
			h.clients[conn] = true
			h.mu.Unlock()

			// Handle reading from this connection in a goroutine
			go h.handleConnection(conn)
		}
	}()
	return nil
}

func (h *TelemetryHub) handleConnection(conn net.Conn) {
	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
	}()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Try to parse the incoming line as a TelemetryEvent to validate it
		var event TelemetryEvent
		if err := json.Unmarshal(line, &event); err != nil {
			log.Printf("Received invalid JSON payload: %s (error: %v)\n", string(line), err)
			continue
		}

		// Fill in timestamp if not set
		if event.Timestamp.IsZero() {
			event.Timestamp = time.Now()
		}

		// Update history cache
		h.mu.Lock()
		h.history = append(h.history, event)
		if len(h.history) > h.maxHistory {
			h.history = h.history[len(h.history)-h.maxHistory:]
		}
		h.mu.Unlock()

		// Broadcast to SSE clients
		h.BroadcastSSE(event)

		// Re-marshal to clean up and ensure it ends with a single newline
		cleanedData, err := json.Marshal(event)
		if err == nil {
			log.Printf("[%s] [%s] %s (State: %s, Component: %s, Severity: %s, Metrics: %v)\n",
				event.AgentID, event.Type, event.Message, event.State, event.Component, event.Severity, event.Metrics)
			h.Broadcast(conn, cleanedData)
		}
	}
}

func (h *TelemetryHub) Broadcast(sender net.Conn, data []byte) {
	payload := append(data, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		// Do not send the event back to the publisher that sent it
		if conn == sender {
			continue
		}

		_, err := conn.Write(payload)
		if err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

func (h *TelemetryHub) AddSSEClient(ch chan TelemetryEvent) {
	h.sseMu.Lock()
	defer h.sseMu.Unlock()
	h.sseClients[ch] = true
}

func (h *TelemetryHub) RemoveSSEClient(ch chan TelemetryEvent) {
	h.sseMu.Lock()
	defer h.sseMu.Unlock()
	delete(h.sseClients, ch)
}

func (h *TelemetryHub) BroadcastSSE(event TelemetryEvent) {
	h.sseMu.Lock()
	defer h.sseMu.Unlock()
	for ch := range h.sseClients {
		select {
		case ch <- event:
		default:
			// Non-blocking write to channel
		}
	}
}

func (h *TelemetryHub) StartHTTPServer(port string) {
	http.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.mu.Lock()
		events := make([]TelemetryEvent, len(h.history))
		copy(events, h.history)
		h.mu.Unlock()

		json.NewEncoder(w).Encode(events)
	})

	http.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		ch := make(chan TelemetryEvent, 50)
		h.AddSSEClient(ch)
		defer h.RemoveSSEClient(ch)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, ": connection established\n\n")
		flusher.Flush()

		ctx := r.Context()
		for {
			select {
			case event := <-ch:
				data, err := json.Marshal(event)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "data: %s\n\n", string(data))
				flusher.Flush()
			case <-ctx.Done():
				return
			}
		}
	})

	log.Printf("Starting HTTP API Server on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Printf("HTTP server failed: %v\n", err)
	}
}

func main() {
	socketPath := "/tmp/foundations.sock"
	hub := NewTelemetryHub(socketPath)
	if err := hub.Start(); err != nil {
		log.Fatalf("Failed to start telemetry hub: %v", err)
	}

	go hub.StartHTTPServer("8085")

	fmt.Printf("Antigravity Foundations Telemetry Hub Initialized.\n")
	fmt.Printf("Listening at %s (socket) and :8085 (HTTP)\n", socketPath)

	// Keep the main goroutine alive
	select {}
}
