package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/MnemoShare/projectboard-mcp/internal/taskboard"
)

const (
	ProtocolVersion = "2024-11-05"
	ServerName      = "taskboard-mcp"
	ServerVersion   = "0.1.0"
)

type Server struct {
	client *taskboard.Client
	tools  []Tool
}

func NewServer(client *taskboard.Client) *Server {
	s := &Server{client: client}
	s.registerTools()
	return s
}

func (s *Server) registerTools() {
	s.tools = []Tool{
		{
			Name:        "list_boards",
			Description: "List all boards",
			InputSchema: InputSchema{Type: "object"},
		},
		{
			Name:        "list_tasks",
			Description: "List tasks with optional filters",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"board_id": {Type: "string", Description: "Filter by board ID"},
					"status": {
						Type:        "string",
						Description: "Filter by status",
						Enum:        []string{"backlog", "todo", "in-progress", "in-qa", "completed", "rfp", "closed"},
					},
					"assignee": {Type: "string", Description: "Filter by assignee email"},
				},
			},
		},
		{
			Name:        "get_task",
			Description: "Get a task by ID or ticket number (e.g., MNS-42)",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {Type: "string", Description: "Task ID or ticket number"},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "create_task",
			Description: "Create a new task on a board. Requires board_id and name. The name is the short title shown on backlog and swimlane views.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"board_id": {Type: "string", Description: "Board ID (required). Use list_boards to find the ID."},
					"name":     {Type: "string", Description: "Short summary/title of the task (required). This is displayed as the task heading on backlog and swimlane board views. Keep it concise — one line, under 80 characters."},
					"description": {Type: "string", Description: "Full detailed description of the task. Supports markdown. Use this for steps to reproduce, acceptance criteria, implementation details, etc."},
					"assignee":    {Type: "string", Description: "Assignee email address. Use list_users to find valid emails."},
					"priority": {
						Type:        "integer",
						Description: "Priority (1=highest, 5=lowest). Defaults to 3.",
					},
					"status": {
						Type:        "string",
						Description: "Initial status. Defaults to 'backlog'.",
						Enum:        []string{"backlog", "todo", "in-progress", "in-qa", "completed", "rfp", "closed"},
					},
				},
				Required: []string{"board_id", "name"},
			},
		},
		{
			Name:        "update_task",
			Description: "Update an existing task's fields. Only provided fields are changed.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id":          {Type: "string", Description: "Task ID or ticket number (e.g., MNS-42). Required."},
					"name":        {Type: "string", Description: "New short summary/title for the task."},
					"description": {Type: "string", Description: "New full description. Supports markdown."},
					"assignee":    {Type: "string", Description: "New assignee email address."},
					"status": {
						Type:        "string",
						Description: "New status.",
						Enum:        []string{"backlog", "todo", "in-progress", "in-qa", "completed", "rfp", "closed"},
					},
					"priority": {
						Type:        "integer",
						Description: "New priority (1=highest, 5=lowest).",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "add_comment",
			Description: "Add a comment to a task",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"task_id": {Type: "string", Description: "Task ID or ticket number"},
					"text":    {Type: "string", Description: "Comment text"},
				},
				Required: []string{"task_id", "text"},
			},
		},
		{
			Name:        "list_users",
			Description: "List all team members (for task assignment)",
			InputSchema: InputSchema{Type: "object"},
		},
		{
			Name:        "create_user",
			Description: "Create or invite a new team member (human or agent)",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"name":     {Type: "string", Description: "User's display name"},
					"email":    {Type: "string", Description: "User's email address (must be unique)"},
					"avatar":   {Type: "string", Description: "Emoji avatar (defaults to 🤖 for agents, 👤 for humans)"},
					"is_agent": {Type: "boolean", Description: "Whether this user is an AI agent (agents get an API token)"},
				},
				Required: []string{"name", "email"},
			},
		},
	}
}

func (s *Server) Handle(req *Request) *Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		return &Response{JSONRPC: "2.0", ID: req.ID}
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &Error{Code: -32601, Message: "Method not found"},
		}
	}
}

func (s *Server) handleInitialize(req *Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: InitializeResult{
			ProtocolVersion: ProtocolVersion,
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{},
			},
			ServerInfo: ServerInfo{
				Name:    ServerName,
				Version: ServerVersion,
			},
		},
	}
}

func (s *Server) handleToolsList(req *Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ToolsListResult{Tools: s.tools},
	}
}

func (s *Server) handleToolsCall(req *Request) *Response {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &Error{Code: -32602, Message: "Invalid params"},
		}
	}

	result, err := s.callTool(params.Name, params.Arguments)
	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentBlock{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			},
		}
	}

	// Convert result to JSON text
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: CallToolResult{
			Content: []ContentBlock{{Type: "text", Text: string(resultJSON)}},
		},
	}
}

func (s *Server) callTool(name string, args map[string]interface{}) (interface{}, error) {
	switch name {
	case "list_boards":
		return s.client.ListBoards()

	case "list_tasks":
		boardID, _ := args["board_id"].(string)
		status, _ := args["status"].(string)
		assignee, _ := args["assignee"].(string)
		return s.client.ListTasks(boardID, status, assignee)

	case "get_task":
		id, ok := args["id"].(string)
		if !ok || id == "" {
			return nil, fmt.Errorf("id is required")
		}
		return s.client.GetTask(id)

	case "create_task":
		boardID, _ := args["board_id"].(string)
		name, _ := args["name"].(string)
		if boardID == "" || name == "" {
			return nil, fmt.Errorf("board_id and name are required")
		}
		return s.client.CreateTask(taskboard.CreateTaskParams{
			BoardID:     boardID,
			Name:        name,
			Description: getString(args, "description"),
			Assignee:    getString(args, "assignee"),
			Status:      getString(args, "status"),
			Priority:    getInt(args, "priority"),
		})

	case "update_task":
		id, _ := args["id"].(string)
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		return s.client.UpdateTask(id, taskboard.UpdateTaskParams{
			Name:        getStringPtr(args, "name"),
			Description: getStringPtr(args, "description"),
			Assignee:    getStringPtr(args, "assignee"),
			Status:      getStringPtr(args, "status"),
			Priority:    getIntPtr(args, "priority"),
		})

	case "add_comment":
		taskID, _ := args["task_id"].(string)
		text, _ := args["text"].(string)
		if taskID == "" || text == "" {
			return nil, fmt.Errorf("task_id and text are required")
		}
		return s.client.AddComment(taskID, text)

	case "list_users":
		return s.client.ListUsers()

	case "create_user":
		name, _ := args["name"].(string)
		email, _ := args["email"].(string)
		if name == "" || email == "" {
			return nil, fmt.Errorf("name and email are required")
		}
		return s.client.CreateUser(taskboard.CreateUserParams{
			Name:    name,
			Email:   email,
			Avatar:  getString(args, "avatar"),
			IsAgent: getBool(args, "is_agent"),
		})

	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func getString(args map[string]interface{}, key string) string {
	v, _ := args[key].(string)
	return v
}

func getStringPtr(args map[string]interface{}, key string) *string {
	if v, ok := args[key].(string); ok {
		return &v
	}
	return nil
}

func getInt(args map[string]interface{}, key string) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getBool(args map[string]interface{}, key string) bool {
	v, _ := args[key].(bool)
	return v
}

func getIntPtr(args map[string]interface{}, key string) *int {
	if v, ok := args[key].(float64); ok {
		i := int(v)
		return &i
	}
	return nil
}
