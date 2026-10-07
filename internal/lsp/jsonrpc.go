package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"strconv"
	"sync"
)

// conn speaks JSON-RPC 2.0 framed with Content-Length headers, as LSP does.
type conn struct {
	w  io.Writer
	wm sync.Mutex // one message at a time

	mu      sync.Mutex
	nextID  int
	pending map[int]chan response
	done    chan struct{} // closed when the server stops talking
	err     error         // why it stopped

	// onNotify handles notifications from the server; onRequest answers its
	// requests. Both run on the reading goroutine and must not block.
	onNotify  func(method string, params json.RawMessage)
	onRequest func(method string, params json.RawMessage) (any, error)
}

type response struct {
	result json.RawMessage
	err    error
}

// message is any incoming message: a response (ID), a notification
// (Method) or a request from the server (both).
type message struct {
	ID     *json.RawMessage `json:"id"`
	Method string           `json:"method"`
	Params json.RawMessage  `json:"params"`
	Result json.RawMessage  `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newConn(w io.Writer) *conn {
	return &conn{w: w, pending: map[int]chan response{}, done: make(chan struct{})}
}

// call sends a request and waits for its result (decoded into result when
// not nil), the server stopping, or ctx.
func (c *conn) call(ctx context.Context, method string, params, result any) error {
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan response, 1)
	c.pending[id] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
	}()

	if err := c.send(struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{"2.0", id, method, params}); err != nil {
		return err
	}
	select {
	case r := <-ch:
		if r.err != nil || result == nil {
			return r.err
		}
		return json.Unmarshal(r.result, result)
	case <-c.done:
		return c.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// notify sends a notification (no answer).
func (c *conn) notify(method string, params any) error {
	return c.send(struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params,omitempty"`
	}{"2.0", method, params})
}

func (c *conn) send(v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.wm.Lock()
	defer c.wm.Unlock()
	_, err = fmt.Fprintf(c.w, "Content-Length: %d\r\n\r\n%s", len(body), body)
	return err
}

// read handles everything the server sends until it stops.
func (c *conn) read(r io.Reader) {
	br := bufio.NewReader(r)
	for {
		body, err := readMessage(br)
		if err != nil {
			c.mu.Lock()
			c.err = err
			c.mu.Unlock()
			close(c.done)
			return
		}
		var m message
		if json.Unmarshal(body, &m) != nil {
			continue
		}
		switch {
		case m.Method != "" && m.ID != nil:
			c.reply(*m.ID, m.Method, m.Params)
		case m.Method != "":
			if c.onNotify != nil {
				c.onNotify(m.Method, m.Params)
			}
		case m.ID != nil:
			var id int
			if json.Unmarshal(*m.ID, &id) != nil {
				continue
			}
			res := response{result: m.Result}
			if m.Error != nil {
				res.err = fmt.Errorf("%s (código %d)", m.Error.Message, m.Error.Code)
			}
			c.mu.Lock()
			if ch, ok := c.pending[id]; ok {
				ch <- res
			}
			c.mu.Unlock()
		}
	}
}

// reply answers a request from the server.
func (c *conn) reply(id json.RawMessage, method string, params json.RawMessage) {
	result, err := any(nil), errors.New("método não suportado: "+method)
	if c.onRequest != nil {
		result, err = c.onRequest(method, params)
	}
	if err != nil {
		_ = c.send(struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Error   any             `json:"error"`
		}{"2.0", id, map[string]any{"code": -32601, "message": err.Error()}})
		return
	}
	_ = c.send(struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Result  any             `json:"result"`
	}{"2.0", id, result})
}

// readMessage reads one framed message body.
func readMessage(br *bufio.Reader) ([]byte, error) {
	header, err := textproto.NewReader(br).ReadMIMEHeader()
	if err != nil {
		return nil, err
	}
	n, err := strconv.Atoi(header.Get("Content-Length"))
	if err != nil || n < 0 {
		return nil, fmt.Errorf("cabeçalho inválido: %v", header)
	}
	body := make([]byte, n)
	_, err = io.ReadFull(br, body)
	return body, err
}
