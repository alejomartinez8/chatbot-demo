#!/bin/bash

# Navigate to the agent directory (parent of scripts)
cd "$(dirname "$0")/.." || exit 1

uv sync

