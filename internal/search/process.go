package search

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Config holds search bridge configuration derived from environment variables.
type Config struct {
	// Port the Node.js bridge listens on (default 7331).
	Port int
	// UseRealProfile tells the bridge to use the real Chrome user profile instead
	// of the isolated ogcode profile. Chrome must be fully closed when enabled.
	UseRealProfile bool
}

// ConfigFromEnv reads OGCODE_SEARCH_BRIDGE_PORT and OGCODE_SEARCH_USE_REAL_PROFILE.
func ConfigFromEnv() Config {
	port := 7331
	if v := os.Getenv("OGCODE_SEARCH_BRIDGE_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port)
	}
	return Config{
		Port:           port,
		UseRealProfile: strings.EqualFold(os.Getenv("OGCODE_SEARCH_USE_REAL_PROFILE"), "true"),
	}
}

// BridgeProcess manages the lifecycle of the Node.js search bridge subprocess.
type BridgeProcess struct {
	cmd    *exec.Cmd
	client *BridgeClient
	cancel context.CancelFunc
}

// StartBridge starts the Node.js bridge process and waits until it is healthy.
// The process is killed when ctx is cancelled.
func StartBridge(ctx context.Context, cfg Config, bridgeDir string) (*BridgeProcess, error) {
	if bridgeDir == "" {
		bridgeDir = resolveBridgeDir()
	}
	bridgeDir = filepath.Clean(bridgeDir)

	if _, err := os.Stat(filepath.Join(bridgeDir, "server.js")); err != nil {
		return nil, fmt.Errorf("search bridge server.js not found at %s (set OGCODE_SEARCH_BRIDGE_DIR to override): %w", bridgeDir, err)
	}

	node, err := resolveNode()
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("http://127.0.0.1:%d", cfg.Port)

	procCtx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(procCtx, node, "server.js")
	cmd.Dir = bridgeDir
	envVars := []string{fmt.Sprintf("OGCODE_SEARCH_BRIDGE_PORT=%d", cfg.Port)}
	if cfg.UseRealProfile {
		envVars = append(envVars, "OGCODE_SEARCH_USE_REAL_PROFILE=true")
	}
	cmd.Env = append(os.Environ(), envVars...)
	cmd.Stdout = os.Stderr // route bridge logs to stderr alongside Go server logs
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start search bridge: %w", err)
	}
	slog.Info("search bridge process started", "pid", cmd.Process.Pid, "addr", addr)

	client := NewBridgeClient(addr)

	// Wait up to 30 s for the bridge to become healthy.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if err := client.Health(procCtx); err == nil {
			slog.Info("search bridge ready", "addr", addr)
			return &BridgeProcess{cmd: cmd, client: client, cancel: cancel}, nil
		}
		select {
		case <-procCtx.Done():
			cancel()
			return nil, fmt.Errorf("context cancelled while waiting for search bridge")
		case <-time.After(500 * time.Millisecond):
		}
	}

	cancel()
	_ = cmd.Process.Kill()
	return nil, fmt.Errorf("search bridge did not become healthy within 30s at %s", addr)
}

// Client returns the HTTP client connected to the running bridge.
func (p *BridgeProcess) Client() *BridgeClient { return p.client }

// Stop shuts down the bridge process gracefully.
func (p *BridgeProcess) Stop() {
	p.cancel()
	if p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
}

// resolveBridgeDir finds the search-bridge directory by checking locations in
// priority order:
//  1. ~/.local/share/ogcode/search-bridge  (installed via make install)
//  2. <binary-dir>/../tools/search-bridge  (running from source repo)
//  3. <binary-dir>/search-bridge           (flat install layout)
func resolveBridgeDir() string {
	candidates := []string{}

	// 1. Standard ~/.local install location (make install)
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates,
			filepath.Join(home, ".local", "share", "ogcode", "search-bridge"),
		)
	}

	// 2. Homebrew share directory (brew install)
	// HOMEBREW_PREFIX is set by Homebrew; fall back to common locations.
	if prefix := os.Getenv("HOMEBREW_PREFIX"); prefix != "" {
		candidates = append(candidates,
			filepath.Join(prefix, "share", "ogcode", "search-bridge"),
		)
	}
	for _, prefix := range []string{"/opt/homebrew", "/usr/local"} {
		candidates = append(candidates,
			filepath.Join(prefix, "share", "ogcode", "search-bridge"),
		)
	}

	// 3 & 4. Relative to binary (running from source repo or flat install)
	if exe, err := os.Executable(); err == nil {
		binDir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(binDir, "..", "tools", "search-bridge"),
			filepath.Join(binDir, "search-bridge"),
		)
	}

	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "server.js")); err == nil {
			return filepath.Clean(c)
		}
	}

	// Return the preferred install path even if it doesn't exist yet — the error
	// from StartBridge will tell the user exactly where to look.
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "ogcode", "search-bridge")
	}
	return "search-bridge"
}

func resolveNode() (string, error) {
	// Prefer nvm/volta/brew paths on macOS where $PATH may be trimmed.
	candidates := []string{"node"}
	if runtime.GOOS == "darwin" {
		home, _ := os.UserHomeDir()
		candidates = append(candidates,
			filepath.Join(home, ".nvm", "versions", "node", "current", "bin", "node"),
			"/opt/homebrew/bin/node",
			"/usr/local/bin/node",
		)
	}
	for _, c := range candidates {
		if p, err := exec.LookPath(c); err == nil {
			return p, nil
		}
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("node not found; install Node.js to enable the search agent")
}

