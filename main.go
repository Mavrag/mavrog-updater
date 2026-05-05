package main

import (
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"golang.org/x/sys/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Single-instance guard: exit if already running.
	mutexName, _ := windows.UTF16PtrFromString("MavrogUpdater_SingleInstance")
	h, err := windows.CreateMutex(nil, false, mutexName)
	if h != 0 {
		defer windows.CloseHandle(h)
	}
	if err == windows.ERROR_ALREADY_EXISTS {
		os.Exit(0)
	}

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err = wails.Run(&options.App{
		Title:     "Mavrog Updater",
		Width:     640,
		Height:    460,
		MinWidth:  540,
		MinHeight: 400,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 17, G: 19, B: 28, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
