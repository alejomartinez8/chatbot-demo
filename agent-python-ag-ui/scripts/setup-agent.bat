@echo off

REM Navigate to the agent directory (parent of scripts)
cd /d "%~dp0\.." || exit /b 1

uv sync

