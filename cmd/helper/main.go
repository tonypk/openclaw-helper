// och-helper is the Go backend process for OpenClaw Helper.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"path/filepath"

	"github.com/tonypk/openclaw-helper/internal/chat"
	"github.com/tonypk/openclaw-helper/internal/checker"
	"github.com/tonypk/openclaw-helper/internal/diagnosis"
	"github.com/tonypk/openclaw-helper/internal/installer"
	"github.com/tonypk/openclaw-helper/internal/ipc"
	"github.com/tonypk/openclaw-helper/internal/report"
	"github.com/tonypk/openclaw-helper/internal/scriptrun"
	"github.com/tonypk/openclaw-helper/internal/types"
)

var version = "0.4.0"

func main() {
	pipePath := flag.String("pipe", "", "IPC pipe/socket address (default: platform default)")
	cliCheck := flag.Bool("check", false, "Run system check in CLI mode and exit")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("och-helper %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	sc := checker.New()

	if *cliCheck {
		runCLICheck(sc)
		return
	}

	runServer(sc, *pipePath)
}

func runCLICheck(sc *checker.SystemChecker) {
	report := sc.RunAll()
	data, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(data))

	if report.OverallReady {
		fmt.Println("\n✅ System is ready for OpenClaw installation")
	} else {
		fmt.Println("\n❌ Some checks failed — see details above")
		os.Exit(1)
	}
}

func runServer(sc *checker.SystemChecker, pipePath string) {
	// Set up remote script cache for hot-updatable installation scripts.
	scriptCacheDir := scriptCacheDir()
	fallback := scriptrun.NewFallbackScripts()
	cache := scriptrun.NewCache(scriptCacheDir, "", fallback)
	runner := scriptrun.NewRunner(cache)

	// Background sync: fetch latest scripts from GitHub.
	go func() {
		if err := cache.Sync(); err != nil {
			log.Printf("[startup] script cache sync failed (will use fallback): %v", err)
		} else {
			log.Printf("[startup] script cache synced to %s", scriptCacheDir)
		}
	}()

	// Precheck stays native Go (needs Windows API for memory/disk checks).
	// All other phases use remote scripts via ScriptPhaseExecutor.
	orch := installer.NewOrchestrator([]installer.PhaseExecutor{
		installer.NewPrecheckExecutor(sc),
		scriptrun.NewScriptPhaseExecutor(installer.PhaseWSL, runner, cache),
		scriptrun.NewScriptPhaseExecutor(installer.PhaseUbuntu, runner, cache),
		scriptrun.NewScriptPhaseExecutor(installer.PhaseNode, runner, cache),
		scriptrun.NewScriptPhaseExecutor(installer.PhaseOpenClaw, runner, cache),
		scriptrun.NewScriptPhaseExecutor(installer.PhaseConfig, runner, cache),
		scriptrun.NewScriptPhaseExecutor(installer.PhaseVerify, runner, cache),
	})

	// Create diagnosis engine and chat handler
	diagEngine := diagnosis.NewEngine()
	playbooks := diagnosis.NewPlaybookRegistry()

	llmProxy := chat.NewLLMProxy([]chat.LLMProvider{
		{Name: "DeepSeek", BaseURL: "https://api.deepseek.com/v1", Model: "deepseek-chat"},
	})
	faqStore := chat.NewFAQStore()
	chatHandler := chat.NewHandler(faqStore, llmProxy, diagEngine)

	router := ipc.NewRouter()
	registerHandlers(router, sc, orch, diagEngine, playbooks, chatHandler)

	srv := ipc.NewServer(router)
	if err := srv.Listen(pipePath); err != nil {
		log.Fatalf("Failed to start IPC server: %v", err)
	}

	log.Printf("och-helper %s started (IPC: %s)", version, srv.Address())

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go srv.Serve()

	sig := <-sigCh
	log.Printf("Received %s, shutting down...", sig)
	orch.Cancel()
	srv.Stop()
	log.Println("Goodbye")
}

func registerHandlers(router *ipc.Router, sc *checker.SystemChecker, orch *installer.Orchestrator, diagEngine *diagnosis.Engine, playbooks *diagnosis.PlaybookRegistry, chatHandler *chat.Handler) {
	// --- Helper ---
	router.Register("helper.ping", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		return "pong", nil
	})

	router.Register("helper.version", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		return types.HelperInfo{
			Version:   version,
			GoVersion: runtime.Version(),
			OS:        runtime.GOOS,
			Arch:      runtime.GOARCH,
		}, nil
	})

	// --- System Check ---
	router.Register("system.check", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		report := sc.RunAll()
		return report, nil
	})

	router.Register("system.checkSingle", func(params json.RawMessage) (interface{}, *types.RPCError) {
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &p); err != nil || p.Name == "" {
			return nil, &types.RPCError{
				Code:    types.ErrCodeInvalidParams,
				Message: "params.name is required",
			}
		}
		result, found := sc.RunSingle(p.Name)
		if !found {
			return nil, &types.RPCError{
				Code:    types.ErrCodeInvalidParams,
				Message: "unknown check: " + p.Name,
			}
		}
		return result, nil
	})

	// --- Install ---
	// Event buffer for polling (Tauri will poll this)
	var eventMu sync.Mutex
	var eventBuf []installer.ProgressEvent
	orch.OnProgress(func(evt installer.ProgressEvent) {
		eventMu.Lock()
		eventBuf = append(eventBuf, evt)
		eventMu.Unlock()
		log.Printf("[install] %s: %s (%d%%)", evt.Phase, evt.Message, evt.Overall)
	})

	router.Register("install.start", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		if err := orch.Start(); err != nil {
			return nil, &types.RPCError{
				Code:    types.ErrCodeInternal,
				Message: err.Error(),
			}
		}
		return "started", nil
	})

	router.Register("install.status", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		return orch.Status(), nil
	})

	router.Register("install.retry", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		if err := orch.Retry(); err != nil {
			return nil, &types.RPCError{
				Code:    types.ErrCodeInternal,
				Message: err.Error(),
			}
		}
		return "retrying", nil
	})

	router.Register("install.cancel", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		orch.Cancel()
		return "cancelled", nil
	})

	router.Register("install.reset", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		if err := orch.Reset(); err != nil {
			return nil, &types.RPCError{
				Code:    types.ErrCodeInternal,
				Message: err.Error(),
			}
		}
		return "reset", nil
	})

	router.Register("install.events", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		eventMu.Lock()
		events := make([]installer.ProgressEvent, len(eventBuf))
		copy(events, eventBuf)
		eventBuf = eventBuf[:0]
		eventMu.Unlock()
		return events, nil
	})

	// --- Diagnosis ---
	router.Register("diagnosis.run", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		diagCtx := diagnosis.Collect(sc)
		report := diagEngine.Diagnose(diagCtx)
		return report, nil
	})

	router.Register("diagnosis.runWithError", func(params json.RawMessage) (interface{}, *types.RPCError) {
		var p struct {
			ErrorLog string `json:"error_log"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &types.RPCError{Code: types.ErrCodeInvalidParams, Message: "invalid params"}
		}
		diagCtx := diagnosis.CollectWithError(sc, p.ErrorLog)
		report := diagEngine.Diagnose(diagCtx)
		return report, nil
	})

	router.Register("diagnosis.repair", func(params json.RawMessage) (interface{}, *types.RPCError) {
		var p struct {
			RepairID string `json:"repair_id"`
		}
		if err := json.Unmarshal(params, &p); err != nil || p.RepairID == "" {
			return nil, &types.RPCError{Code: types.ErrCodeInvalidParams, Message: "params.repair_id required"}
		}
		result := playbooks.Run(context.Background(), p.RepairID)
		return result, nil
	})

	// --- Chat ---
	router.Register("chat.ask", func(params json.RawMessage) (interface{}, *types.RPCError) {
		var p struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(params, &p); err != nil || p.Message == "" {
			return nil, &types.RPCError{Code: types.ErrCodeInvalidParams, Message: "params.message required"}
		}
		resp := chatHandler.Ask(p.Message)
		return resp, nil
	})

	router.Register("chat.setContext", func(params json.RawMessage) (interface{}, *types.RPCError) {
		var p struct {
			Phase    string `json:"phase,omitempty"`
			ErrorLog string `json:"error_log,omitempty"`
			Language string `json:"language,omitempty"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &types.RPCError{Code: types.ErrCodeInvalidParams, Message: "invalid params"}
		}
		if p.Phase != "" {
			chatHandler.SetPhase(p.Phase)
		}
		if p.ErrorLog != "" {
			chatHandler.SetErrorLog(p.ErrorLog)
		}
		if p.Language != "" {
			chatHandler.SetLanguage(p.Language)
		}
		return "ok", nil
	})

	router.Register("chat.suggestions", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		return chatHandler.GetSuggestions(), nil
	})

	// --- Report ---
	router.Register("report.collect", func(_ json.RawMessage) (interface{}, *types.RPCError) {
		r := report.Collect(version, sc, orch, diagEngine)
		return r, nil
	})

	router.Register("report.submit", func(params json.RawMessage) (interface{}, *types.RPCError) {
		var p struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &types.RPCError{Code: types.ErrCodeInvalidParams, Message: "invalid params"}
		}
		if p.Title == "" {
			p.Title = "Installation issue"
		}

		r := report.Collect(version, sc, orch, diagEngine)
		r.Title = p.Title
		r.Description = p.Description

		result := report.ReportResult{}

		// Primary: send to Telegram channel (synchronous)
		if report.TelegramConfigured() {
			if err := report.SendToTelegram(context.Background(), r); err != nil {
				log.Printf("[report] telegram send failed: %v", err)
				result.ErrorMessage = err.Error()
				// Fallback: provide GitHub URL for manual submission
				result.FallbackURL = report.BuildIssueURL(r)
			} else {
				result.Submitted = true
				result.TelegramSent = true
				log.Printf("[report] feedback sent to telegram")
			}
		} else {
			// Telegram not configured, fallback to GitHub URL
			log.Printf("[report] telegram not configured, using github URL fallback")
			result.FallbackURL = report.BuildIssueURL(r)
		}

		return result, nil
	})
}

// scriptCacheDir returns the directory for cached remote scripts.
func scriptCacheDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "openclaw-helper", "scripts")
}
