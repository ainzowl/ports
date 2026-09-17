package main

import (
	_ "embed"
	"runtime"

	"fyne.io/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Windows tray icons must be ICO format (LoadImage rejects PNG). Wails
// regenerates this .ico from build/appicon.png (our icon.png) on every
// build, so tray, exe and titlebar always share the same artwork.
//
//go:embed build/windows/icon.ico
var iconData []byte

// appQuit is closed when the user chooses Quit (tray or close modal).
var appQuit = make(chan struct{})

func startTray(a *App) {
	go func() {
		// systray's message pump has thread affinity on Windows: the window
		// it creates and the GetMessage loop must stay on one OS thread.
		// Without LockOSThread the goroutine can migrate and the tray icon
		// goes dead (visible but ignores all clicks).
		runtime.LockOSThread()
		systray.Run(func() { onTrayReady(a) }, nil)
	}()
}

func onTrayReady(a *App) {
	systray.SetIcon(iconData)
	systray.SetTitle("Ports")
	systray.SetTooltip("Ports - monitor and kill port listeners")

	// Left click (or double click) on the tray icon shows the window,
	// no need to use the context menu.
	systray.SetOnTapped(func() {
		wailsruntime.WindowShow(a.ctx)
		wailsruntime.WindowUnminimise(a.ctx)
	})

	mShow := systray.AddMenuItem("Show Ports", "Show the Ports window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit Ports")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				wailsruntime.WindowShow(a.ctx)
			case <-mQuit.ClickedCh:
				systray.Quit()
				wailsruntime.Quit(a.ctx)
				return
			case <-appQuit:
				systray.Quit()
				return
			}
		}
	}()
}

// quitApp performs a full application exit from the tray or close modal.
func quitApp(a *App) {
	close(appQuit)
	wailsruntime.Quit(a.ctx)
}
