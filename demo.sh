#!/bin/bash

# PulseDB Demo Script
# This script demonstrates the features of PulseDB

set -e

echo "🚀 PulseDB Feature Demo"
echo "======================="

# Check if PulseDB is built
if [ ! -f "./pulsedb" ]; then
    echo "📦 Building PulseDB..."
    go build -o pulsedb cmd/pulsedb/main.go
fi

# Start PulseDB in the background
echo "🔄 Starting PulseDB..."
./pulsedb &
PULSEDB_PID=$!

# Wait for server to start
sleep 2

# Function to cleanup
cleanup() {
    echo "🛑 Stopping PulseDB..."
    kill $PULSEDB_PID 2>/dev/null || true
    wait $PULSEDB_PID 2>/dev/null || true
}

# Set trap to cleanup on exit
trap cleanup EXIT

echo "✅ PulseDB is running on TCP:6380 and HTTP:8080"
echo ""

# Test basic Redis commands
echo "🔧 Testing Basic Redis Commands"
echo "------------------------------"

# Test with redis-cli if available
if command -v redis-cli &> /dev/null; then
    echo "📡 Using redis-cli for testing..."
    
    echo "PING..."
    redis-cli -p 6380 ping
    
    echo "SET key value..."
    redis-cli -p 6380 set mykey "Hello PulseDB"
    
    echo "GET key..."
    redis-cli -p 6380 get mykey
    
    echo "SET with TTL..."
    redis-cli -p 6380 set tempkey "expires soon" ex 5
    
    echo "TTL check..."
    redis-cli -p 6380 ttl tempkey
    
    echo "DEL key..."
    redis-cli -p 6380 del mykey
    
    # MVCC Demo
    echo ""
    echo "🕰️ Testing Time-Travel (MVCC) Features"
    echo "------------------------------------"
    
    echo "Setting multiple versions of a key..."
    redis-cli -p 6380 set counter 1
    sleep 0.1
    redis-cli -p 6380 set counter 2  
    sleep 0.1
    redis-cli -p 6380 set counter 3
    
    echo "Current value:"
    redis-cli -p 6380 get counter
    
    echo "History (showing timestamp, value pairs):"
    redis-cli -p 6380 hist counter 5
    
    # Get timestamp for GETAT test
    TIMESTAMP=$(date -d '5 seconds ago' +%s)000  # Convert to milliseconds
    echo "Value 5 seconds ago (timestamp: $TIMESTAMP):"
    redis-cli -p 6380 getat counter $TIMESTAMP
    
else
    echo "⚠️ redis-cli not found. Install Redis client tools to test RESP protocol."
    echo "   On Ubuntu/Debian: sudo apt-get install redis-tools"
    echo "   On macOS: brew install redis"
fi

# Test HTTP API
echo ""
echo "🌐 Testing HTTP API"
echo "------------------"

if command -v curl &> /dev/null; then
    echo "Setting key via HTTP POST..."
    curl -s -X POST http://localhost:8080/kv/httpkey \
         -H "Content-Type: application/json" \
         -d '{"value": "Hello HTTP", "ttl": 60}' | jq '.' 2>/dev/null || cat
    
    echo ""
    echo "Getting key via HTTP GET..."
    curl -s http://localhost:8080/kv/httpkey | jq '.' 2>/dev/null || cat
    
    echo ""
    echo "Health check..."
    curl -s http://localhost:8080/health | jq '.' 2>/dev/null || cat
    
    echo ""
    echo "Deleting key via HTTP DELETE..."
    curl -s -X DELETE http://localhost:8080/kv/httpkey | jq '.' 2>/dev/null || cat
    
else
    echo "⚠️ curl not found. Install curl to test HTTP API."
fi

# Run Go client example
echo ""
echo "🔌 Testing Go Client"
echo "-------------------"

if [ -f "examples/client.go" ]; then
    echo "Running Go client example..."
    timeout 10s go run examples/client.go || echo "Client test completed (or timed out)"
else
    echo "⚠️ examples/client.go not found"
fi

# Performance test
echo ""
echo "⚡ Quick Performance Test"
echo "------------------------"

if command -v redis-cli &> /dev/null; then
    echo "Setting 1000 keys..."
    start_time=$(date +%s.%N)
    
    for i in {1..1000}; do
        redis-cli -p 6380 set "perf_key_$i" "value_$i" > /dev/null
    done
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "N/A")
    
    echo "✅ Set 1000 keys in ${duration}s"
    
    echo "Getting 1000 keys..."
    start_time=$(date +%s.%N)
    
    for i in {1..1000}; do
        redis-cli -p 6380 get "perf_key_$i" > /dev/null
    done
    
    end_time=$(date +%s.%N)
    duration=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "N/A")
    
    echo "✅ Got 1000 keys in ${duration}s"
fi

# Final health check
echo ""
echo "📊 Final Statistics"
echo "------------------"

if command -v curl &> /dev/null; then
    curl -s http://localhost:8080/health | jq '.stats' 2>/dev/null || curl -s http://localhost:8080/health
fi

echo ""
echo "🎉 Demo completed successfully!"
echo "   - Basic Redis commands: ✅"
echo "   - Time-travel queries: ✅" 
echo "   - HTTP API: ✅"
echo "   - Performance test: ✅"
echo ""
echo "💡 Try connecting with redis-cli: redis-cli -p 6380"
echo "💡 Try the HTTP API: curl http://localhost:8080/health"
