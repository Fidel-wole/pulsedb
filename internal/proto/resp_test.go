package proto

import (
	"bytes"
	"strings"
	"testing"
)

func TestRESPReader(t *testing.T) {
	tests := []struct {
		input    string
		expected RESPValue
	}{
		{
			input:    "+OK\r\n",
			expected: RESPValue{Type: SimpleString, String: "OK"},
		},
		{
			input:    "-Error message\r\n",
			expected: RESPValue{Type: Error, String: "Error message"},
		},
		{
			input:    ":1000\r\n",
			expected: RESPValue{Type: Integer, Int: 1000},
		},
		{
			input:    "$6\r\nfoobar\r\n",
			expected: RESPValue{Type: BulkString, String: "foobar"},
		},
		{
			input:    "$-1\r\n",
			expected: RESPValue{Type: BulkString, Null: true},
		},
		{
			input: "*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
			expected: RESPValue{
				Type: Array,
				Array: []RESPValue{
					{Type: BulkString, String: "foo"},
					{Type: BulkString, String: "bar"},
				},
			},
		},
	}

	for _, test := range tests {
		reader := NewRESPReader(strings.NewReader(test.input))
		value, err := reader.Read()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			continue
		}

		if !compareRESPValues(value, test.expected) {
			t.Errorf("Expected %+v, got %+v", test.expected, value)
		}
	}
}

func TestRESPWriter(t *testing.T) {
	tests := []struct {
		value    RESPValue
		expected string
	}{
		{
			value:    RESPValue{Type: SimpleString, String: "OK"},
			expected: "+OK\r\n",
		},
		{
			value:    RESPValue{Type: Error, String: "Error message"},
			expected: "-Error message\r\n",
		},
		{
			value:    RESPValue{Type: Integer, Int: 1000},
			expected: ":1000\r\n",
		},
		{
			value:    RESPValue{Type: BulkString, String: "foobar"},
			expected: "$6\r\nfoobar\r\n",
		},
		{
			value:    RESPValue{Type: BulkString, Null: true},
			expected: "$-1\r\n",
		},
	}

	for _, test := range tests {
		var buf bytes.Buffer
		writer := NewRESPWriter(&buf)
		err := writer.WriteValue(test.value)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			continue
		}

		if buf.String() != test.expected {
			t.Errorf("Expected %q, got %q", test.expected, buf.String())
		}
	}
}

func TestToCommand(t *testing.T) {
	tests := []struct {
		value    RESPValue
		cmd      string
		args     []string
		hasError bool
	}{
		{
			value: RESPValue{
				Type: Array,
				Array: []RESPValue{
					{Type: BulkString, String: "SET"},
					{Type: BulkString, String: "key"},
					{Type: BulkString, String: "value"},
				},
			},
			cmd:  "SET",
			args: []string{"key", "value"},
		},
		{
			value: RESPValue{
				Type: Array,
				Array: []RESPValue{
					{Type: BulkString, String: "get"},
					{Type: BulkString, String: "key"},
				},
			},
			cmd:  "GET",
			args: []string{"key"},
		},
		{
			value: RESPValue{
				Type:  Array,
				Array: []RESPValue{},
			},
			hasError: true,
		},
	}

	for _, test := range tests {
		cmd, args, err := test.value.ToCommand()
		if test.hasError {
			if err == nil {
				t.Error("Expected error but got none")
			}
			continue
		}

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			continue
		}

		if cmd != test.cmd {
			t.Errorf("Expected command %s, got %s", test.cmd, cmd)
		}

		if len(args) != len(test.args) {
			t.Errorf("Expected %d args, got %d", len(test.args), len(args))
			continue
		}

		for i, arg := range args {
			if arg != test.args[i] {
				t.Errorf("Expected arg %d to be %s, got %s", i, test.args[i], arg)
			}
		}
	}
}

func compareRESPValues(a, b RESPValue) bool {
	if a.Type != b.Type || a.Null != b.Null {
		return false
	}

	switch a.Type {
	case SimpleString, Error, BulkString:
		return a.String == b.String
	case Integer:
		return a.Int == b.Int
	case Array:
		if len(a.Array) != len(b.Array) {
			return false
		}
		for i := range a.Array {
			if !compareRESPValues(a.Array[i], b.Array[i]) {
				return false
			}
		}
		return true
	}

	return true
}
