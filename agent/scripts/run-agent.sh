#!/bin/bash

# Navigate to the agent directory (parent of scripts)
cd "$(dirname "$0")/.." || exit 1

# Activate the virtual environment
source .venv/bin/activate

# Run the agent
uv run agent.py

