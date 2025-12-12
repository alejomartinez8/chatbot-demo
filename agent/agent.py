"""Weather Agent with AG-UI Integration.

This module implements a conversational weather assistant using Google's ADK (Agent Development Kit)
and the AG-UI protocol. The agent can provide real-time weather information for any location
worldwide using the Open-Meteo API.

Features:
    - Real-time weather data retrieval
    - Natural language conversation interface
    - AG-UI protocol support for seamless frontend integration
    - Automatic geocoding for location resolution
    - Detailed weather conditions including temperature, humidity, wind, and more

Requirements:
    - Python 3.12+
    - Google API Key (set in GOOGLE_API_KEY environment variable)
    - FastAPI and uvicorn for server
    - httpx for async HTTP requests

Usage:
    Run directly:
        $ python agent.py
    
    Or use the convenience script:
        $ ./scripts/run-agent.sh
    
    The server will start on http://0.0.0.0:8000

Environment Variables:
    GOOGLE_API_KEY: Required. Your Google AI API key for Gemini model access.
                   Get one from: https://aistudio.google.com/apikey

API Endpoint:
    POST / : Main endpoint for AG-UI protocol communication
"""

from __future__ import annotations

from fastapi import FastAPI
from ag_ui_adk import ADKAgent, add_adk_fastapi_endpoint
from google.adk.agents import LlmAgent
from google.adk import tools as adk_tools
from dotenv import load_dotenv
import httpx

# Load environment variables from .env file
load_dotenv()


def get_weather_condition(code: int) -> str:
    """Map WMO weather code to human-readable condition.
    
    The World Meteorological Organization (WMO) defines standardized weather codes
    that are used by many weather APIs. This function translates these numeric codes
    into descriptive text.

    Args:
        code: WMO weather code (integer from 0-99).

    Returns:
        Human-readable weather condition string. Returns "Unknown" if code is not recognized.
    
    Examples:
        >>> get_weather_condition(0)
        'Clear sky'
        >>> get_weather_condition(61)
        'Slight rain'
        >>> get_weather_condition(95)
        'Thunderstorm'
    
    References:
        https://www.nodc.noaa.gov/archive/arc0021/0002199/1.1/data/0-data/HTML/WMO-CODE/WMO4677.HTM
    """
    conditions = {
        0: "Clear sky",
        1: "Mainly clear",
        2: "Partly cloudy",
        3: "Overcast",
        45: "Foggy",
        48: "Depositing rime fog",
        51: "Light drizzle",
        53: "Moderate drizzle",
        55: "Dense drizzle",
        56: "Light freezing drizzle",
        57: "Dense freezing drizzle",
        61: "Slight rain",
        63: "Moderate rain",
        65: "Heavy rain",
        66: "Light freezing rain",
        67: "Heavy freezing rain",
        71: "Slight snow fall",
        73: "Moderate snow fall",
        75: "Heavy snow fall",
        77: "Snow grains",
        80: "Slight rain showers",
        81: "Moderate rain showers",
        82: "Violent rain showers",
        85: "Slight snow showers",
        86: "Heavy snow showers",
        95: "Thunderstorm",
        96: "Thunderstorm with slight hail",
        99: "Thunderstorm with heavy hail",
    }
    return conditions.get(code, "Unknown")


async def get_weather(location: str) -> dict[str, str | float]:
    """Get current weather for a location using Open-Meteo API.
    
    This function performs two API calls:
    1. Geocoding: Converts location name to latitude/longitude coordinates
    2. Weather: Fetches current weather data for those coordinates
    
    The Open-Meteo API is free and doesn't require an API key.

    Args:
        location: City or location name (e.g., "Paris", "New York", "Tokyo").
                 Can be in any language - the geocoding API supports international names.

    Returns:
        Dictionary containing weather information with the following keys:
            - temperature (float): Current temperature in Celsius
            - feelsLike (float): Apparent temperature in Celsius
            - humidity (float): Relative humidity percentage (0-100)
            - windSpeed (float): Wind speed in km/h
            - windGust (float): Wind gust speed in km/h
            - conditions (str): Human-readable weather condition
            - location (str): Resolved location name
    
    Raises:
        ValueError: If the location cannot be found by the geocoding service.
        httpx.HTTPError: If there's an error communicating with the API.
    
    Examples:
        >>> weather = await get_weather("London")
        >>> print(f"{weather['temperature']}°C in {weather['location']}")
        15.2°C in London
        
        >>> weather = await get_weather("Tokyo")
        >>> print(weather['conditions'])
        Partly cloudy
    
    API References:
        - Geocoding: https://open-meteo.com/en/docs/geocoding-api
        - Weather: https://open-meteo.com/en/docs
    """
    async with httpx.AsyncClient() as client:
        # Geocode the location
        geocoding_url = (
            f"https://geocoding-api.open-meteo.com/v1/search?name={location}&count=1"
        )
        geocoding_response = await client.get(geocoding_url)
        geocoding_data = geocoding_response.json()

        if not geocoding_data.get("results"):
            raise ValueError(f"Location '{location}' not found")

        result = geocoding_data["results"][0]
        latitude = result["latitude"]
        longitude = result["longitude"]
        name = result["name"]

        # Get weather data
        weather_url = (
            f"https://api.open-meteo.com/v1/forecast?"
            f"latitude={latitude}&longitude={longitude}"
            f"&current=temperature_2m,apparent_temperature,relative_humidity_2m,"
            f"wind_speed_10m,wind_gusts_10m,weather_code"
        )
        weather_response = await client.get(weather_url)
        weather_data = weather_response.json()

        current = weather_data["current"]

        return {
            "temperature": current["temperature_2m"],
            "feelsLike": current["apparent_temperature"],
            "humidity": current["relative_humidity_2m"],
            "windSpeed": current["wind_speed_10m"],
            "windGust": current["wind_gusts_10m"],
            "conditions": get_weather_condition(current["weather_code"]),
            "location": name,
        }


# ============================================================================
# Agent Configuration
# ============================================================================

# Create the LLM agent with weather capabilities
sample_agent = LlmAgent(
    name="assistant",
    model="gemini-2.5-flash",  # Using the latest Gemini model
    instruction="""
      You are a helpful weather assistant that provides accurate weather information.

      Your primary function is to help users get weather details for specific locations. When responding:
      - Always ask for a location if none is provided
      - If the location name isn't in English, please translate it
      - If giving a location with multiple parts (e.g. "New York, NY"), use the most relevant part (e.g. "New York")
      - Include relevant details like humidity, wind conditions, and precipitation
      - Keep responses concise but informative

      Use the get_weather tool to fetch current weather data.
      """,
    tools=[
        adk_tools.preload_memory_tool.PreloadMemoryTool(),  # Enables conversation memory
        get_weather,  # Weather data retrieval tool
    ],
)

# Wrap the ADK agent with AG-UI protocol support
# This enables communication with AG-UI compatible frontends via Server-Sent Events
chat_agent = ADKAgent(
    adk_agent=sample_agent,
    user_id="demo_user",  # User identifier for session management
    session_timeout_seconds=3600,  # Session expires after 1 hour of inactivity
    use_in_memory_services=True,  # Use in-memory storage (no database required)
)

# Create FastAPI application
app = FastAPI(
    title="ADK Middleware Weather Agent",
    description="A weather assistant agent using Google ADK and AG-UI protocol",
    version="1.0.0",
)

# Register the AG-UI endpoint at root path
# This endpoint handles all agent communication via POST requests
add_adk_fastapi_endpoint(app, chat_agent, path="/")

# ============================================================================
# Server Entry Point
# ============================================================================

if __name__ == "__main__":
    import os
    import uvicorn

    # Validate that required environment variables are set
    if not os.getenv("GOOGLE_API_KEY"):
        print("⚠️  Warning: GOOGLE_API_KEY environment variable not set!")
        print("   Set it with: export GOOGLE_API_KEY='your-key-here'")
        print("   Or create a .env file in the agent directory")
        print("   Get a key from: https://aistudio.google.com/apikey")
        print()

    # Start the FastAPI server
    # The agent will be accessible at http://localhost:8000
    uvicorn.run(
        app,
        host="0.0.0.0",  # Listen on all network interfaces
        port=8000,  # Default port for the agent
        log_level="info",  # Set logging level
    )

