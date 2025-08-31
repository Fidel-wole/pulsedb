package proto

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// RESPType represents the type of RESP data
type RESPType byte

const (
	SimpleString RESPType = '+'
	Error        RESPType = '-'
	Integer      RESPType = ':'
	BulkString   RESPType = '$'
	Array        RESPType = '*'
)

// RESPValue represents a RESP protocol value
type RESPValue struct {
	Type   RESPType
	String string
	Int    int64
	Array  []RESPValue
	Null   bool
}

// RESPReader reads RESP protocol messages
type RESPReader struct {
	reader *bufio.Reader
}

// NewRESPReader creates a new RESP reader
func NewRESPReader(r io.Reader) *RESPReader {
	return &RESPReader{
		reader: bufio.NewReader(r),
	}
}

// Read reads a RESP value from the reader
func (r *RESPReader) Read() (RESPValue, error) {
	typeByte, err := r.reader.ReadByte()
	if err != nil {
		return RESPValue{}, err
	}

	switch RESPType(typeByte) {
	case SimpleString:
		return r.readSimpleString()
	case Error:
		return r.readError()
	case Integer:
		return r.readInteger()
	case BulkString:
		return r.readBulkString()
	case Array:
		return r.readArray()
	default:
		return RESPValue{}, fmt.Errorf("unknown RESP type: %c", typeByte)
	}
}

func (r *RESPReader) readSimpleString() (RESPValue, error) {
	line, err := r.readLine()
	if err != nil {
		return RESPValue{}, err
	}
	return RESPValue{Type: SimpleString, String: line}, nil
}

func (r *RESPReader) readError() (RESPValue, error) {
	line, err := r.readLine()
	if err != nil {
		return RESPValue{}, err
	}
	return RESPValue{Type: Error, String: line}, nil
}

func (r *RESPReader) readInteger() (RESPValue, error) {
	line, err := r.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	val, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		return RESPValue{}, fmt.Errorf("invalid integer: %s", line)
	}

	return RESPValue{Type: Integer, Int: val}, nil
}

func (r *RESPReader) readBulkString() (RESPValue, error) {
	line, err := r.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	length, err := strconv.Atoi(line)
	if err != nil {
		return RESPValue{}, fmt.Errorf("invalid bulk string length: %s", line)
	}

	if length == -1 {
		return RESPValue{Type: BulkString, Null: true}, nil
	}

	if length < 0 {
		return RESPValue{}, fmt.Errorf("invalid bulk string length: %d", length)
	}

	data := make([]byte, length+2) // +2 for \r\n
	_, err = io.ReadFull(r.reader, data)
	if err != nil {
		return RESPValue{}, err
	}

	return RESPValue{Type: BulkString, String: string(data[:length])}, nil
}

func (r *RESPReader) readArray() (RESPValue, error) {
	line, err := r.readLine()
	if err != nil {
		return RESPValue{}, err
	}

	length, err := strconv.Atoi(line)
	if err != nil {
		return RESPValue{}, fmt.Errorf("invalid array length: %s", line)
	}

	if length == -1 {
		return RESPValue{Type: Array, Null: true}, nil
	}

	if length < 0 {
		return RESPValue{}, fmt.Errorf("invalid array length: %d", length)
	}

	array := make([]RESPValue, length)
	for i := 0; i < length; i++ {
		value, err := r.Read()
		if err != nil {
			return RESPValue{}, err
		}
		array[i] = value
	}

	return RESPValue{Type: Array, Array: array}, nil
}

func (r *RESPReader) readLine() (string, error) {
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	// Remove \r\n
	if len(line) >= 2 && line[len(line)-2:] == "\r\n" {
		line = line[:len(line)-2]
	} else if len(line) >= 1 && line[len(line)-1:] == "\n" {
		line = line[:len(line)-1]
	}

	return line, nil
}

// RESPWriter writes RESP protocol messages
type RESPWriter struct {
	writer io.Writer
}

// NewRESPWriter creates a new RESP writer
func NewRESPWriter(w io.Writer) *RESPWriter {
	return &RESPWriter{writer: w}
}

// WriteValue writes a RESP value
func (w *RESPWriter) WriteValue(value RESPValue) error {
	switch value.Type {
	case SimpleString:
		return w.WriteSimpleString(value.String)
	case Error:
		return w.WriteError(value.String)
	case Integer:
		return w.WriteInteger(value.Int)
	case BulkString:
		if value.Null {
			return w.WriteNullBulkString()
		}
		return w.WriteBulkString(value.String)
	case Array:
		if value.Null {
			return w.WriteNullArray()
		}
		return w.WriteArray(value.Array)
	default:
		return fmt.Errorf("unknown RESP type: %c", value.Type)
	}
}

// WriteSimpleString writes a simple string
func (w *RESPWriter) WriteSimpleString(s string) error {
	_, err := fmt.Fprintf(w.writer, "+%s\r\n", s)
	return err
}

// WriteError writes an error
func (w *RESPWriter) WriteError(s string) error {
	_, err := fmt.Fprintf(w.writer, "-%s\r\n", s)
	return err
}

// WriteInteger writes an integer
func (w *RESPWriter) WriteInteger(i int64) error {
	_, err := fmt.Fprintf(w.writer, ":%d\r\n", i)
	return err
}

// WriteBulkString writes a bulk string
func (w *RESPWriter) WriteBulkString(s string) error {
	_, err := fmt.Fprintf(w.writer, "$%d\r\n%s\r\n", len(s), s)
	return err
}

// WriteNullBulkString writes a null bulk string
func (w *RESPWriter) WriteNullBulkString() error {
	_, err := fmt.Fprintf(w.writer, "$-1\r\n")
	return err
}

// WriteArray writes an array
func (w *RESPWriter) WriteArray(arr []RESPValue) error {
	if _, err := fmt.Fprintf(w.writer, "*%d\r\n", len(arr)); err != nil {
		return err
	}

	for _, value := range arr {
		if err := w.WriteValue(value); err != nil {
			return err
		}
	}

	return nil
}

// WriteNullArray writes a null array
func (w *RESPWriter) WriteNullArray() error {
	_, err := fmt.Fprintf(w.writer, "*-1\r\n")
	return err
}

// ToStringArray converts a RESP array to a string slice
func (v RESPValue) ToStringArray() ([]string, error) {
	if v.Type != Array {
		return nil, fmt.Errorf("value is not an array")
	}

	if v.Null {
		return nil, nil
	}

	result := make([]string, len(v.Array))
	for i, val := range v.Array {
		if val.Type == BulkString && !val.Null {
			result[i] = val.String
		} else if val.Type == SimpleString {
			result[i] = val.String
		} else {
			return nil, fmt.Errorf("array element %d is not a string", i)
		}
	}

	return result, nil
}

// ToCommand converts a RESP array to a command (first element is command name, rest are args)
func (v RESPValue) ToCommand() (string, []string, error) {
	args, err := v.ToStringArray()
	if err != nil {
		return "", nil, err
	}

	if len(args) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}

	cmd := strings.ToUpper(args[0])
	return cmd, args[1:], nil
}
