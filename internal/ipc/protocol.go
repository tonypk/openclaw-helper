// Package ipc provides JSON-RPC 2.0 over Named Pipe (Windows) or Unix socket (dev).
package ipc

import (
	"encoding/json"
	"fmt"

	"github.com/tonypk/openclaw-helper/internal/types"
)

const jsonRPCVersion = "2.0"

// Handler processes a JSON-RPC method call and returns a result or error.
type Handler func(params json.RawMessage) (interface{}, *types.RPCError)

// Router maps method names to handlers.
type Router struct {
	handlers map[string]Handler
}

// NewRouter creates an empty router.
func NewRouter() *Router {
	return &Router{handlers: make(map[string]Handler)}
}

// Register adds a handler for a method.
func (r *Router) Register(method string, handler Handler) {
	r.handlers[method] = handler
}

// Dispatch processes a raw JSON-RPC request and returns a response.
func (r *Router) Dispatch(data []byte) []byte {
	var req types.RPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return mustMarshal(types.RPCResponse{
			JSONRPC: jsonRPCVersion,
			ID:      nil,
			Error: &types.RPCError{
				Code:    types.ErrCodeParse,
				Message: "Parse error: " + err.Error(),
			},
		})
	}

	if req.JSONRPC != jsonRPCVersion || req.Method == "" {
		return mustMarshal(types.RPCResponse{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error: &types.RPCError{
				Code:    types.ErrCodeInvalidRequest,
				Message: "Invalid request",
			},
		})
	}

	handler, ok := r.handlers[req.Method]
	if !ok {
		return mustMarshal(types.RPCResponse{
			JSONRPC: jsonRPCVersion,
			ID:      req.ID,
			Error: &types.RPCError{
				Code:    types.ErrCodeMethodNotFound,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		})
	}

	// Marshal params to raw JSON for the handler
	var rawParams json.RawMessage
	if req.Params != nil {
		rawParams, _ = json.Marshal(req.Params)
	}

	result, rpcErr := handler(rawParams)
	resp := types.RPCResponse{
		JSONRPC: jsonRPCVersion,
		ID:      req.ID,
	}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}

	return mustMarshal(resp)
}

func mustMarshal(v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		// This should never happen with our types, but be safe
		return []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"internal marshal error"}}`)
	}
	return data
}
