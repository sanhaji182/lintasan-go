// Package memory provides a pure-Go TF-IDF embedder and Redis-backed vector store
// using a minimal RESP2 client with zero external dependencies.
package memory

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

// Client is a minimal Redis client speaking RESP2 over raw TCP.
type Client struct {
	mu     sync.Mutex
	conn   net.Conn
	reader *bufio.Reader
	addr   string
}

// NewClient connects to a Redis server at addr.
func NewClient(addr string) (*Client, error) {
	c := &Client{addr: addr}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) connect() error {
	conn, err := net.DialTimeout("tcp", c.addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("resp: dial %s: %w", c.addr, err)
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

func (c *Client) reconnect() error {
	if c.conn != nil {
		c.conn.Close()
	}
	return c.connect()
}

// Do executes a Redis command and returns the parsed response.
func (c *Client) Do(cmd string, args ...interface{}) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	parts := make([][]byte, 1+len(args))
	parts[0] = []byte(cmd)
	for i, a := range args {
		switch v := a.(type) {
		case string:
			parts[i+1] = []byte(v)
		case []byte:
			parts[i+1] = v
		case int:
			parts[i+1] = []byte(strconv.Itoa(v))
		case int64:
			parts[i+1] = []byte(strconv.FormatInt(v, 10))
		case float64:
			parts[i+1] = []byte(strconv.FormatFloat(v, 'f', -1, 64))
		case nil:
			parts[i+1] = []byte{}
		default:
			parts[i+1] = []byte(fmt.Sprint(v))
		}
	}

	var buf []byte
	buf = append(buf, '*')
	buf = append(buf, strconv.Itoa(len(parts))...)
	buf = append(buf, '\r', '\n')
	for _, p := range parts {
		buf = append(buf, '$')
		buf = append(buf, strconv.Itoa(len(p))...)
		buf = append(buf, '\r', '\n')
		buf = append(buf, p...)
		buf = append(buf, '\r', '\n')
	}

	err := c.writeAll(buf)
	if err != nil {
		if rerr := c.reconnect(); rerr != nil {
			return nil, rerr
		}
		if err2 := c.writeAll(buf); err2 != nil {
			return nil, err2
		}
	}
	return c.readResponse()
}

func (c *Client) writeAll(data []byte) error {
	for len(data) > 0 {
		n, err := c.conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func (c *Client) readResponse() (interface{}, error) {
	b, err := c.reader.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("resp read: %w", err)
	}
	switch b {
	case '+':
		return c.readSimpleString()
	case '-':
		return nil, c.readError()
	case ':':
		return c.readInteger()
	case '$':
		return c.readBulkString()
	case '*':
		return c.readArray()
	default:
		return nil, fmt.Errorf("resp: unexpected type byte %q", b)
	}
}

func (c *Client) readSimpleString() (string, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return line[:len(line)-2], nil
}

func (c *Client) readError() error {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return err
	}
	return errors.New(line[:len(line)-2])
}

func (c *Client) readInteger() (int64, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(line[:len(line)-2], 10, 64)
}

func (c *Client) readBulkString() (interface{}, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	l, err := strconv.Atoi(line[:len(line)-2])
	if err != nil {
		return nil, fmt.Errorf("resp: bad bulk length: %w", err)
	}
	if l < 0 {
		return nil, nil
	}
	buf := make([]byte, l+2)
	if _, err := io.ReadFull(c.reader, buf); err != nil {
		return nil, err
	}
	return buf[:l], nil
}

func (c *Client) readArray() ([]interface{}, error) {
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	l, err := strconv.Atoi(line[:len(line)-2])
	if err != nil {
		return nil, fmt.Errorf("resp: bad array length: %w", err)
	}
	if l < 0 {
		return nil, nil
	}
	arr := make([]interface{}, l)
	for i := 0; i < l; i++ {
		v, err := c.readResponse()
		if err != nil {
			return nil, err
		}
		arr[i] = v
	}
	return arr, nil
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// respToString converts a response value to string.
func respToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprint(v)
	}
}
