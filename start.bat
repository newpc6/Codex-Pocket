@echo off
cd /d %~dp0

echo [1/2] Building...
go build -o codexpocket.exe ./cmd/codexpocket-agent
if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo [2/2] Starting...
codexpocket.exe
