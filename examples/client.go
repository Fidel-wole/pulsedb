package main

import (
	"fmt"
	"net"
	"time"

	"pulsedb/internal/proto"
)

func main() {
	// Connect to PulseDB
	conn, err := net.Dial("tcp", "localhost:6380")
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer conn.Close()

	reader := proto.NewRESPReader(conn)
	writer := proto.NewRESPWriter(conn)

	// Test PING command
	fmt.Println("Testing PING command...")
	pingCmd := proto.RESPValue{
		Type: proto.Array,
		Array: []proto.RESPValue{
			{Type: proto.BulkString, String: "PING"},
		},
	}

	if err := writer.WriteValue(pingCmd); err != nil {
		fmt.Printf("Failed to write PING: %v\n", err)
		return
	}

	response, err := reader.Read()
	if err != nil {
		fmt.Printf("Failed to read PING response: %v\n", err)
		return
	}
	fmt.Printf("PING response: %+v\n", response)

	// Test SET command
	fmt.Println("\nTesting SET command...")
	setCmd := proto.RESPValue{
		Type: proto.Array,
		Array: []proto.RESPValue{
			{Type: proto.BulkString, String: "SET"},
			{Type: proto.BulkString, String: "testkey"},
			{Type: proto.BulkString, String: "testvalue"},
		},
	}

	if err := writer.WriteValue(setCmd); err != nil {
		fmt.Printf("Failed to write SET: %v\n", err)
		return
	}

	response, err = reader.Read()
	if err != nil {
		fmt.Printf("Failed to read SET response: %v\n", err)
		return
	}
	fmt.Printf("SET response: %+v\n", response)

	// Test GET command
	fmt.Println("\nTesting GET command...")
	getCmd := proto.RESPValue{
		Type: proto.Array,
		Array: []proto.RESPValue{
			{Type: proto.BulkString, String: "GET"},
			{Type: proto.BulkString, String: "testkey"},
		},
	}

	if err := writer.WriteValue(getCmd); err != nil {
		fmt.Printf("Failed to write GET: %v\n", err)
		return
	}

	response, err = reader.Read()
	if err != nil {
		fmt.Printf("Failed to read GET response: %v\n", err)
		return
	}
	fmt.Printf("GET response: %+v\n", response)

	// Test MVCC commands
	fmt.Println("\nTesting MVCC commands...")

	// Set multiple versions
	now := time.Now().UnixMilli()

	setCmd2 := proto.RESPValue{
		Type: proto.Array,
		Array: []proto.RESPValue{
			{Type: proto.BulkString, String: "SET"},
			{Type: proto.BulkString, String: "mvcc_test"},
			{Type: proto.BulkString, String: "version1"},
		},
	}
	writer.WriteValue(setCmd2)
	reader.Read()

	time.Sleep(10 * time.Millisecond)

	setCmd3 := proto.RESPValue{
		Type: proto.Array,
		Array: []proto.RESPValue{
			{Type: proto.BulkString, String: "SET"},
			{Type: proto.BulkString, String: "mvcc_test"},
			{Type: proto.BulkString, String: "version2"},
		},
	}
	writer.WriteValue(setCmd3)
	reader.Read()

	// Test GETAT command
	fmt.Println("\nTesting GETAT command...")
	getatCmd := proto.RESPValue{
		Type: proto.Array,
		Array: []proto.RESPValue{
			{Type: proto.BulkString, String: "GETAT"},
			{Type: proto.BulkString, String: "mvcc_test"},
			{Type: proto.BulkString, String: fmt.Sprintf("%d", now+5)},
		},
	}

	if err := writer.WriteValue(getatCmd); err != nil {
		fmt.Printf("Failed to write GETAT: %v\n", err)
		return
	}

	response, err = reader.Read()
	if err != nil {
		fmt.Printf("Failed to read GETAT response: %v\n", err)
		return
	}
	fmt.Printf("GETAT response: %+v\n", response)

	// Test HIST command
	fmt.Println("\nTesting HIST command...")
	histCmd := proto.RESPValue{
		Type: proto.Array,
		Array: []proto.RESPValue{
			{Type: proto.BulkString, String: "HIST"},
			{Type: proto.BulkString, String: "mvcc_test"},
		},
	}

	if err := writer.WriteValue(histCmd); err != nil {
		fmt.Printf("Failed to write HIST: %v\n", err)
		return
	}

	response, err = reader.Read()
	if err != nil {
		fmt.Printf("Failed to read HIST response: %v\n", err)
		return
	}
	fmt.Printf("HIST response: %+v\n", response)

	fmt.Println("\nAll tests completed!")
}
