package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// Server listens on a named pipe (Windows) or unix socket (dev) for JSON-RPC calls.
type Server struct {
	router   *Router
	listener net.Listener
	wg       sync.WaitGroup
	quit     chan struct{}
}

// NewServer creates an IPC server with the given router.
func NewServer(router *Router) *Server {
	return &Server{
		router: router,
		quit:   make(chan struct{}),
	}
}

// Listen starts listening on the given address. The network type depends on the platform.
func (s *Server) Listen(address string) error {
	ln, err := listen(address)
	if err != nil {
		return fmt.Errorf("ipc listen: %w", err)
	}
	s.listener = ln
	return nil
}

// Serve accepts connections until Stop is called.
func (s *Server) Serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Printf("[ipc] accept error: %v", err)
				continue
			}
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.handleConn(conn)
		}()
	}
}

// Stop closes the listener and waits for active connections to finish.
func (s *Server) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
}

// Address returns the listening address. Useful for tests using ":0".
func (s *Server) Address() string {
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // up to 1MB per message

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		resp := s.router.Dispatch(line)
		if _, err := conn.Write(append(resp, '\n')); err != nil {
			if err != io.EOF {
				log.Printf("[ipc] write error: %v", err)
			}
			return
		}
	}
}

// Call is a helper to send a JSON-RPC request and read the response from a connection.
// Useful for testing.
func Call(conn net.Conn, method string, params interface{}) (json.RawMessage, error) {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if _, err := conn.Write(append(data, '\n')); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	return resp.Result, nil
}
