/*
File: main.go
Description: This file establishes the core interfaces, types, and the recursive orchestration
loop required for a self-healing, agentic architecture. It includes the Agent state machine,
the primary execution routine, and a polling mechanism to receive instructions from the
axis-mundi MCP server.
*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"foundations/mcp"
)

// TelemetryEvent represents a structured log or state change.
type TelemetryEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "LOG", "STATE", "TASK"
	Message   string    `json:"message"`
	State     string    `json:"state,omitempty"`
	AgentID   string    `json:"agent_id"`
}

// TelemetryPublisher manages Unix socket connections and broadcasts events.
type TelemetryPublisher struct {
	SocketPath string
	clients    map[net.Conn]bool
	mu         sync.Mutex
}

func NewTelemetryPublisher(path string) *TelemetryPublisher {
	return &TelemetryPublisher{
		SocketPath: path,
		clients:    make(map[net.Conn]bool),
	}
}

func (p *TelemetryPublisher) Start() error {
	_ = os.Remove(p.SocketPath)
	l, err := net.Listen("unix", p.SocketPath)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				continue
			}
			p.mu.Lock()
			p.clients[conn] = true
			p.mu.Unlock()
		}
	}()
	return nil
}

func (p *TelemetryPublisher) Broadcast(event TelemetryEvent) {
	data, _ := json.Marshal(event)
	data = append(data, '\n')

	p.mu.Lock()
	defer p.mu.Unlock()
	for conn := range p.clients {
		_, err := conn.Write(data)
		if err != nil {
			conn.Close()
			delete(p.clients, conn)
		}
	}
}

// Action defines the interface for any unit of work the agent can perform.
type Action interface {
	Execute(ctx context.Context) error
	Name() string
}

// Result captures the outcome of an execution attempt.
type Result struct {
	Timestamp time.Time
	Error     error
	Metadata  map[string]interface{}
}

// Agent represents the autonomous entity with its internal state and memory.
type Agent struct {
	ID         string
	MaxRetries int
	History    []Result
	State      string // "IDLE", "EXECUTING", "HEALING", "COMPLETE"
	mcpClient  *mcp.Client
	cliCommand string
	publisher  *TelemetryPublisher
}

func (a *Agent) setState(state string) {
	a.State = state
	a.publisher.Broadcast(TelemetryEvent{
		Timestamp: time.Now(),
		Type:      "STATE",
		Message:   fmt.Sprintf("Agent state changed to %s", state),
		State:     state,
		AgentID:   a.ID,
	})
}

func (a *Agent) logTelemetry(msg string) {
	log.Printf("[%s] %s\n", a.ID, msg)
	a.publisher.Broadcast(TelemetryEvent{
		Timestamp: time.Now(),
		Type:      "LOG",
		Message:   msg,
		State:     a.State,
		AgentID:   a.ID,
	})
}

// Orchestrator manages the recursive feedback loop.
func (a *Agent) Orchestrate(ctx context.Context, action Action) error {
	a.setState("EXECUTING")
	a.logTelemetry(fmt.Sprintf("Starting Action: %s", action.Name()))

	err := a.runRecursive(ctx, action, 0)

	if err != nil {
		a.setState("FAILED")
		return err
	}

	a.setState("COMPLETE")
	return nil
}

// runRecursive handles the actual nesting of execution and healing logic.
func (a *Agent) runRecursive(ctx context.Context, action Action, depth int) error {
	if depth >= a.MaxRetries {
		return fmt.Errorf("maximum recursion depth reached: %d", depth)
	}

	err := action.Execute(ctx)

	res := Result{
		Timestamp: time.Now(),
		Error:     err,
		Metadata:  make(map[string]interface{}),
	}
	a.History = append(a.History, res)

	if err == nil {
		return nil
	}

	// Trigger Healing State
	a.setState("HEALING")
	a.logTelemetry(fmt.Sprintf("Failure detected: %v. Initiating healing cycle %d...", err, depth+1))

	return a.runRecursive(ctx, action, depth+1)
}

// StartPolling connects to the MCP server and constantly requests pending tasks.
func (a *Agent) StartPolling(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.pollTasks()
		}
	}
}

func (a *Agent) pollTasks() {
	items, err := a.mcpClient.ListWorkspace()
	if err != nil {
		a.logTelemetry(fmt.Sprintf("Error listing workspace: %v", err))
		return
	}

	for _, item := range items {
		if item.Type == "keep" && item.Status == "Execute" {
			a.logTelemetry(fmt.Sprintf("Found keep note to execute: %s (%s)", item.Title, item.ID))

			// Mark as Active
			_ = a.mcpClient.SetStatus(item.ID, "Active")
			a.setState("EXECUTING")

			// Execute task based on instruction
			content, err := a.mcpClient.GetItemContent(item.ID, item.Type)
			if err != nil {
				a.logTelemetry(fmt.Sprintf("Error getting content: %v", err))
				_ = a.mcpClient.SetStatus(item.ID, "Error")
				a.setState("IDLE")
				continue
			}

			a.logTelemetry(fmt.Sprintf("Executing keep note content: %s", content))

			// If the instruction implies "restart", handle that
			if item.Title == "Restart" || content == "restart" {
				a.logTelemetry("Restart command received. Restarting...")
				_ = a.mcpClient.SetStatus(item.ID, "Complete")
				a.restartSelf()
			} else {
				a.logTelemetry(fmt.Sprintf("Launching CLI Agent (%s) in current terminal...", a.cliCommand))

				// Write the instruction to a temp file to safely handle multiline/special chars
				tmpFile, err := os.CreateTemp("", "axis-mundi-*.txt")
				if err != nil {
					a.logTelemetry(fmt.Sprintf("Failed to create temp file: %v", err))
					_ = a.mcpClient.SetStatus(item.ID, "Error")
					a.setState("IDLE")
					continue
				}
				_, _ = tmpFile.Write([]byte(content))
				tmpFile.Close()
				tmpPath := tmpFile.Name()

				// Launch via interactive bash login shell so nvm/PATH is sourced and TTY is inherited
				shellCmd := fmt.Sprintf("%s \"$(cat '%s')\"; EXIT_CODE=$?; rm -f '%s'; exit $EXIT_CODE", a.cliCommand, tmpPath, tmpPath)
				cmd := exec.Command("bash", "-ic", shellCmd)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				err = cmd.Run()
				if err != nil {
					a.logTelemetry(fmt.Sprintf("CLI Agent session aborted or failed: %v", err))
					_ = a.mcpClient.SetStatus(item.ID, "Blocked")
				} else {
					a.logTelemetry("CLI Agent session completed successfully.")
					_ = a.mcpClient.SetStatus(item.ID, "Complete")
				}
				a.setState("IDLE")
			}
		}
	}
}

// restartSelf uses syscall.Exec to replace the current process with a new instance of itself.
func (a *Agent) restartSelf() {
	binary, err := os.Executable()
	if err != nil {
		a.logTelemetry(fmt.Sprintf("Failed to get executable path: %v", err))
		return
	}

	a.logTelemetry(fmt.Sprintf("Restarting binary: %s", binary))
	err = syscall.Exec(binary, os.Args, os.Environ())
	if err != nil {
		a.logTelemetry(fmt.Sprintf("Failed to exec: %v", err))
	}
}

func main() {
	baseURL := os.Getenv("AXIS_MUNDI_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/mcp"
	}

	agentCLI := os.Getenv("AGENT_CLI")
	if agentCLI == "" {
		agentCLI = "gemini"
	}

	publisher := NewTelemetryPublisher("/tmp/foundations.sock")
	if err := publisher.Start(); err != nil {
		log.Fatalf("Failed to start telemetry publisher: %v", err)
	}

	ctx := context.Background()
	agent := &Agent{
		ID:         "ORCHESTRATOR-01",
		MaxRetries: 5,
		State:      "IDLE",
		mcpClient:  mcp.NewClient(baseURL),
		cliCommand: agentCLI,
		publisher:  publisher,
	}

	fmt.Printf("Agentic Foundation Initialized. ID: %s\n", agent.ID)
	fmt.Printf("Using CLI Agent: %s\n", agent.cliCommand)
	fmt.Printf("Connecting to MCP Server at %s\n", baseURL)

	agent.setState("IDLE")
	agent.StartPolling(ctx)
}
