# Weather Agent

A conversational AI weather assistant built with Google's Agent Development Kit (ADK) and the AG-UI protocol.

## Overview

This agent provides real-time weather information through natural language conversation. It uses:

- **Google Gemini 2.5 Flash** - Latest LLM for natural conversations
- **Open-Meteo API** - Free weather data (no API key needed)
- **AG-UI Protocol** - Standardized agent communication via Server-Sent Events
- **FastAPI** - High-performance async web framework

## Features

### Weather Information
- Current temperature and "feels like" temperature
- Humidity levels
- Wind speed and gusts
- Weather conditions (clear, cloudy, rain, snow, etc.)
- Automatic location geocoding (supports international names)

### Conversation Capabilities
- Natural language understanding
- Multi-turn conversations with memory
- Handles ambiguous locations
- Asks for clarification when needed

## Architecture

```
┌─────────────┐      AG-UI Protocol      ┌──────────────┐
│   Frontend  │ ◄──────────────────────► │ FastAPI      │
│  (React)    │   Server-Sent Events     │ Server       │
└─────────────┘                          └──────┬───────┘
                                                 │
                                          ┌──────▼───────┐
                                          │ ADK Agent    │
                                          │ (AG-UI)      │
                                          └──────┬───────┘
                                                 │
                              ┌──────────────────┼──────────────────┐
                              │                  │                  │
                       ┌──────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐
                       │ Gemini 2.5   │  │ Memory Tool  │  │ Weather Tool │
                       │ Flash (LLM)  │  │              │  │ (Open-Meteo) │
                       └──────────────┘  └──────────────┘  └──────────────┘
```

## Setup

### Prerequisites

- Python 3.12 or higher
- [uv](https://docs.astral.sh/uv/) - Fast Python package installer
- Google API Key for Gemini

### Installation

1. **Install dependencies:**
   ```bash
   uv sync
   ```

2. **Set up environment variables:**
   ```bash
   # Create .env file
   echo "GOOGLE_API_KEY=your_api_key_here" > .env
   ```
   
   Get your API key from: https://aistudio.google.com/apikey

### Running the Agent

**Option 1: Using the convenience script (recommended)**
```bash
# From agent directory
cd agent
./scripts/run-agent.sh
```

**Option 2: Direct execution**
```bash
# From agent directory
uv run agent.py
```

**Option 3: Using activated venv**
```bash
source .venv/bin/activate
python agent.py
```

The agent will start on `http://0.0.0.0:8000`

## Usage Examples

### Via Frontend

Start the React frontend (from project root):
```bash
yarn dev
```

Then chat naturally:
- "What's the weather in Paris?"
- "How's the weather in Tokyo today?"
- "Tell me about the weather in New York"

### Via API (curl)

```bash
# Start a conversation
curl -X POST http://localhost:8000/ \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "What is the weather in London?"}]
  }'
```

## Configuration

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GOOGLE_API_KEY` | Yes | Your Google AI API key for Gemini model access |

### Agent Parameters

Edit `agent.py` to customize:

```python
sample_agent = LlmAgent(
    name="assistant",           # Agent name
    model="gemini-2.5-flash",   # LLM model to use
    instruction="...",          # System instructions
    tools=[...],                # Available tools
)

chat_agent = ADKAgent(
    adk_agent=sample_agent,
    user_id="demo_user",                # User identifier
    session_timeout_seconds=3600,        # 1 hour session timeout
    use_in_memory_services=True,        # Use memory (no DB required)
)
```

### Available Models

- `gemini-2.5-flash` (default) - Latest, fastest, most capable
- `gemini-2.0-flash` - Previous generation
- `gemini-1.5-flash` - Older version
- `gemini-1.5-pro` - More capable but slower

## Development

### Project Structure

```
agent/
├── agent.py          # Main agent implementation
├── pyproject.toml    # Python dependencies
├── .env             # Environment variables (gitignored)
├── .gitignore       # Git ignore rules
└── README.md        # This file
```

### Key Components

**`get_weather_condition(code: int)`**
- Maps WMO weather codes to human-readable strings
- Used internally by `get_weather`

**`get_weather(location: str)`**
- Async function that fetches weather data
- Performs geocoding + weather data retrieval
- Returns structured dictionary with all weather info

**`sample_agent`**
- LlmAgent instance with weather capabilities
- Configured with system instructions and tools

**`chat_agent`**
- ADKAgent wrapper that adds AG-UI protocol support
- Manages sessions and state

**`app`**
- FastAPI application instance
- Serves the AG-UI endpoint at `/`

### Adding New Tools

To add capabilities beyond weather:

1. **Define the tool function:**
   ```python
   async def my_tool(param: str) -> dict:
       """Tool description for the LLM."""
       # Tool implementation
       return {"result": "value"}
   ```

2. **Add to agent's tools list:**
   ```python
   sample_agent = LlmAgent(
       # ...
       tools=[
           adk_tools.preload_memory_tool.PreloadMemoryTool(),
           get_weather,
           my_tool,  # Add your tool here
       ],
   )
   ```

3. **Update the system instructions:**
   ```python
   instruction="""
       You are a helpful assistant.
       
       Use get_weather for weather information.
       Use my_tool for [describe when to use it].
       """
   ```

## API Reference

### Open-Meteo APIs

**Geocoding API**
- Endpoint: `https://geocoding-api.open-meteo.com/v1/search`
- Docs: https://open-meteo.com/en/docs/geocoding-api
- Free, no API key required

**Weather API**
- Endpoint: `https://api.open-meteo.com/v1/forecast`
- Docs: https://open-meteo.com/en/docs
- Free, no API key required

### AG-UI Protocol

**Endpoint:** `POST /`

**Request Format:**
```json
{
  "messages": [
    {"role": "user", "content": "What's the weather in Paris?"}
  ]
}
```

**Response:** Server-Sent Events stream with agent responses and tool calls

## Troubleshooting

### "GOOGLE_API_KEY environment variable not set"

**Solution:** Create a `.env` file with your API key:
```bash
echo "GOOGLE_API_KEY=your_key_here" > .env
```

### "Location not found"

**Causes:**
- Typo in location name
- Very obscure location not in geocoding database

**Solution:** Try:
- More well-known nearby city
- Include country name: "Springfield, USA"

### Port 8000 already in use

**Solution:** Either:
- Stop the other service using port 8000
- Change the port in `agent.py`:
  ```python
  uvicorn.run(app, host="0.0.0.0", port=8001)
  ```

### Import errors after updating dependencies

**Solution:** Resync the virtual environment:
```bash
uv sync --reinstall
```

## Performance

- **Cold start:** ~2-3 seconds (first request)
- **Typical response:** 1-3 seconds
- **Memory usage:** ~200-300 MB
- **Concurrent requests:** Supports multiple simultaneous conversations

## License

MIT License - see project root for details

## Resources

- [Google ADK Documentation](https://github.com/google/adk)
- [AG-UI Protocol Specification](https://github.com/google/ag-ui)
- [Open-Meteo API](https://open-meteo.com/)
- [FastAPI Documentation](https://fastapi.tiangolo.com/)

