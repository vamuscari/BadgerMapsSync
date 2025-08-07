#!/bin/bash

# Mock Server Build Script
# Build and test the mock server for BadgerMaps API testing

echo "🔧 Building Mock Server for BadgerMaps API Testing"
echo "=================================================="

# Build the mock server
echo "📦 Building mock server..."
cd testing
go build -o mock_server mock_server.go

if [ $? -eq 0 ]; then
    echo "✅ Mock server built successfully: testing/mock_server"
    
    # Test the build
    echo "🧪 Testing mock server build..."
    ./mock_server --help 2>/dev/null || echo "Mock server built and ready to run"
    
    echo ""
    echo "🚀 Usage:"
    echo "  cd testing"
    echo "  ./mock_server 8080"
    echo ""
    echo "📋 Available endpoints:"
    echo "  GET  /api/2/profile/"
    echo "  GET  /api/2/customers/"
    echo "  GET  /api/2/customers/{id}/"
    echo "  PATCH /api/2/customers/{id}/"
    echo "  GET  /api/2/appointments/"
    echo "  POST /api/2/appointments/"
    echo "  GET  /api/2/routes/"
    echo "  GET  /api/2/routes/{id}/"
    echo "  GET  /api/2/search/users/"
    echo "  GET  /api/2/datafields/"
    
else
    echo "❌ Failed to build mock server"
    exit 1
fi 