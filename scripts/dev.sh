#!/bin/bash
# Development script to run both Go server and React dev server concurrently

set -e

# Trap to kill background processes on exit
trap 'kill 0' EXIT

# Start Go server in background
echo "Starting Go server..."
cd server
go run ./cmd/calendarapp &
GO_PID=$!
cd ..

# Give Go server time to start
sleep 2

# Start React dev server in foreground
echo "Starting React dev server..."
cd web
pnpm run dev
