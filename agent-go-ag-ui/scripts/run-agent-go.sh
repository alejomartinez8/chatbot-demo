#!/bin/bash

# Navigate to the agent-go-ag-ui directory (parent of scripts)
cd "$(dirname "$0")/.." || exit 1

# Load environment variables from .env file if it exists
if [ -f .env ]; then
    set -a
    source .env
    set +a
fi

# Check if GOOGLE_API_KEY is set
if [ -z "$GOOGLE_API_KEY" ]; then
    echo "⚠️  Warning: GOOGLE_API_KEY environment variable not set!"
    echo "   Set it with: export GOOGLE_API_KEY='your-key-here'"
    echo "   Or create a .env file in the agent-go-ag-ui directory"
    echo "   Get a key from: https://aistudio.google.com/apikey"
    echo ""
fi

# Run the agent
go run .

