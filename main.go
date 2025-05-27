package main

import (
	"embed"
	"fmt"
	"lottery-optimizer-gui/internal/config"
	"lottery-optimizer-gui/internal/logs"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

var (
	version = "v1.2.0"
)

func main() {
	fmt.Println("🚀 =================================")
	fmt.Println("🚀 LOTTERY OPTIMIZER MAIN INICIADO")
	fmt.Printf("🚀 Versão: %s\n", version)
	fmt.Println("🚀 =================================")

	// Inicializar sistema de logs especializado
	if err := logs.Init(); err != nil {
		fmt.Printf("⚠️ Erro ao inicializar logs: %v\n", err)
	} else {
		fmt.Println("✅ Sistema de logs especializado inicializado")
	}

	// Inicializar configuração
	config.Init()
	logs.LogMain("✅ Configuração inicializada")

	// Create an instance of the app structure
	app := NewApp()
	logs.LogMain("✅ App instance criada")

	// Create application with options
	logs.LogMain("🚀 Iniciando Wails com interface gráfica...")
	logs.LogMain("🔧 Configurações da janela: 1200x800, mínimo: 1000x700")

	err := wails.Run(&options.App{
		Title:     "🎰 Lottery Optimizer - Estratégias Inteligentes",
		Width:     1200,
		Height:    800,
		MinWidth:  1000,
		MinHeight: 700,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 15, G: 23, B: 42, A: 1}, // Azul escuro elegante
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		// Configurações específicas para Windows - SIMPLIFICADAS
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			DisablePinchZoom:     false,
			Theme:                windows.SystemDefault,
		},
		// FORÇAR APARECIMENTO DA JANELA
		HideWindowOnClose: false,
		AlwaysOnTop:       false,
		Fullscreen:        false,
		StartHidden:       false, // GARANTIR que não inicia hidden
		// Configurações de desenvolvimento - DESATIVAR DEBUG
		Debug: options.Debug{
			OpenInspectorOnStartup: false, // Desativar debug automático
		},
	})

	if err != nil {
		logs.LogError(logs.CategoryMain, "❌ ERRO CRÍTICO ao iniciar Wails: %v", err)
		fmt.Printf("❌ ERRO CRÍTICO ao iniciar Wails: %v\n", err)
		fmt.Println("💡 Verifique os logs especializados em:")
		fmt.Printf("   📁 %s\n", logs.GetLogDir())
		fmt.Println("🔧 Pressione Enter para sair...")
		fmt.Scanln()
	} else {
		logs.LogMain("✅ Wails executado com sucesso!")
	}
}
