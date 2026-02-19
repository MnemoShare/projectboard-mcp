# TaskBoard MCP

MCP (Model Context Protocol) server for [TaskBoard](https://github.com/MnemoShare/projectboard) — enables AI assistants like Claude Code, Cursor, and others to manage tasks directly.

## Installation

### From Source

```bash
go install github.com/MnemoShare/projectboard-mcp@latest
```

### Pre-built Binaries

Download from [Releases](https://github.com/MnemoShare/projectboard-mcp/releases).

## Configuration

### Option 1: Environment Variables (Recommended)

```bash
export TASKBOARD_URL="https://planning.mnemoshare.com"
export TASKBOARD_TOKEN="your-api-token"
```

### Option 2: Config File

Create `~/.config/taskboard-mcp/config.json`:

```json
{
  "url": "https://planning.mnemoshare.com",
  "token": "your-api-token"
}
```

### Getting Your API Token

1. Log into TaskBoard
2. Go to Settings → API Tokens
3. Create a new token

## Usage with Claude Code

Add to your Claude Code MCP config (`~/.claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "taskboard": {
      "command": "taskboard-mcp",
      "env": {
        "TASKBOARD_URL": "https://planning.mnemoshare.com",
        "TASKBOARD_TOKEN": "your-api-token"
      }
    }
  }
}
```

## Available Tools

| Tool | Description |
|------|-------------|
| `list_boards` | List all boards |
| `list_tasks` | List tasks with optional filters (board, status, assignee) |
| `get_task` | Get a task by ID or ticket number (e.g., `MNS-42`) |
| `create_task` | Create a new task |
| `update_task` | Update a task (status, assignee, title, etc.) |
| `add_comment` | Add a comment to a task |
| `list_users` | List team members for assignment |

### Status Values

The following status values are supported:
- `backlog` - Not ready for work
- `todo` - Ready to pick up
- `in-progress` - Currently working on it
- `in-qa` - Ready for QA review
- `completed` - QA passed
- `rfp` - Ready for production
- `closed` - Released

## Examples

### List tasks assigned to you

```
list_tasks(assignee: "julia@mnemoshare.com", status: "todo")
```

### Update task status

```
update_task(id: "MNS-42", status: "completed")
```

### Create a new task

```
create_task(
  board_id: "...",
  title: "Implement user authentication",
  description: "Add login/logout functionality",
  assignee: "derrick@mnemoshare.com",
  priority: 2
)
```

## Development

```bash
# Build
go build -o taskboard-mcp .

# Test locally
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./taskboard-mcp
```

## License

MIT
