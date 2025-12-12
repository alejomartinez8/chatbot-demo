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
- Python 3.12+
- [uv](https://docs.astral.sh/uv/) - Fast Python package installer
- Google API Key for Gemini

## Quick Start

### 1. Set Up the Agent

First, set up the Python weather agent:

```bash
# Run the setup script (this installs dependencies)
./scripts/setup-agent.sh

# Create .env file in the agent directory with your Google API key
echo "GOOGLE_API_KEY=your_actual_api_key_here" > agent/.env
```

Get your API key from: https://aistudio.google.com/apikey

### 2. Start the Agent

```bash
# Run the agent (starts on localhost:8000)
./scripts/run-agent.sh
```

Or if you're on Windows:

```bash
scripts\setup-agent.bat
scripts\run-agent.bat
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

### Agent (Python)
- FastAPI - Web framework
- Google ADK - Agent Development Kit
- ag-ui-adk - AG-UI protocol adapter
- Gemini 2.0 Flash - LLM model
- httpx - Async HTTP client

## Agent Features

The included weather agent provides:
- Real-time weather information for any location
- Current temperature, feels-like temperature, humidity
- Wind speed and gust information
- Weather conditions (clear, cloudy, rain, snow, etc.)
- Powered by Open-Meteo API (no API key required for weather data)

## License

MIT
