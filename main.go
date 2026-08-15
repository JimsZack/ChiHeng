package main

import (
	"context"
	"embed"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/JimsZack/ChiHeng/internal/config"
	"github.com/JimsZack/ChiHeng/internal/schema"
	"github.com/JimsZack/ChiHeng/internal/service"
	"github.com/JimsZack/ChiHeng/internal/store"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	ctx := context.Background()
	setupLogging()

	dbPath, err := config.DBPath()
	if err != nil {
		slog.Error("解析数据目录失败", "error", err)
		os.Exit(1)
	}
	database, err := store.Open(ctx, dbPath)
	if err != nil {
		slog.Error("打开数据库失败", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if _, err := schema.EnsureSchema(ctx, database.DB()); err != nil {
		slog.Error("初始化数据库结构失败", "error", err)
		os.Exit(1)
	}

	dataDir, err := config.DataDir()
	if err != nil {
		slog.Error("解析数据目录失败", "error", err)
		os.Exit(1)
	}
	secrets, err := service.NewFileSecretStore(filepath.Join(dataDir, "secrets"))
	if err != nil {
		slog.Error("初始化密钥存储失败", "error", err)
		os.Exit(1)
	}

	app := NewApp(database, secrets)

	err = wails.Run(&options.App{
		Title:     config.AppDisplayName,
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Bind: []interface{}{
			app,
		},
		OnStartup:  app.onStartup,
		OnShutdown: app.onShutdown,
		Menu:       app.appMenu(),
		// 关闭窗口时隐藏到后台，可通过菜单恢复（替代系统托盘行为）。
		HideWindowOnClose: true,
		Logger:            logger.NewDefaultLogger(),
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Theme:                windows.SystemDefault,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarDefault(),
			Appearance:           mac.DefaultAppearance,
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			About: &mac.AboutInfo{
				Title:   config.AppDisplayName,
				Message: config.Tagline,
			},
		},
		Linux: &linux.Options{
			ProgramName:      config.AppDisplayName,
			WindowIsTranslucent: false,
			WebviewGpuPolicy: linux.WebviewGpuPolicyAlways,
		},
	})
	if err != nil {
		slog.Error("应用启动失败", "error", err)
		os.Exit(1)
	}
}

// setupLogging 将日志同时输出到控制台与用户数据目录下的日志文件。
func setupLogging() {
	logDir, err := config.LogDir()
	if err != nil {
		return
	}
	file, err := os.OpenFile(filepath.Join(logDir, "chiheng.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(io.MultiWriter(os.Stderr, file), nil)))
}
