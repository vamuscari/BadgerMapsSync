#!/bin/bash

# Build and run the mock BadgerMaps API server

echo "Building mock server..."
go build -o mock_server mock_server.go

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo "Starting mock server on port 8080..."
    echo "Press Ctrl+C to stop the server"
    echo ""
    ./mock_server 8080
else
    echo "Build failed!"
    exit 1
fi 