@echo off

REM Navigate to the agent-go-ag-ui directory (parent of scripts)
cd /d "%~dp0\.." || exit /b 1

REM Load environment variables from .env file if it exists
if exist .env (
    for /f "usebackq tokens=1,* delims==" %%a in (".env") do (
        set "%%a=%%b"
    )
)

REM Check if GOOGLE_API_KEY is set
if "%GOOGLE_API_KEY%"=="" (
    echo ⚠️  Warning: GOOGLE_API_KEY environment variable not set!
    echo    Set it with: set GOOGLE_API_KEY=your-key-here
    echo    Or create a .env file in the agent-go-ag-ui directory
    echo    Get a key from: https://aistudio.google.com/apikey
    echo.
)

REM Check if reflex is installed, if not use go run
where reflex >nul 2>nul
if %ERRORLEVEL% EQU 0 (
    echo 🔄 Starting agent with auto-reload (reflex)...
    echo    The agent will automatically restart when you make changes to .go files
    echo.
    reflex -r "\.go$" -s -- go run .
) else (
    echo 💡 Tip: Install 'reflex' for auto-reload on file changes:
    echo    go install github.com/cespare/reflex@latest
    echo    Make sure %%GOPATH%%\bin is in your PATH
    echo.
    echo Starting agent (no auto-reload)...
    go run .
)

