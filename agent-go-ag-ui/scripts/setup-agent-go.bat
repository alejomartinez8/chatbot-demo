@echo off

REM Navigate to the agent-go-ag-ui directory (parent of scripts)
cd /d "%~dp0\.." || exit /b 1

echo Setting up Go ADK Agent...

REM Check if Go is installed
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo ❌ Error: Go is not installed or not in PATH
    echo    Please install Go 1.24.4 or later from: https://go.dev/dl/
    exit /b 1
)

REM Check Go version
go version

REM Download dependencies
echo Downloading dependencies...
go mod download

REM Tidy up the module
echo Tidying up module...
go mod tidy

REM Install reflex for auto-reload (optional but recommended)
echo Installing reflex (live reload tool)...
where reflex >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    go install github.com/cespare/reflex@latest
    if %ERRORLEVEL% EQU 0 (
        echo ✅ reflex installed successfully
        echo    Make sure %GOPATH%\bin is in your PATH
    ) else (
        echo ⚠️  Failed to install reflex. You can install it manually later with:
        echo    go install github.com/cespare/reflex@latest
    )
) else (
    echo ✅ reflex is already installed
)

echo ✅ Setup complete!
echo.
echo Next steps:
echo 1. Create a .env file in agent-go-ag-ui/ with your GOOGLE_API_KEY
echo 2. Run the agent with: scripts\run-agent-go.bat
echo    (The agent will auto-reload when you make changes to .go files if reflex is installed)
echo.

