# Go ADK Agent with AG-UI Integration

A Go-based agent using Google's Agent Development Kit (ADK) with AG-UI protocol support. This agent provides time information for cities worldwide and integrates seamlessly with the React frontend.

## Features

- 🤖 **Google ADK Integration** - Uses the official ADK Go library
- 🔌 **AG-UI Protocol Support** - Dual transport support (SSE and Connect RPC)
- 🚀 **Connect RPC** - Type-safe RPC with Protocol Buffers (gRPC-compatible)
- 💨 **Server-Sent Events** - Real-time streaming via SSE
- ⏰ **Time Agent** - Tells the current time in specified cities
- 🔍 **Google Search Tool** - Uses Google Search for location/timezone lookup

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
│   ├── agui/                    # AG-UI protocol implementation
│   │   ├── handler.go          # HTTP handler for SSE endpoint
│   │   ├── connect_handler.go  # Connect RPC handler
│   │   ├── streamer.go         # Agent response streaming
│   │   ├── state.go            # State management per thread
│   │   └── types.go            # AG-UI protocol types (RunAgentInput, etc.)
│   ├── config/                  # Configuration management
│   ├── server/                  # HTTP server setup and lifecycle
│   └── session/                 # Session management (ADK)
├── proto/                       # Protocol Buffer definitions
│   └── agui/
│       └── v1/
│           └── agui.proto      # Service and message definitions
├── gen/                         # Generated code (from protobuf)
│   └── proto/
│       └── agui/
│           └── v1/
│               ├── agui.pb.go  # Generated message types
│               └── aguiv1connect/
│                   └── agui.connect.go  # Generated Connect handlers
├── docs/                        # Documentation
│   └── CONNECT_RPC.md          # Connect RPC implementation guide
├── scripts/                     # Build and run scripts
├── buf.yaml                     # Buf configuration
├── buf.gen.yaml                 # Buf code generation config
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

The server implements the AG-UI protocol following the [official specification](https://docs.ag-ui.com) with **dual transport support**:

#### Endpoints

- **`POST /sse`** - Server-Sent Events transport (HTTP/1.1 + SSE)
- **`POST /connect`** - Connect RPC transport (HTTP/1.1, HTTP/2, gRPC-compatible)

Both endpoints support the same AG-UI protocol events:
- `STATE_SNAPSHOT` - Complete state sent on initial connection (when messages array is empty)
- `RUN_STARTED` - Indicates start of agent execution
- `TEXT_MESSAGE_START` - Indicates start of response
- `TEXT_MESSAGE_CONTENT` - Streaming text chunks (delta)
- `TEXT_MESSAGE_END` - Indicates end of response
- `RUN_FINISHED` - Indicates completion of agent execution
- `RUN_ERROR` - Error event if execution fails
- `TOOL_CALL_START`, `TOOL_CALL_ARGS`, `TOOL_CALL_RESULT`, `TOOL_CALL_END` - Tool execution events

**Input**: `RunAgentInput` JSON format (same for both endpoints)

**Output**: 
- SSE endpoint: Server-Sent Events stream
- Connect RPC endpoint: Protobuf stream (JSON or binary)

> 📖 **For detailed information about the Connect RPC implementation, see [docs/CONNECT_RPC.md](docs/CONNECT_RPC.md)**

### Type Safety

The implementation uses official AG-UI SDK types:
- `Message` type from `github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events`
- All events follow the official AG-UI protocol specification
- State management uses `STATE_SNAPSHOT` for initial synchronization according to [AG-UI State Management](https://docs.ag-ui.com/concepts/state)

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
- `github.com/ag-ui-protocol/ag-ui/sdks/community/go` - Official AG-UI Go SDK
- `connectrpc.com/connect` - Connect RPC library
- `google.golang.org/protobuf` - Protocol Buffers for Go
- Standard library packages for HTTP and JSON

### Code Generation

The project uses Protocol Buffers and Buf for code generation:

```bash
# Install code generation tools
go install github.com/bufbuild/buf/cmd/buf@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest

# Generate code from .proto files
buf generate
```

Generated code is stored in `gen/` directory (gitignored).

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

## Documentation

- **[Connect RPC Implementation Guide](docs/CONNECT_RPC.md)** - Comprehensive guide to the Connect RPC implementation, including architecture, data flow, and implementation details.

## API Reference

### POST /sse

Server-Sent Events endpoint for AG-UI protocol communication.

### POST /connect

Connect RPC endpoint for AG-UI protocol communication (gRPC-compatible).

### POST / (Legacy)

Legacy endpoint (redirects to `/sse`). Use `/sse` or `/connect` explicitly.

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

**Initial connection (empty messages array):**
```
data: {"type":"STATE_SNAPSHOT","snapshot":{...}}
```

**Agent execution:**
```
data: {"type":"RUN_STARTED","threadId":"...","runId":"..."}
data: {"type":"TEXT_MESSAGE_START","messageId":"...","role":"assistant"}
data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"...","delta":"The current time"}
data: {"type":"TEXT_MESSAGE_CONTENT","messageId":"...","delta":" in Paris is..."}
data: {"type":"TEXT_MESSAGE_END","messageId":"..."}
data: {"type":"RUN_FINISHED","threadId":"...","runId":"..."}
```

## License

MIT

