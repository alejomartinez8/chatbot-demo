# Go ADK Agent with AG-UI Integration

A Go-based agent using Google's Agent Development Kit (ADK) with AG-UI protocol support. This agent provides time information for cities worldwide and integrates seamlessly with the React frontend.

## Features

- 🤖 **Google ADK Integration** - Uses the official ADK Go library
- 🔌 **AG-UI Protocol Support** - HTTP/SSE communication with frontend
- ⏰ **Time Agent** - Tells the current time in specified cities
- 🔍 **Google Search Tool** - Uses Google Search for location/timezone lookup
- 💨 **Streaming Responses** - Real-time streaming via Server-Sent Events

## Prerequisites

- **Go 1.24.4 or later** - [Download Go](https://go.dev/dl/)
- **Google API Key** - Get one from [Google AI Studio](https://aistudio.google.com/apikey)
- **ADK Go v0.2.0 or later** - Automatically installed via `go mod`
- **Reflex (optional)** - Auto-reload tool, installed automatically by setup scripts

## Quick Start

### 1. Set Up the Agent

Run the setup script to install dependencies:

```bash
cd agent-go-ag-ui
./scripts/setup-agent-go.sh
```

Or manually:

```bash
cd agent-go-ag-ui
go mod download
go mod tidy
```

### 2. Configure API Key

Create a `.env` file in the `agent-go-ag-ui/` directory:

```bash
echo 'export GOOGLE_API_KEY="your_actual_api_key_here"' > agent-go-ag-ui/.env
```

Or set it as an environment variable:

```bash
export GOOGLE_API_KEY="your_actual_api_key_here"
```

### 3. Run the Agent

```bash
cd agent-go-ag-ui
./scripts/run-agent-go.sh
```

The agent will start on `http://localhost:8000` (or the port specified in the `PORT` environment variable).

**🔄 Auto-Reload**: The run scripts automatically use `reflex` to restart the agent when you make changes to `.go` files. If `reflex` is not installed, the scripts will fall back to `go run .` (without auto-reload).

To install `reflex` manually:
```bash
go install github.com/cespare/reflex@latest
# Make sure $GOPATH/bin is in your PATH
```

Or run without auto-reload:
```bash
cd agent-go-ag-ui
go run .
```

### 4. Connect the Frontend

The React frontend is already configured to connect to the agent. Just start the frontend:

```bash
yarn dev
```

Visit `http://localhost:5173` and start chatting with the agent!

## Project Structure

This project follows the [Standard Go Project Layout](https://github.com/golang-standards/project-layout):

```
agent-go-ag-ui/
├── cmd/
│   └── agent/
│       └── main.go              # Application entry point
├── internal/
│   ├── agent/                   # Agent creation and configuration
│   ├── config/                  # Configuration management
│   ├── handler/                 # HTTP handler for AG-UI protocol
│   ├── server/                  # HTTP server setup and lifecycle
│   ├── session/                 # Session management
│   └── stream/                  # Agent response streaming
├── scripts/                     # Build and run scripts
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
└── README.md                    # This file
```

## How It Works

### Agent Implementation

The agent is created using `llmagent.New()` with:
- **Model**: `gemini-3-pro-preview` (latest Gemini model)
- **Tools**: Google Search for location/timezone information
- **Instruction**: "You are a helpful assistant that tells the current time in a city."

### AG-UI Protocol

The server implements the AG-UI protocol:
- **Endpoint**: `POST /` (root path)
- **Protocol**: HTTP with Server-Sent Events (SSE)
- **Input**: `RunAgentInput` JSON format
- **Output**: SSE stream with AG-UI events:
  - `TEXT_MESSAGE_START` - Indicates start of response
  - `TEXT_MESSAGE_CONTENT` - Streaming text chunks (delta)
  - `TEXT_MESSAGE_END` - Indicates end of response

### Agent Execution

The agent uses ADK's `runner.Run()` method:
1. Creates an in-memory session
2. Converts user message to `genai.Content`
3. Executes the agent via the runner
4. Collects events and extracts text from responses
5. Streams the response back via SSE

## Configuration

### Environment Variables

- `GOOGLE_API_KEY` (required) - Your Google AI API key
- `PORT` (optional) - Server port (default: 8000)

### Example `.env` file:

```bash
GOOGLE_API_KEY=your_api_key_here
PORT=8000
```

## Development

### Auto-Reload Development

The agent supports automatic restart on file changes using `reflex`. When you run the agent with the provided scripts, it will:

- Watch for changes in `.go` files
- Automatically rebuild and restart the server
- Preserve environment variables and configuration

The setup scripts will automatically install `reflex` for you. If you prefer not to use auto-reload, you can run `go run .` directly.

### Building

```bash
cd agent-go-ag-ui
go build -o agent-go-ag-ui .
```

### Running Tests

```bash
cd agent-go-ag-ui
go test ./...
```

### Dependencies

Key dependencies:
- `google.golang.org/adk` - Agent Development Kit
- `google.golang.org/genai` - Gemini API client
- Standard library packages for HTTP and JSON

## Troubleshooting

### "GOOGLE_API_KEY environment variable is required"

Make sure you've set the API key:
- Create a `.env` file in `agent-go-ag-ui/`
- Or export it as an environment variable
- Get your key from [Google AI Studio](https://aistudio.google.com/apikey)

### "Failed to create model"

- Verify your API key is correct
- Check your internet connection
- Ensure you have access to the Gemini API

### Port Already in Use

Change the port:
```bash
PORT=8080 go run .
```

Or update the `.env` file:
```bash
PORT=8080
```

### Frontend Can't Connect

- Ensure the agent is running on port 8000 (or update `vite.config.js`)
- Check that CORS headers are being sent (they should be automatic)
- Verify the frontend proxy configuration

## API Reference

### POST /

Main endpoint for AG-UI protocol communication.

**Request:**
```json
{
  "threadId": "string",
  "runId": "string",
  "messages": [
    {
      "id": "string",
      "role": "user",
      "content": "What time is it in Paris?"
    }
  ],
  "state": {},
  "tools": [],
  "context": [],
  "forwardedProps": {}
}
```

**Response:** Server-Sent Events stream

```
data: {"type":"TEXT_MESSAGE_START"}
data: {"type":"TEXT_MESSAGE_CONTENT","delta":"The current time"}
data: {"type":"TEXT_MESSAGE_CONTENT","delta":" in Paris is..."}
data: {"type":"TEXT_MESSAGE_END"}
```

## License

MIT

