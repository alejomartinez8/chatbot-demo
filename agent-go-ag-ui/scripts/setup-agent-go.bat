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

echo ✅ Setup complete!
echo.
echo Next steps:
echo 1. Create a .env file in agent-go-ag-ui/ with your GOOGLE_API_KEY
echo 2. Run the agent with: scripts\run-agent-go.bat
echo.

