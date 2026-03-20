// Package scriptrun implements remote script-driven installation.
// Scripts report progress via ##OCH: protocol lines on stdout.
package scriptrun

import (
	"encoding/json"
	"strconv"
	"strings"
)

// MessageType identifies the kind of ##OCH: protocol message.
type MessageType string

const (
	MsgProgress MessageType = "PROGRESS"
	MsgDetail   MessageType = "DETAIL"
	MsgError    MessageType = "ERROR"
	MsgReboot   MessageType = "REBOOT"
	MsgVerify   MessageType = "VERIFY"
	MsgDiag     MessageType = "DIAG"
	MsgHeal     MessageType = "HEAL"
)

// ProtocolMessage is a parsed ##OCH: line from script output.
type ProtocolMessage struct {
	Type     MessageType
	Progress int    // 0-100, only for PROGRESS
	Text     string // message text
	// For VERIFY: "PASS" or "FAIL:reason"
	VerifyOK     bool
	VerifyReason string
	// For DIAG: parsed JSON key-value
	DiagKey   string
	DiagValue string
	// For HEAL: "START", "STRATEGY", "REPAIR", "RETRY", "RESOLVED", "ESCALATE"
	HealType   string
	HealIssue  string
	HealDetail string
}

const protocolPrefix = "##OCH:"

// ParseLine attempts to parse a single line as an ##OCH: protocol message.
// Returns nil if the line is not a protocol message.
func ParseLine(line string) *ProtocolMessage {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, protocolPrefix) {
		return nil
	}
	rest := line[len(protocolPrefix):]

	// Split into type:payload
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) == 0 {
		return nil
	}

	msgType := MessageType(strings.TrimSpace(parts[0]))
	payload := ""
	if len(parts) > 1 {
		payload = parts[1]
	}

	switch msgType {
	case MsgProgress:
		return parseProgress(payload)
	case MsgDetail:
		return &ProtocolMessage{Type: MsgDetail, Text: payload}
	case MsgError:
		return &ProtocolMessage{Type: MsgError, Text: payload}
	case MsgReboot:
		return &ProtocolMessage{Type: MsgReboot, Text: payload}
	case MsgVerify:
		return parseVerify(payload)
	case MsgDiag:
		return parseDiag(payload)
	case MsgHeal:
		return parseHeal(payload)
	default:
		return nil
	}
}

// parseProgress parses "50:Installing..." format.
func parseProgress(payload string) *ProtocolMessage {
	parts := strings.SplitN(payload, ":", 2)
	if len(parts) == 0 {
		return &ProtocolMessage{Type: MsgProgress, Text: payload}
	}

	pct, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return &ProtocolMessage{Type: MsgProgress, Text: payload}
	}

	text := ""
	if len(parts) > 1 {
		text = parts[1]
	}

	return &ProtocolMessage{Type: MsgProgress, Progress: pct, Text: text}
}

// parseVerify parses "PASS" or "FAIL:reason".
func parseVerify(payload string) *ProtocolMessage {
	payload = strings.TrimSpace(payload)
	if strings.HasPrefix(payload, "PASS") {
		return &ProtocolMessage{Type: MsgVerify, VerifyOK: true, Text: "PASS"}
	}
	// FAIL:reason
	reason := strings.TrimPrefix(payload, "FAIL")
	reason = strings.TrimPrefix(reason, ":")
	return &ProtocolMessage{Type: MsgVerify, VerifyOK: false, VerifyReason: reason, Text: payload}
}

// parseDiag parses '{"key":"k","value":"v"}' format.
func parseDiag(payload string) *ProtocolMessage {
	payload = strings.TrimSpace(payload)
	var kv struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(payload), &kv); err != nil {
		return &ProtocolMessage{Type: MsgDiag, Text: payload}
	}
	return &ProtocolMessage{Type: MsgDiag, DiagKey: kv.Key, DiagValue: kv.Value, Text: payload}
}

// parseHeal parses "TYPE:issue:detail" format where TYPE is START, STRATEGY, REPAIR, RETRY, RESOLVED, ESCALATE.
func parseHeal(payload string) *ProtocolMessage {
	payload = strings.TrimSpace(payload)
	healParts := strings.SplitN(payload, ":", 2)
	if len(healParts) == 0 {
		return &ProtocolMessage{Type: MsgHeal, Text: payload}
	}

	msg := &ProtocolMessage{Type: MsgHeal, HealType: healParts[0]}
	if len(healParts) > 1 {
		msg.HealDetail = healParts[1]
	}

	// For START and RESOLVED types, the detail is the issue identifier
	if msg.HealType == "START" || msg.HealType == "RESOLVED" {
		msg.HealIssue = msg.HealDetail
	}

	msg.Text = payload
	return msg
}
