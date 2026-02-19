package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/MnemoShare/projectboard-mcp/internal/mcp"
	"github.com/MnemoShare/projectboard-mcp/internal/taskboard"
)

func main() {
	// Initialize TaskBoard client
	client, err := taskboard.NewClientFromEnv()
	if err != nil {
		log.Fatalf("Failed to initialize TaskBoard client: %v", err)
	}

	// Create MCP server
	server := mcp.NewServer(client)

	// Run stdio transport
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		
		var request mcp.Request
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		response := server.Handle(&request)
		
		respBytes, _ := json.Marshal(response)
		fmt.Println(string(respBytes))
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading stdin: %v", err)
	}
}

func sendError(id interface{}, code int, message, data string) {
	resp := mcp.Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcp.Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	respBytes, _ := json.Marshal(resp)
	fmt.Println(string(respBytes))
}
