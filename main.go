package main

import (
	"embed"
	"fmt"

	"Fuploader/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	fmt.Println("Starting Fuploader...")

	application := app.NewApp()
	fmt.Println("App created")

	err := wails.Run(&options.App{
		Title:  "Fuploader",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: app.NewFileLoader(), // 添加自定义 handler 处理静态文件
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        application.Startup,
		OnShutdown:       application.Shutdown,
		Bind: []interface{}{
			application,
		},
	})

	if err != nil {
		fmt.Println("Error:", err.Error())
	} else {
		fmt.Println("Application exited normally")
	}
}
