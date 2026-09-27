package main

import (
	"embed"
	"fmt"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

// appVersion is injected at link time by `make build|dev|release` (-X main.appVersion).
var appVersion = "dev"

// releaseRepo is the GitHub "owner/name" whose latest release the updater checks.
// Override at build time with -ldflags "-X main.releaseRepo=owner/name".
var releaseRepo = "cybersafetyid/BersihDisk"

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "BersihDisk — Pembersih Disk",
		Width:     1180,
		Height:    780,
		MinWidth:  900,
		MinHeight: 620,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 10, G: 10, B: 10, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
			About: &mac.AboutInfo{
				Title:   "BersihDisk",
				Message: fmt.Sprintf("Pembersih disk untuk developer\nVersi %s", appVersion),
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
