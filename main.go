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
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"foundations/mcp"
)

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
}

// Orchestrator manages the recursive feedback loop.
func (a *Agent) Orchestrate(ctx context.Context, action Action) error {
	a.State = "EXECUTING"
	fmt.Printf("[%s] Starting Action: %s\n", a.ID, action.Name())

	err := a.runRecursive(ctx, action, 0)

	if err != nil {
		a.State = "FAILED"
		return err
	}

	a.State = "COMPLETE"
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
	a.State = "HEALING"
	fmt.Printf("[%s] Failure detected: %v. Initiating healing cycle %d...\n", a.ID, err, depth+1)

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
		log.Printf("[%s] Error listing workspace: %v\n", a.ID, err)
		return
	}

	for _, item := range items {
		if item.Status == "Pending" {
			log.Printf("[%s] Found pending task: %s (%s)\n", a.ID, item.Title, item.ID)

			// Mark as In Progress
			_ = a.mcpClient.SetStatus(item.ID, "In Progress")

			// Execute task based on instruction
			content, err := a.mcpClient.GetItemContent(item.ID, item.Type)
			if err != nil {
				log.Printf("[%s] Error getting content: %v\n", a.ID, err)
				_ = a.mcpClient.SetStatus(item.ID, "Failed")
				continue
			}

			log.Printf("[%s] Executing task content: %s\n", a.ID, content)

			// If the instruction implies "restart", handle that
			if item.Title == "Restart" || content == "restart" {
				log.Printf("[%s] Restart command received. Restarting...\n", a.ID)
				_ = a.mcpClient.SetStatus(item.ID, "Complete")
				a.restartSelf()
			} else {
				// Mark complete
				_ = a.mcpClient.SetStatus(item.ID, "Complete")
				log.Printf("[%s] Task %s completed.\n", a.ID, item.ID)
			}
		}
	}
}

// restartSelf uses syscall.Exec to replace the current process with a new instance of itself.
func (a *Agent) restartSelf() {
	binary, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable path: %v", err)
		return
	}

	log.Printf("[%s] Restarting binary: %s", a.ID, binary)
	err = syscall.Exec(binary, os.Args, os.Environ())
	if err != nil {
		log.Printf("Failed to exec: %v", err)
	}
}

func main() {
	baseURL := os.Getenv("AXIS_MUNDI_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080/mcp"
	}

	ctx := context.Background()
	agent := &Agent{
		ID:         "ORCHESTRATOR-01",
		MaxRetries: 5,
		State:      "IDLE",
		mcpClient:  mcp.NewClient(baseURL),
	}

	fmt.Printf("Agentic Foundation Initialized. ID: %s\n", agent.ID)
	fmt.Printf("Connecting to MCP Server at %s\n", baseURL)

	agent.StartPolling(ctx)
}
