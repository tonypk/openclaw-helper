// Package types defines shared types for the OpenClaw Helper.
package types

// CheckStatus represents the status of a system check.
type CheckStatus string

const (
	StatusPass     CheckStatus = "pass"
	StatusFail     CheckStatus = "fail"
	StatusWarn     CheckStatus = "warn"
	StatusChecking CheckStatus = "checking"
	StatusSkipped  CheckStatus = "skipped"
)

// CheckResult holds the result of a single system check.
type CheckResult struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Message string      `json:"message"`
	Detail  string      `json:"detail,omitempty"`
}

// SystemReport aggregates all system check results.
type SystemReport struct {
	OS             CheckResult `json:"os"`
	Memory         CheckResult `json:"memory"`
	Disk           CheckResult `json:"disk"`
	Virtualization CheckResult `json:"virtualization"`
	WSL            CheckResult `json:"wsl"`
	Node           CheckResult `json:"node"`
	OpenClaw       CheckResult `json:"openclaw"`
	OverallReady   bool        `json:"overall_ready"`
}

// HelperInfo holds version and runtime info for the helper process.
type HelperInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// RPCRequest is a JSON-RPC 2.0 request.
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// RPCResponse is a JSON-RPC 2.0 response.
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	ErrCodeParse          = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)
