package taskboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

type Config struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// NewClientFromEnv creates a client from environment variables or config file
func NewClientFromEnv() (*Client, error) {
	// Try environment variables first
	baseURL := os.Getenv("TASKBOARD_URL")
	token := os.Getenv("TASKBOARD_TOKEN")

	// Fall back to config file
	if baseURL == "" || token == "" {
		cfg, err := loadConfig()
		if err == nil {
			if baseURL == "" {
				baseURL = cfg.URL
			}
			if token == "" {
				token = cfg.Token
			}
		}
	}

	if baseURL == "" {
		return nil, fmt.Errorf("TASKBOARD_URL not set (env or config)")
	}
	if token == "" {
		return nil, fmt.Errorf("TASKBOARD_TOKEN not set (env or config)")
	}

	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		token:   token,
		http:    &http.Client{},
	}, nil
}

func loadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(home, ".config", "taskboard-mcp", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Client) request(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Board types
type Board struct {
	ID           string `json:"_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	TicketPrefix string `json:"ticketPrefix"`
}

// Task types
type Task struct {
	ID           string   `json:"_id"`
	TicketNumber string   `json:"ticketNumber"`
	BoardID      string   `json:"boardId"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	Priority     int      `json:"priority"`
	Assignee     string   `json:"assignee"`
	Tags         []string `json:"tags"`
}

type User struct {
	ID      string `json:"_id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	IsAgent bool   `json:"isAgent"`
}

// API methods

func (c *Client) ListBoards() ([]Board, error) {
	data, err := c.request("GET", "/api/boards", nil)
	if err != nil {
		return nil, err
	}

	var boards []Board
	if err := json.Unmarshal(data, &boards); err != nil {
		return nil, err
	}

	return boards, nil
}

func (c *Client) ListTasks(boardID, status, assignee string) ([]Task, error) {
	params := url.Values{}
	if boardID != "" {
		params.Set("boardId", boardID)
	}
	if status != "" {
		params.Set("status", status)
	}
	if assignee != "" {
		params.Set("assignee", assignee)
	}

	path := "/api/tasks"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	data, err := c.request("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (c *Client) GetTask(idOrTicket string) (*Task, error) {
	// Try ticket number first (e.g., MNS-42)
	if strings.Contains(idOrTicket, "-") {
		data, err := c.request("GET", "/api/tasks/by-ticket/"+idOrTicket, nil)
		if err == nil {
			var task Task
			if json.Unmarshal(data, &task) == nil {
				return &task, nil
			}
		}
	}

	// Fall back to ID
	data, err := c.request("GET", "/api/tasks/"+idOrTicket, nil)
	if err != nil {
		return nil, err
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

type CreateTaskParams struct {
	BoardID     string `json:"boardId"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Assignee    string `json:"assignee,omitempty"`
	Status      string `json:"status,omitempty"`
	Priority    int    `json:"priority,omitempty"`
}

func (c *Client) CreateTask(params CreateTaskParams) (*Task, error) {
	data, err := c.request("POST", "/api/tasks", params)
	if err != nil {
		return nil, err
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

type UpdateTaskParams struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Assignee    *string `json:"assignee,omitempty"`
	Status      *string `json:"status,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
}

func (c *Client) UpdateTask(idOrTicket string, params UpdateTaskParams) (*Task, error) {
	// Resolve ticket number to ID if needed
	task, err := c.GetTask(idOrTicket)
	if err != nil {
		return nil, err
	}

	_, err = c.request("PUT", "/api/tasks/"+task.ID, params)
	if err != nil {
		return nil, err
	}

	// Fetch updated task
	return c.GetTask(task.ID)
}

type AddCommentParams struct {
	Text string `json:"text"`
}

func (c *Client) AddComment(idOrTicket, text string) (map[string]interface{}, error) {
	// Resolve ticket number to ID if needed
	task, err := c.GetTask(idOrTicket)
	if err != nil {
		return nil, err
	}

	data, err := c.request("POST", "/api/tasks/"+task.ID+"/comments", AddCommentParams{Text: text})
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result, nil
}

func (c *Client) ListUsers() ([]User, error) {
	data, err := c.request("GET", "/api/users", nil)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, err
	}

	return users, nil
}
