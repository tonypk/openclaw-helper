package scriptrun

import (
	"testing"
)

func TestParseLineProgress(t *testing.T) {
	tests := []struct {
		line     string
		wantPct  int
		wantText string
	}{
		{"##OCH:PROGRESS:50:Installing...", 50, "Installing..."},
		{"##OCH:PROGRESS:0:Starting", 0, "Starting"},
		{"##OCH:PROGRESS:100:Done!", 100, "Done!"},
		{"  ##OCH:PROGRESS:25:Quarter done  ", 25, "Quarter done"},
	}

	for _, tt := range tests {
		msg := ParseLine(tt.line)
		if msg == nil {
			t.Errorf("ParseLine(%q) = nil, want PROGRESS msg", tt.line)
			continue
		}
		if msg.Type != MsgProgress {
			t.Errorf("ParseLine(%q).Type = %s, want PROGRESS", tt.line, msg.Type)
		}
		if msg.Progress != tt.wantPct {
			t.Errorf("ParseLine(%q).Progress = %d, want %d", tt.line, msg.Progress, tt.wantPct)
		}
		if msg.Text != tt.wantText {
			t.Errorf("ParseLine(%q).Text = %q, want %q", tt.line, msg.Text, tt.wantText)
		}
	}
}

func TestParseLineDetail(t *testing.T) {
	msg := ParseLine("##OCH:DETAIL:apt-get is downloading packages")
	if msg == nil || msg.Type != MsgDetail {
		t.Fatal("expected DETAIL message")
	}
	if msg.Text != "apt-get is downloading packages" {
		t.Errorf("Text = %q", msg.Text)
	}
}

func TestParseLineError(t *testing.T) {
	msg := ParseLine("##OCH:ERROR:connection refused")
	if msg == nil || msg.Type != MsgError {
		t.Fatal("expected ERROR message")
	}
	if msg.Text != "connection refused" {
		t.Errorf("Text = %q", msg.Text)
	}
}

func TestParseLineReboot(t *testing.T) {
	msg := ParseLine("##OCH:REBOOT:WSL needs restart")
	if msg == nil || msg.Type != MsgReboot {
		t.Fatal("expected REBOOT message")
	}
	if msg.Text != "WSL needs restart" {
		t.Errorf("Text = %q", msg.Text)
	}
}

func TestParseLineVerifyPass(t *testing.T) {
	msg := ParseLine("##OCH:VERIFY:PASS")
	if msg == nil || msg.Type != MsgVerify {
		t.Fatal("expected VERIFY message")
	}
	if !msg.VerifyOK {
		t.Error("VerifyOK should be true")
	}
}

func TestParseLineVerifyFail(t *testing.T) {
	msg := ParseLine("##OCH:VERIFY:FAIL:node not found")
	if msg == nil || msg.Type != MsgVerify {
		t.Fatal("expected VERIFY message")
	}
	if msg.VerifyOK {
		t.Error("VerifyOK should be false")
	}
	if msg.VerifyReason != "node not found" {
		t.Errorf("VerifyReason = %q, want %q", msg.VerifyReason, "node not found")
	}
}

func TestParseLineDiag(t *testing.T) {
	msg := ParseLine(`##OCH:DIAG:{"key":"node_version","value":"v22.1.0"}`)
	if msg == nil || msg.Type != MsgDiag {
		t.Fatal("expected DIAG message")
	}
	if msg.DiagKey != "node_version" {
		t.Errorf("DiagKey = %q", msg.DiagKey)
	}
	if msg.DiagValue != "v22.1.0" {
		t.Errorf("DiagValue = %q", msg.DiagValue)
	}
}

func TestParseLineDiagInvalidJSON(t *testing.T) {
	msg := ParseLine("##OCH:DIAG:not json")
	if msg == nil || msg.Type != MsgDiag {
		t.Fatal("expected DIAG message")
	}
	if msg.DiagKey != "" {
		t.Errorf("DiagKey should be empty for invalid JSON, got %q", msg.DiagKey)
	}
}

func TestParseLineNonProtocol(t *testing.T) {
	tests := []string{
		"regular output line",
		"",
		"# comment",
		"##OCH without colon",
	}
	for _, line := range tests {
		msg := ParseLine(line)
		if msg != nil {
			t.Errorf("ParseLine(%q) should return nil, got %+v", line, msg)
		}
	}
}

func TestParseLineUnknownType(t *testing.T) {
	msg := ParseLine("##OCH:UNKNOWN:some data")
	if msg != nil {
		t.Errorf("ParseLine with unknown type should return nil, got %+v", msg)
	}
}
