package ipc

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tonypk/openclaw-helper/internal/types"
)

func TestRouterDispatch_Ping(t *testing.T) {
	router := NewRouter()
	router.Register("ping", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		return "pong", nil
	})

	req := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	resp := router.Dispatch([]byte(req))

	var r types.RPCResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Error != nil {
		t.Fatalf("unexpected error: %v", r.Error)
	}

	result, _ := json.Marshal(r.Result)
	if string(result) != `"pong"` {
		t.Errorf("expected pong, got %s", result)
	}
}

func TestRouterDispatch_MethodNotFound(t *testing.T) {
	router := NewRouter()
	req := `{"jsonrpc":"2.0","id":1,"method":"nonexistent"}`
	resp := router.Dispatch([]byte(req))

	var r types.RPCResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if r.Error.Code != types.ErrCodeMethodNotFound {
		t.Errorf("expected code %d, got %d", types.ErrCodeMethodNotFound, r.Error.Code)
	}
}

func TestRouterDispatch_ParseError(t *testing.T) {
	router := NewRouter()
	resp := router.Dispatch([]byte("not json"))

	var r types.RPCResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if r.Error == nil {
		t.Fatal("expected parse error")
	}
	if r.Error.Code != types.ErrCodeParse {
		t.Errorf("expected code %d, got %d", types.ErrCodeParse, r.Error.Code)
	}
}

func TestRouterDispatch_InvalidRequest(t *testing.T) {
	router := NewRouter()
	// Missing method
	req := `{"jsonrpc":"2.0","id":1}`
	resp := router.Dispatch([]byte(req))

	var r types.RPCResponse
	json.Unmarshal(resp, &r)
	if r.Error == nil || r.Error.Code != types.ErrCodeInvalidRequest {
		t.Errorf("expected invalid request error, got %+v", r.Error)
	}
}

func TestRouterDispatch_WithParams(t *testing.T) {
	router := NewRouter()
	router.Register("echo", func(params json.RawMessage) (interface{}, *types.RPCError) {
		var p struct {
			Msg string `json:"msg"`
		}
		json.Unmarshal(params, &p)
		return p.Msg, nil
	})

	req := `{"jsonrpc":"2.0","id":1,"method":"echo","params":{"msg":"hello"}}`
	resp := router.Dispatch([]byte(req))

	var r types.RPCResponse
	json.Unmarshal(resp, &r)
	if r.Error != nil {
		t.Fatalf("unexpected error: %v", r.Error)
	}
	result, _ := json.Marshal(r.Result)
	if string(result) != `"hello"` {
		t.Errorf("expected hello, got %s", result)
	}
}

func TestServerIntegration(t *testing.T) {
	router := NewRouter()
	router.Register("helper.ping", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		return "pong", nil
	})
	router.Register("helper.version", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		return types.HelperInfo{Version: "test", GoVersion: "go1.22", OS: "test", Arch: "test"}, nil
	})

	sockPath := filepath.Join(os.TempDir(), "openclaw-test.sock")
	os.Remove(sockPath)

	srv := NewServer(router)
	if err := srv.Listen(sockPath); err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer srv.Stop()

	go srv.Serve()

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// Test ping
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	result, err := Call(conn, "helper.ping", nil)
	if err != nil {
		t.Fatalf("call ping: %v", err)
	}
	var pong string
	json.Unmarshal(result, &pong)
	if pong != "pong" {
		t.Errorf("expected pong, got %q", pong)
	}

	// Test version
	result, err = Call(conn, "helper.version", nil)
	if err != nil {
		t.Fatalf("call version: %v", err)
	}
	var info types.HelperInfo
	json.Unmarshal(result, &info)
	if info.Version != "test" {
		t.Errorf("expected version=test, got %q", info.Version)
	}

	// Test unknown method
	_, err = Call(conn, "nonexistent", nil)
	if err == nil {
		t.Error("expected error for unknown method")
	}
}
