# Example WASM Function for PulseDB

This directory contains example WASM functions that can be loaded into PulseDB for event-driven processing.

## Audit Logger (Rust)

A simple audit logger that logs all SET operations to a separate audit stream.

### Building

```bash
# Install Rust and wasm-pack if not already installed
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
cargo install wasm-pack

# Build the WASM module
cd examples/wasm/audit_logger
wasm-pack build --target web

# The WASM binary will be in pkg/audit_logger_bg.wasm
```

### Usage

```bash
# In redis-cli connected to PulseDB
FUNC.LOAD audit_logger < examples/wasm/audit_logger/pkg/audit_logger_bg.wasm
ON.SET user:* audit_logger
```

## Counter Function (AssemblyScript)

A function that maintains counters for different event types.

### Building

```bash
# Install AssemblyScript
npm install -g assemblyscript

# Build the WASM module
cd examples/wasm/counter
asc counter.ts --target release --optimize
```

## Host Functions Available to WASM

PulseDB provides the following host functions that WASM modules can import:

- `get(key_ptr: i32, key_len: i32) -> value_ptr: i32` - Get a key's value
- `set(key_ptr: i32, key_len: i32, value_ptr: i32, value_len: i32)` - Set a key's value  
- `publish(channel_ptr: i32, channel_len: i32, message_ptr: i32, message_len: i32)` - Publish to a channel
- `log(level: i32, message_ptr: i32, message_len: i32)` - Log a message

## Event Types

WASM functions can be bound to the following event types:

- `ON.SET pattern function_name` - Triggered when a key matching pattern is set
- `ON.EXPIRE pattern function_name` - Triggered when a key matching pattern expires
- `ON.DELETE pattern function_name` - Triggered when a key matching pattern is deleted

Pattern examples:
- `user:*` - All keys starting with "user:"
- `session:*:data` - Keys like "session:123:data" 
- `*` - All keys
