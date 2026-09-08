package main

import (
	"embed"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:         "Ports",
		Width:         1160,
		Height:        760,
		MinWidth:      860,
		MinHeight:     560,
		Frameless:     true,
		DisableResize: false,
		// Closing the window hides it; the close modal (or tray Quit) ends
		// the process. This keeps Ports alive in the tray.
		HideWindowOnClose: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 11, G: 14, B: 19, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.Dark,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

// startHidden reports whether the app was launched with --hidden
// (used by the "start on launch" setting).
func startHidden() bool {
	for _, arg := range os.Args[1:] {
		if strings.EqualFold(arg, "--hidden") || strings.EqualFold(arg, "-hidden") {
			return true
		}
	}
	return false
}
