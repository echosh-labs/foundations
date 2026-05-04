package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type WorkspaceItem struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Snippet string `json:"snippet"`
	Status string `json:"status"`
}

type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      interface{} `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      interface{}     `json:"id"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type Client struct {
	BaseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL}
}

func (c *Client) CallTool(name string, arguments map[string]interface{}) (json.RawMessage, error) {
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      name,
			"arguments": arguments,
		},
		ID: 1,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(c.BaseURL, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d (%s)", resp.StatusCode, string(body))
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON-RPC response: %v (body: %s)", err, string(body))
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s (code: %d)", rpcResp.Error.Message, rpcResp.Error.Code)
	}

	// Extract result from MCP tool call response
	var callResult struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(rpcResp.Result, &callResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tool result: %v (result: %s)", err, string(rpcResp.Result))
	}

	if len(callResult.Content) == 0 {
		return nil, fmt.Errorf("tool returned no content")
	}

	return json.RawMessage(callResult.Content[0].Text), nil
}

func (c *Client) ListWorkspace() ([]WorkspaceItem, error) {
	res, err := c.CallTool("list_workspace", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var result struct {
		Items []WorkspaceItem `json:"items"`
	}
	if err := json.Unmarshal(res, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal items: %v (result: %s)", err, string(res))
	}

	return result.Items, nil
}

func (c *Client) GetStatus(id string) (string, error) {
	res, err := c.CallTool("get_status", map[string]interface{}{"id": id})
	if err != nil {
		return "", err
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(res, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal status: %v (result: %s)", err, string(res))
	}

	return result.Status, nil
}

func (c *Client) SetStatus(id, status string) error {
	_, err := c.CallTool("set_status", map[string]interface{}{"id": id, "status": status})
	return err
}

func (c *Client) GetItemContent(id, itemType string) (string, error) {
	var toolName string
	var argName string
	switch itemType {
	case "keep":
		toolName = "get_note"
		argName = "noteId"
	case "doc":
		toolName = "get_doc"
		argName = "documentId"
	case "sheet":
		toolName = "get_sheet"
		argName = "spreadsheetId"
	case "gmail":
		toolName = "get_gmail_thread"
		argName = "threadId"
	default:
		return "", fmt.Errorf("unsupported item type: %s", itemType)
	}

	res, err := c.CallTool(toolName, map[string]interface{}{argName: id})
	if err != nil {
		return "", err
	}

	var content string
	if err := json.Unmarshal(res, &content); err != nil {
		// Some tools might return complex objects, we might need more specific parsing later
		content = string(res)
	}

	return content, nil
}
