package main

import (
	"embed"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"golang.org/x/sys/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// singleInstanceMutex is held for the process lifetime. selfupdate releases
// it right before spawning the new child so the restart doesn't collide.
var singleInstanceMutex windows.Handle

func main() {
	// Single-instance guard. Retry briefly to survive a self-update restart
	// where the old parent is still closing its mutex handle.
	mutexName, _ := windows.UTF16PtrFromString("MavrogUpdater_SingleInstance")
	var h windows.Handle
	var err error
	for i := 0; i < 20; i++ {
		h, err = windows.CreateMutex(nil, false, mutexName)
		if err != windows.ERROR_ALREADY_EXISTS {
			break
		}
		if h != 0 {
			windows.CloseHandle(h)
		}
		time.Sleep(150 * time.Millisecond)
	}
	if err == windows.ERROR_ALREADY_EXISTS {
		os.Exit(0)
	}
	singleInstanceMutex = h
	defer func() {
		if singleInstanceMutex != 0 {
			windows.CloseHandle(singleInstanceMutex)
		}
	}()

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
