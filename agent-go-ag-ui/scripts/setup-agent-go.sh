#!/bin/bash

# Navigate to the agent-go-ag-ui directory (parent of scripts)
cd "$(dirname "$0")/.." || exit 1

echo "Setting up Go ADK Agent..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed or not in PATH"
    echo "   Please install Go 1.24.4 or later from: https://go.dev/dl/"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Found Go version: $GO_VERSION"

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Tidy up the module
echo "Tidying up module..."
go mod tidy

# Install reflex for auto-reload (optional but recommended)
echo "Installing reflex (live reload tool)..."
if ! command -v reflex &> /dev/null; then
    go install github.com/cespare/reflex@latest
    if [ $? -eq 0 ]; then
        echo "✅ reflex installed successfully"
        echo "   Make sure $(go env GOPATH)/bin is in your PATH"
    else
        echo "⚠️  Failed to install reflex. You can install it manually later with:"
        echo "   go install github.com/cespare/reflex@latest"
    fi
else
    echo "✅ reflex is already installed"
fi

echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Create a .env file in agent-go-ag-ui/ with your GOOGLE_API_KEY"
echo "2. Run the agent with: ./scripts/run-agent-go.sh"
echo "   (The agent will auto-reload when you make changes to .go files if reflex is installed)"
echo ""

