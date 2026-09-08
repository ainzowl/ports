package main

import (
	_ "embed"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
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
	go systray.Run(func() { onTrayReady(a) }, nil)
}

func onTrayReady(a *App) {
	systray.SetIcon(iconData)
	systray.SetTitle("Ports")
	systray.SetTooltip("Ports - monitor and kill port listeners")

	// Left click (or double click) on the tray icon shows the window,
	// no need to use the context menu.
	systray.SetOnTapped(func() {
		runtime.WindowShow(a.ctx)
		runtime.WindowUnminimise(a.ctx)
	})

	mShow := systray.AddMenuItem("Show Ports", "Show the Ports window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit Ports")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				runtime.WindowShow(a.ctx)
			case <-mQuit.ClickedCh:
				systray.Quit()
				runtime.Quit(a.ctx)
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
	runtime.Quit(a.ctx)
}
