# Chatbot Demo with AG-UI Client

A modern React + Vite chatbot application that connects to agents supporting the AG-UI protocol.

## Features

- 🤖 **AG-UI Protocol Integration** - Connects to any AG-UI compatible agent
- 🔌 **Observable Streaming** - Real-time responses via Server-Sent Events
- 💚 **Connection Health Check** - Automatic status monitoring
- 🎨 **Modern UI** - React 19 + Tailwind CSS v4 + shadcn/ui components
- 🧩 **AI SDK Elements** - Enhanced conversation UI with auto-scroll
- ⚡ **Fast Development** - Vite with hot reload
- 💪 **TypeScript** - Full type safety

## Prerequisites

- Node.js 18+
- Google API Key for Gemini (get from [Google AI Studio](https://aistudio.google.com/apikey))

**For Python Agent:**
- Python 3.12+
- [uv](https://docs.astral.sh/uv/) - Fast Python package installer

**For Go Agent:**
- Go 1.24.4 or later - [Download Go](https://go.dev/dl/)

## Quick Start

### 1. Set Up an Agent

Choose either the **Python** or **Go** agent:

#### Option A: Python Agent (Weather)

```bash
# Run the setup script (this installs dependencies)
cd agent-python-ag-ui
./scripts/setup-agent.sh

# Create .env file in the agent directory with your Google API key
echo "GOOGLE_API_KEY=your_actual_api_key_here" > .env
```

**Windows:**
```bash
cd agent-python-ag-ui
scripts\setup-agent.bat
echo GOOGLE_API_KEY=your_actual_api_key_here > .env
```

#### Option B: Go Agent (Time)

```bash
# Run the setup script (this installs dependencies)
cd agent-go-ag-ui
./scripts/setup-agent-go.sh

# Create .env file in the agent directory with your Google API key
echo GOOGLE_API_KEY=your_actual_api_key_here > .env
```

**Windows:**
```bash
cd agent-go-ag-ui
scripts\setup-agent-go.bat
echo set GOOGLE_API_KEY=your_actual_api_key_here > .env
```

### 2. Start the Agent

#### Python Agent:
```bash
# Run the agent (starts on localhost:8000)
cd agent-python-ag-ui
./scripts/run-agent.sh
```

**Windows:**
```bash
cd agent-python-ag-ui
scripts\run-agent.bat
```

#### Go Agent:
```bash
# Run the agent (starts on localhost:8000)
cd agent-go-ag-ui
./scripts/run-agent-go.sh
```

**Windows:**
```bash
cd agent-go-ag-ui
scripts\run-agent-go.bat
```

### 3. Start the Frontend

In a separate terminal:

```bash
# Install dependencies
yarn install

# Start development server
yarn dev
```

Visit `http://localhost:5173`

## Configuration

The application proxies requests to avoid CORS issues. Update `vite.config.js` if your agent runs on a different port:

```js
server: {
  proxy: {
    '/api/agent': {
      target: 'http://localhost:8000',  // Change this if needed
      changeOrigin: true,
      rewrite: (path) => path.replace(/^\/api\/agent/, ''),
    },
  },
}
```

## Adding UI Components

This project uses [shadcn/ui](https://ui.shadcn.com/) for components and [AI SDK Elements](https://sdk.vercel.ai/docs/ai-sdk-ui/ai-sdk-elements) for conversation UI.

### Adding shadcn components:
```bash
npx shadcn@latest add <component-name>
```

### Adding AI Elements components:
```bash
npx shadcn@latest add @ai-elements/<component-name>
```

Example:
```bash
npx shadcn@latest add @ai-elements/conversation
```

## Build for Production

```bash
# Build
yarn build

# Preview
yarn preview
```

## Tech Stack

### Frontend
- React 19
- TypeScript
- Vite
- Tailwind CSS v4
- shadcn/ui - Component library
- AI SDK Elements - Conversation components
- @ag-ui/core (types for AG-UI protocol)

### Agents

**Python Agent:**
- FastAPI - Web framework
- Google ADK - Agent Development Kit
- ag-ui-adk - AG-UI protocol adapter
- Gemini 2.0 Flash - LLM model
- httpx - Async HTTP client

**Go Agent:**
- Go 1.24.4+ - Programming language
- Google ADK Go - Agent Development Kit for Go
- google.golang.org/genai - Gemini API client
- Standard library HTTP/SSE support

## Agent Features

### Python Agent (Weather)
The Python weather agent provides:
- Real-time weather information for any location
- Current temperature, feels-like temperature, humidity
- Wind speed and gust information
- Weather conditions (clear, cloudy, rain, snow, etc.)
- Powered by Open-Meteo API (no API key required for weather data)

### Go Agent (Time)
The Go time agent provides:
- Current time information for cities worldwide
- Timezone lookup using Google Search
- Real-time streaming responses via Server-Sent Events
- Powered by Gemini 3 Pro model

## License

MIT
