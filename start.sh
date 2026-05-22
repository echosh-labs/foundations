#!/usr/bin/env bash

# Antigravity Foundations Orchestrator - Startup Script
set -e

# Curated HSL tailormade colors for visual presentation in the console
VIOLET='\033[0;35m'
CYAN='\033[0;36m'
PEACH='\033[0;31m'
GRAY='\033[0;90m'
NC='\033[0m' # No Color

echo -e "${VIOLET}"
echo "    _   _  _ _____ ___ ___  ___  _ __   _____ _   _"
echo "   /_\ | \| |_   _|_ _/ __|| _ \/_\ \ \ / /_ _|_| |_|"
echo "  / _ \| .\` | | |  | | (_ ||   / _ \ \ V / | | |_   _|"
echo " /_/ \_\_|\_| |_| |___\___||_|_/_/ \_\ \_/ |___|  |_|"
echo -e "          Cognitive Interpretation Engine${NC}\n"

echo -e "${GRAY}-------------------------------------------------------${NC}"
echo -e "Starting Antigravity Foundations telemetry hub..."

# Navigate to project home
PROJECT_DIR="/home/justin/code/echosh-labs/foundations"
cd "$PROJECT_DIR"

# Cleanup stale socket
if [ -S "/tmp/foundations.sock" ]; then
    echo -e "${GRAY}Removing stale socket at /tmp/foundations.sock...${NC}"
    rm -f "/tmp/foundations.sock"
fi

# 1. Start Telemetry Socket Server in background
./foundations > /dev/null 2>&1 &
TELEMETRY_PID=$!

# 2. Start Web Server in background
python3 -m http.server 8000 --directory ./web-app > /dev/null 2>&1 &
WEB_PID=$!

# Define cleanup trap to kill background processes on exit
cleanup() {
    echo -e "\n${PEACH}Shutting down foundations environment...${NC}"
    kill "$TELEMETRY_PID" "$WEB_PID" 2>/dev/null || true
    echo -e "${CYAN}Foundations offline. Goodbye!${NC}"
}
trap cleanup EXIT INT TERM

# Wait for socket to bind
sleep 1.2

echo -e "${CYAN}● Telemetry Server  : ACTIVE  (Socket: /tmp/foundations.sock)${NC}"
echo -e "${CYAN}● Storyboard Web App: ONLINE  (Link: http://localhost:8000)${NC}"
echo -e "${GRAY}-------------------------------------------------------${NC}"
echo -e "Launching Bubble Tea TUI dashboard...\n"

# 3. Launch TUI in foreground
if [ -f "./tui-app/axis-mundi-tui" ]; then
    ./tui-app/axis-mundi-tui
else
    echo -e "${PEACH}Error: axis-mundi-tui binary not found. Compiling now...${NC}"
    cd tui-app && go build -o axis-mundi-tui main.go
    ./axis-mundi-tui
fi
