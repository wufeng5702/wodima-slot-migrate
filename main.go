package main

import (
	"embed"

	"github.com/wailsapp/wails/v3/pkg/application"

	"wodima-slot-migrate/internal/android"
	"wodima-slot-migrate/internal/migrate"
	"wodima-slot-migrate/internal/steam"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := application.New(application.Options{
		Name:        "wodima-slot-migrate",
		Description: "",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	// Wire services after creating the app so that services can access
	// the application instance (Dialog, Logger, Events, etc.).
	app.RegisterService(application.NewService(android.NewService(app)))
	app.RegisterService(application.NewService(migrate.NewService(app)))
	app.RegisterService(application.NewService(steam.NewService(app)))

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "wodima-slot-migrate",
		Width:  1024,
		Height: 768,
	})

	_ = window

	err := app.Run()
	if err != nil {
		panic(err)
	}
}
