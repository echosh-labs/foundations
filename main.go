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
	"os/exec"
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
	cliCommand string
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
		if item.Type == "keep" && item.Status == "Execute" {
			log.Printf("[%s] Found keep note to execute: %s (%s)\n", a.ID, item.Title, item.ID)

			// Mark as Active
			_ = a.mcpClient.SetStatus(item.ID, "Active")

			// Execute task based on instruction
			content, err := a.mcpClient.GetItemContent(item.ID, item.Type)
			if err != nil {
				log.Printf("[%s] Error getting content: %v\n", a.ID, err)
				_ = a.mcpClient.SetStatus(item.ID, "Error")
				continue
			}

			log.Printf("[%s] Executing keep note content: %s\n", a.ID, content)

			// If the instruction implies "restart", handle that
			if item.Title == "Restart" || content == "restart" {
				log.Printf("[%s] Restart command received. Restarting...\n", a.ID)
				_ = a.mcpClient.SetStatus(item.ID, "Complete")
				a.restartSelf()
			} else {
				log.Printf("[%s] Launching CLI Agent (%s) in current terminal...\n", a.ID, a.cliCommand)

				// Write the instruction to a temp file to safely handle multiline/special chars
				tmpFile, err := os.CreateTemp("", "axis-mundi-*.txt")
				if err != nil {
					log.Printf("[%s] Failed to create temp file: %v\n", a.ID, err)
					_ = a.mcpClient.SetStatus(item.ID, "Error")
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
					log.Printf("[%s] CLI Agent session aborted or failed: %v\n", a.ID, err)
					_ = a.mcpClient.SetStatus(item.ID, "Blocked")
				} else {
					log.Printf("[%s] CLI Agent session completed successfully.\n", a.ID)
					_ = a.mcpClient.SetStatus(item.ID, "Complete")
				}
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

	agentCLI := os.Getenv("AGENT_CLI")
	if agentCLI == "" {
		agentCLI = "gemini"
	}

	ctx := context.Background()
	agent := &Agent{
		ID:         "ORCHESTRATOR-01",
		MaxRetries: 5,
		State:      "IDLE",
		mcpClient:  mcp.NewClient(baseURL),
		cliCommand: agentCLI,
	}

	fmt.Printf("Agentic Foundation Initialized. ID: %s\n", agent.ID)
	fmt.Printf("Using CLI Agent: %s\n", agent.cliCommand)
	fmt.Printf("Connecting to MCP Server at %s\n", baseURL)

	agent.StartPolling(ctx)
}
