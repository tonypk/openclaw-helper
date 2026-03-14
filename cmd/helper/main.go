// och-helper is the Go backend process for OpenClaw Helper.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"github.com/tonypk/openclaw-helper/internal/checker"
	"github.com/tonypk/openclaw-helper/internal/installer"
	"github.com/tonypk/openclaw-helper/internal/ipc"
	"github.com/tonypk/openclaw-helper/internal/types"
)

var version = "0.1.0"

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
	// Create installer orchestrator
	orch := installer.NewOrchestrator([]installer.PhaseExecutor{
		installer.NewPrecheckExecutor(sc),
		&installer.WSLInstaller{},
		&installer.UbuntuConfigurer{},
		&installer.NodeInstaller{},
		&installer.OpenClawInstaller{},
		&installer.ConfigPhase{},
		&installer.VerifyPhase{},
	})

	router := ipc.NewRouter()
	registerHandlers(router, sc, orch)

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

func registerHandlers(router *ipc.Router, sc *checker.SystemChecker, orch *installer.Orchestrator) {
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
}
