package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type Settings struct {
	AutoRefresh  bool   `json:"autoRefresh"`
	IntervalSecs int    `json:"intervalSecs"`
	ShowPaths    bool   `json:"showPaths"`
	ShowSystem   bool   `json:"showSystem"`
	CloseAction  string `json:"closeAction"`
	StartOnBoot  bool   `json:"startOnBoot"`
}

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	startTray(a)
}

func settingsPath() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return "ports-settings.json"
	}
	return filepath.Join(base, "ports", "settings.json")
}

func (a *App) GetSettings() Settings {
	s := Settings{
		AutoRefresh:  true,
		IntervalSecs: 5,
		ShowPaths:    true,
		ShowSystem:   true,
		CloseAction:  "ask",
		StartOnBoot:  startOnBootEnabled(),
	}
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return s
	}
	_ = json.Unmarshal(data, &s)
	if s.IntervalSecs < 2 {
		s.IntervalSecs = 5
	}
	if s.CloseAction != "quit" && s.CloseAction != "hide" {
		s.CloseAction = "ask"
	}
	return s
}

func (a *App) SaveSettings(s Settings) error {
	if s.IntervalSecs < 2 {
		s.IntervalSecs = 5
	}
	if s.CloseAction != "quit" && s.CloseAction != "hide" {
		s.CloseAction = "ask"
	}
	if err := setStartOnBoot(s.StartOnBoot); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	p := settingsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func (a *App) ListPorts() ([]PortProcess, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 45*time.Second)
	defer cancel()
	return listAll(ctx), nil
}

func (a *App) KillProcess(source, distro string, pid int) error {
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()
	return killProcess(ctx, source, distro, pid)
}

func (a *App) OpenFolder(source, distro, path string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	return openFolder(ctx, source, distro, path)
}

// CloseWindow is called by the custom close button. The frontend decides
// between ask/quit/hide; quit fully exits, hide just hides the window.
func (a *App) CloseWindow(choice string) {
	switch choice {
	case "quit":
		quitApp(a)
	default:
		runtime.WindowHide(a.ctx)
	}
}

// QuitApp fully exits from the tray menu or UI.
func (a *App) QuitApp() {
	quitApp(a)
}

// ShowWindow restores the window from the tray or frontend.
func (a *App) ShowWindow() {
	runtime.WindowShow(a.ctx)
}

// StartHidden lets the frontend know if we launched with --hidden.
func (a *App) StartHidden() bool {
	return startHidden()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
