package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"sync"

	desktopapp "diskbenchmark/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var frontendAssets embed.FS

//go:embed build/appicon.png
var applicationIcon []byte

type desktopRuntime struct {
	mu  sync.RWMutex
	ctx context.Context
}

func (r *desktopRuntime) startup(ctx context.Context) {
	r.mu.Lock()
	r.ctx = ctx
	r.mu.Unlock()
}

func (r *desktopRuntime) context() (context.Context, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.ctx == nil {
		return nil, fmt.Errorf("desktop runtime is not ready")
	}
	return r.ctx, nil
}

func (r *desktopRuntime) emit(name string, payload any) {
	ctx, err := r.context()
	if err != nil {
		log.Printf("emit %s: %v", name, err)
		return
	}
	wailsruntime.EventsEmit(ctx, name, payload)
}

func main() {
	desktop := &desktopRuntime{}
	service := desktopapp.NewService(
		desktopapp.WithEventEmitter(desktop.emit),
		desktopapp.WithDialogProvider(desktopapp.DialogProvider{
			SelectDirectory: func() (string, error) {
				ctx, err := desktop.context()
				if err != nil {
					return "", err
				}
				return wailsruntime.OpenDirectoryDialog(ctx, wailsruntime.OpenDialogOptions{
					Title:                "Select benchmark directory",
					CanCreateDirectories: true,
				})
			},
			SelectSuiteFile: func() (string, error) {
				ctx, err := desktop.context()
				if err != nil {
					return "", err
				}
				return wailsruntime.OpenFileDialog(ctx, wailsruntime.OpenDialogOptions{
					Title: "Select workload suite",
					Filters: []wailsruntime.FileFilter{{
						DisplayName: "JSON workload suites (*.json)",
						Pattern:     "*.json",
					}},
				})
			},
			SelectSaveFile: func(request desktopapp.SaveDialogRequest) (string, error) {
				ctx, err := desktop.context()
				if err != nil {
					return "", err
				}
				return wailsruntime.SaveFileDialog(ctx, wailsruntime.SaveDialogOptions{
					Title:                request.Title,
					DefaultFilename:      request.DefaultFilename,
					CanCreateDirectories: true,
					Filters: []wailsruntime.FileFilter{{
						DisplayName: request.DisplayName,
						Pattern:     request.Pattern,
					}},
				})
			},
		}),
	)

	err := wails.Run(&options.App{
		Title:              "DiskBenchmark",
		Width:              800,
		Height:             600,
		DisableResize:      true,
		MinWidth:           800,
		MinHeight:          600,
		MaxWidth:           800,
		MaxHeight:          600,
		BackgroundColour:   options.NewRGB(244, 247, 250),
		AssetServer:        &assetserver.Options{Assets: frontendAssets},
		OnStartup:          desktop.startup,
		Bind:               []interface{}{service},
		SingleInstanceLock: &options.SingleInstanceLock{UniqueId: "io.github.diskbenchmark.desktop"},
		Windows: &windows.Options{
			Theme:           windows.SystemDefault,
			WindowClassName: "DiskBenchmarkWindow",
		},
		Mac: &mac.Options{
			About: &mac.AboutInfo{
				Title:   "DiskBenchmark",
				Message: "Native disk benchmark frontend",
				Icon:    applicationIcon,
			},
		},
		Linux: &linux.Options{
			Icon:             applicationIcon,
			ProgramName:      "diskbenchmark",
			WebviewGpuPolicy: linux.WebviewGpuPolicyOnDemand,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
