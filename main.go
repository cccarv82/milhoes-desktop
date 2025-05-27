package main

import (
	"embed"
	"fmt"
	"lottery-optimizer-gui/internal/config"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

var (
	version = "v1.1.5"
)

func main() {
	fmt.Println("🚀 =================================")
	fmt.Println("🚀 LOTTERY OPTIMIZER MAIN INICIADO")
	fmt.Printf("🚀 Versão: %s\n", version)
	fmt.Println("🚀 =================================")

	// Inicializar configuração
	config.Init()
	fmt.Println("✅ Configuração inicializada")

	// Create an instance of the app structure
	app := NewApp()
	fmt.Println("✅ App instance criada")

	// Create application with options
	fmt.Println("🚀 Iniciando Wails com interface gráfica...")
	fmt.Println("🔧 Configurações da janela:")
	fmt.Println("   - Tamanho: 1200x800")
	fmt.Println("   - Mínimo: 1000x700")
	fmt.Println("   - StartHidden: false")
	fmt.Println("   - Debug: true")

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
		// Configurações de desenvolvimento - ATIVAR DEBUG COMPLETO
		Debug: options.Debug{
			OpenInspectorOnStartup: true, // Ativar para debug
		},
	})

	if err != nil {
		fmt.Printf("❌ ERRO CRÍTICO ao iniciar Wails: %v\n", err)
		fmt.Println("💡 Possíveis causas:")
		fmt.Println("   - WebView2 não instalado ou desatualizado")
		fmt.Println("   - Problemas com assets do frontend")
		fmt.Println("   - Conflitos de antivírus")
		fmt.Println("🔧 Pressione Enter para sair...")
		fmt.Scanln()
	} else {
		fmt.Println("✅ Wails executado com sucesso!")
	}
}
