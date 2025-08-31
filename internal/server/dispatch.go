package server

import (
	"fmt"
	"strconv"
	"strings"

	"pulsedb/internal/proto"
	"pulsedb/internal/store"
)

// CommandHandler represents a command handler function
type CommandHandler func(args []string) proto.RESPValue

// CommandDispatcher handles command dispatching and execution
type CommandDispatcher struct {
	store    *store.Store
	commands map[string]CommandHandler
}

// NewCommandDispatcher creates a new command dispatcher
func NewCommandDispatcher(store *store.Store, metrics interface{}) *CommandDispatcher {
	dispatcher := &CommandDispatcher{
		store:    store,
		commands: make(map[string]CommandHandler),
	}

	// Register core commands
	dispatcher.registerCommands()

	return dispatcher
}

// registerCommands registers all available commands
func (d *CommandDispatcher) registerCommands() {
	d.commands["PING"] = d.handlePing
	d.commands["SET"] = d.handleSet
	d.commands["GET"] = d.handleGet
	d.commands["DEL"] = d.handleDel
	d.commands["EXPIRE"] = d.handleExpire
	d.commands["TTL"] = d.handleTTL
	d.commands["GETAT"] = d.handleGetAt
	d.commands["HIST"] = d.handleHist
}

// Dispatch processes a RESP command and returns a response
func (d *CommandDispatcher) Dispatch(value proto.RESPValue) proto.RESPValue {
	cmd, args, err := value.ToCommand()
	if err != nil {
		return proto.RESPValue{
			Type:   proto.Error,
			String: fmt.Sprintf("ERR %s", err.Error()),
		}
	}

	handler, exists := d.commands[cmd]
	if !exists {
		return proto.RESPValue{
			Type:   proto.Error,
			String: fmt.Sprintf("ERR unknown command '%s'", cmd),
		}
	}

	return handler(args)
}

// Command handlers

func (d *CommandDispatcher) handlePing(args []string) proto.RESPValue {
	if len(args) == 0 {
		return proto.RESPValue{Type: proto.SimpleString, String: "PONG"}
	}
	if len(args) == 1 {
		return proto.RESPValue{Type: proto.BulkString, String: args[0]}
	}
	return proto.RESPValue{
		Type:   proto.Error,
		String: "ERR wrong number of arguments for 'ping' command",
	}
}

func (d *CommandDispatcher) handleSet(args []string) proto.RESPValue {
	if len(args) < 2 {
		return proto.RESPValue{
			Type:   proto.Error,
			String: "ERR wrong number of arguments for 'set' command",
		}
	}

	key := args[0]
	value := args[1]
	var ttlMs int64

	// Parse optional TTL arguments (PX milliseconds, EX seconds)
	for i := 2; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return proto.RESPValue{
				Type:   proto.Error,
				String: "ERR syntax error",
			}
		}

		option := strings.ToUpper(args[i])
		ttlStr := args[i+1]

		switch option {
		case "PX":
			ttl, err := strconv.ParseInt(ttlStr, 10, 64)
			if err != nil || ttl <= 0 {
				return proto.RESPValue{
					Type:   proto.Error,
					String: "ERR invalid expire time in 'set' command",
				}
			}
			ttlMs = ttl
		case "EX":
			ttl, err := strconv.ParseInt(ttlStr, 10, 64)
			if err != nil || ttl <= 0 {
				return proto.RESPValue{
					Type:   proto.Error,
					String: "ERR invalid expire time in 'set' command",
				}
			}
			ttlMs = ttl * 1000 // Convert seconds to milliseconds
		default:
			return proto.RESPValue{
				Type:   proto.Error,
				String: fmt.Sprintf("ERR syntax error near '%s'", option),
			}
		}
	}

	d.store.Set(key, value, ttlMs)
	return proto.RESPValue{Type: proto.SimpleString, String: "OK"}
}

func (d *CommandDispatcher) handleGet(args []string) proto.RESPValue {
	if len(args) != 1 {
		return proto.RESPValue{
			Type:   proto.Error,
			String: "ERR wrong number of arguments for 'get' command",
		}
	}

	key := args[0]
	value, exists := d.store.Get(key)
	if !exists {
		return proto.RESPValue{Type: proto.BulkString, Null: true}
	}

	return proto.RESPValue{Type: proto.BulkString, String: value}
}

func (d *CommandDispatcher) handleDel(args []string) proto.RESPValue {
	if len(args) == 0 {
		return proto.RESPValue{
			Type:   proto.Error,
			String: "ERR wrong number of arguments for 'del' command",
		}
	}

	deleted := int64(0)
	for _, key := range args {
		if d.store.Delete(key) {
			deleted++
		}
	}

	return proto.RESPValue{Type: proto.Integer, Int: deleted}
}

func (d *CommandDispatcher) handleExpire(args []string) proto.RESPValue {
	if len(args) != 2 {
		return proto.RESPValue{
			Type:   proto.Error,
			String: "ERR wrong number of arguments for 'expire' command",
		}
	}

	key := args[0]
	ttl, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return proto.RESPValue{
			Type:   proto.Error,
			String: "ERR value is not an integer or out of range",
		}
	}

	if d.store.Expire(key, ttl*1000) { // Convert seconds to milliseconds
		return proto.RESPValue{Type: proto.Integer, Int: 1}
	}

	return proto.RESPValue{Type: proto.Integer, Int: 0}
}

func (d *CommandDispatcher) handleTTL(args []string) proto.RESPValue {
	if len(args) != 1 {
		return proto.RESPValue{
			Type:   proto.Error,
			String: "ERR wrong number of arguments for 'ttl' command",
		}
	}

	key := args[0]
	ttlMs := d.store.TTL(key)
	ttlSeconds := ttlMs / 1000 // Convert milliseconds to seconds

	return proto.RESPValue{Type: proto.Integer, Int: ttlSeconds}
}

func (d *CommandDispatcher) handleGetAt(args []string) proto.RESPValue {
	if len(args) != 2 {
		return proto.RESPValue{
			Type:   proto.Error,
			String: "ERR wrong number of arguments for 'getat' command",
		}
	}

	key := args[0]
	timestamp, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return proto.RESPValue{
			Type:   proto.Error,
			String: "ERR value is not an integer or out of range",
		}
	}

	value, exists := d.store.GetAt(key, timestamp)
	if !exists {
		return proto.RESPValue{Type: proto.BulkString, Null: true}
	}

	return proto.RESPValue{Type: proto.BulkString, String: value}
}

func (d *CommandDispatcher) handleHist(args []string) proto.RESPValue {
	if len(args) < 1 || len(args) > 2 {
		return proto.RESPValue{
			Type:   proto.Error,
			String: "ERR wrong number of arguments for 'hist' command",
		}
	}

	key := args[0]
	limit := 0

	if len(args) == 2 {
		var err error
		limit, err = strconv.Atoi(args[1])
		if err != nil || limit < 0 {
			return proto.RESPValue{
				Type:   proto.Error,
				String: "ERR value is not a valid limit",
			}
		}
	}

	history := d.store.History(key, limit)

	// Build response array
	result := make([]proto.RESPValue, len(history)*2)
	for i, version := range history {
		result[i*2] = proto.RESPValue{
			Type: proto.Integer,
			Int:  version.Timestamp,
		}
		result[i*2+1] = proto.RESPValue{
			Type:   proto.BulkString,
			String: version.Data,
		}
	}

	return proto.RESPValue{Type: proto.Array, Array: result}
}
