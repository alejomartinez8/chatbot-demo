@echo off

REM Navigate to the agent directory (parent of scripts)
cd /d "%~dp0\.." || exit /b 1

REM Activate the virtual environment
call .venv\Scripts\activate

REM Run the agent
uv run agent.py

