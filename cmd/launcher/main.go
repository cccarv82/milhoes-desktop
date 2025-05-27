package main

import (
	"context"
	"fmt"
	"lottery-optimizer-gui/internal/logs"
	"lottery-optimizer-gui/internal/updater"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

const (
	appExecutable   = "milhoes.exe"
	appName         = "Lottery Optimizer"
	launcherVersion = "v1.1.8"
	githubRepo      = "cccarv82/milhoes-releases" // Repositório de releases públicas
)

type Launcher struct {
	appDir  string
	appPath string
	updater *updater.Updater
}

func NewLauncher() (*Launcher, error) {
	launcherPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter caminho do launcher: %w", err)
	}

	appDir := filepath.Dir(launcherPath)
	appPath := filepath.Join(appDir, appExecutable)

	// Inicializar updater com versão do app principal (não do launcher)
	updaterInstance := updater.NewUpdater(launcherVersion, githubRepo)

	return &Launcher{
		appDir:  appDir,
		appPath: appPath,
		updater: updaterInstance,
	}, nil
}

func (l *Launcher) checkMainApp() error {
	if _, err := os.Stat(l.appPath); os.IsNotExist(err) {
		return fmt.Errorf("app principal não encontrado: %s", l.appPath)
	}
	return nil
}

func (l *Launcher) checkForUpdates() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("🌐 Verificando GitHub API: %s\\n", githubRepo)
	logs.LogLauncher("🌐 Verificando GitHub API: %s", githubRepo)

	// Verificar se há atualizações disponíveis
	updateInfo, err := l.updater.CheckForUpdates(ctx)
	if err != nil {
		return fmt.Errorf("erro na verificação de atualizações: %w", err)
	}

	if !updateInfo.Available {
		fmt.Printf("✅ Aplicação está atualizada: %s\\n", updateInfo.Version)
		logs.LogLauncher("✅ Aplicação está atualizada: %s", updateInfo.Version)
		return nil
	}

	// Nova versão disponível - informar usuário
	fmt.Printf("🚀 Nova versão disponível: %s -> %s\\n", updateInfo.Version, updateInfo.Version)
	logs.LogLauncher("🚀 Nova versão disponível: %s", updateInfo.Version)

	// Baixar atualização
	fmt.Printf("📥 Baixando atualização...\\n")
	logs.LogLauncher("📥 Baixando atualização...")

	progressFunc := func(downloaded, total int64) {
		percentage := float64(downloaded) / float64(total) * 100
		fmt.Printf("\\r📊 Progresso: %.1f%% (%d/%d bytes)", percentage, downloaded, total)
	}

	if err := l.updater.DownloadUpdate(ctx, updateInfo, progressFunc); err != nil {
		return fmt.Errorf("erro no download: %w", err)
	}

	fmt.Printf("\\n✅ Download concluído\\n")
	logs.LogLauncher("✅ Download concluído")

	// APLICAR ATUALIZAÇÃO IMEDIATAMENTE
	fmt.Printf("🔧 Aplicando atualização IMEDIATAMENTE...\\n")
	logs.LogLauncher("🔧 Aplicando atualização IMEDIATAMENTE...")

	if err := l.updater.InstallImmediate(updateInfo); err != nil {
		fmt.Printf("⚠️ Erro ao aplicar atualização: %v\\n", err)
		fmt.Printf("⚠️ Continuando com versão atual...\\n")
		logs.LogError(logs.CategoryLauncher, "⚠️ Erro ao aplicar atualização: %v", err)
		return nil // Não falha completamente, apenas continua com versão atual
	}

	fmt.Printf("✅ Atualização aplicada com sucesso!\\n")
	fmt.Printf("🔄 App será iniciado na versão mais recente...\\n")
	logs.LogLauncher("✅ Atualização aplicada com sucesso!")

	return nil
}

func (l *Launcher) applyPendingUpdate() error {
	updateScriptPath := filepath.Join(l.appDir, "apply_update.bat")

	if _, err := os.Stat(updateScriptPath); os.IsNotExist(err) {
		// Nenhuma atualização pendente
		return nil
	}

	fmt.Printf("📦 Aplicando atualização pendente...\n")
	logs.LogLauncher("📦 Aplicando atualização pendente...")

	cmd := exec.Command("cmd", "/C", updateScriptPath)
	cmd.Dir = l.appDir

	// Redirecionar output para mostrar progresso
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logs.LogError(logs.CategoryLauncher, "Falha na execução do script de atualização: %v", err)
		return fmt.Errorf("falha na execução do script de atualização: %w", err)
	}

	fmt.Printf("✅ Atualização aplicada com sucesso!\n")
	logs.LogLauncher("✅ Atualização aplicada com sucesso!")
	time.Sleep(1 * time.Second)

	return nil
}

func (l *Launcher) startMainApp() error {
	fmt.Printf("🚀 Iniciando %s...\n", appName)
	logs.LogLauncher("🚀 Iniciando %s...", appName)

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Método específico para Windows
		cmd = exec.Command(l.appPath)
		cmd.Dir = l.appDir

		// Configurar para criar processo completamente independente
		cmd.SysProcAttr = &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}

		// Desconectar completamente do launcher
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil

		logs.LogLauncher("🔧 Configurações Windows: processo independente, sem redirecionamento")
	} else {
		// Método genérico para outros sistemas
		cmd = exec.Command(l.appPath)
		cmd.Dir = l.appDir
		logs.LogLauncher("🔧 Configurações genéricas para OS: %s", runtime.GOOS)
	}

	// Iniciar processo
	if err := cmd.Start(); err != nil {
		logs.LogError(logs.CategoryLauncher, "Falha ao iniciar processo: %v", err)
		return fmt.Errorf("falha ao iniciar processo: %w", err)
	}

	fmt.Printf("✅ %s iniciado com sucesso (PID: %d)\n", appName, cmd.Process.Pid)
	logs.LogLauncher("✅ %s iniciado com sucesso (PID: %d)", appName, cmd.Process.Pid)

	// Dar um tempo para o app inicializar
	time.Sleep(2 * time.Second)

	// Verificar se o processo ainda está rodando
	if cmd.Process != nil {
		// Tentar verificar se processo está ativo (Windows específico)
		if runtime.GOOS == "windows" {
			// Liberar referência ao processo para deixá-lo independente
			cmd.Process.Release()
		}
		fmt.Printf("✅ Aplicativo está executando independentemente\n")
		logs.LogLauncher("✅ Aplicativo executando independentemente, launcher pode ser fechado")
	}

	return nil
}

func (l *Launcher) run() error {
	fmt.Printf("\n🚀 ===============================================\n")
	fmt.Printf("🚀 %s Launcher %s\n", appName, launcherVersion)
	fmt.Printf("🚀 ===============================================\n\n")

	logs.LogLauncher("🚀 %s Launcher %s iniciado", appName, launcherVersion)

	// Etapa 1: Verificar app principal
	fmt.Printf("🔍 [1/3] Verificando aplicativo principal...\n")
	logs.LogLauncher("🔍 [1/3] Verificando aplicativo principal...")
	if err := l.checkMainApp(); err != nil {
		logs.LogError(logs.CategoryLauncher, "❌ %v", err)
		return fmt.Errorf("❌ %w", err)
	}
	fmt.Printf("✅ Aplicativo principal encontrado\n\n")
	logs.LogLauncher("✅ Aplicativo principal encontrado: %s", l.appPath)

	// Etapa 2: Verificar e aplicar atualizações
	fmt.Printf("🔄 [2/3] Verificando atualizações...\n")
	logs.LogLauncher("🔄 [2/3] Verificando atualizações...")

	// 2.1: Aplicar atualizações pendentes primeiro
	if err := l.applyPendingUpdate(); err != nil {
		fmt.Printf("⚠️ Erro ao aplicar atualização pendente: %v\n", err)
		logs.LogError(logs.CategoryLauncher, "⚠️ Erro ao aplicar atualização pendente: %v", err)
	}

	// 2.2: Verificar por novas atualizações online
	if err := l.checkForUpdates(); err != nil {
		fmt.Printf("⚠️ Erro na verificação de atualizações: %v\n", err)
		fmt.Printf("⚠️ Continuando com versão atual...\n\n")
		logs.LogError(logs.CategoryLauncher, "⚠️ Erro na verificação de atualizações: %v", err)
	} else {
		fmt.Printf("✅ Verificação de atualizações concluída\n\n")
		logs.LogLauncher("✅ Verificação de atualizações concluída")
	}

	// Etapa 3: Iniciar app principal
	fmt.Printf("🚀 [3/3] Iniciando aplicativo principal...\n")
	logs.LogLauncher("🚀 [3/3] Iniciando aplicativo principal...")
	if err := l.startMainApp(); err != nil {
		logs.LogError(logs.CategoryLauncher, "❌ %v", err)
		return fmt.Errorf("❌ %w", err)
	}

	fmt.Printf("\n🎉 ===============================================\n")
	fmt.Printf("🎉 Launcher concluído com sucesso!\n")
	fmt.Printf("🎉 %s está rodando independentemente\n", appName)
	fmt.Printf("🎉 ===============================================\n\n")

	logs.LogLauncher("🎉 Launcher concluído com sucesso - %s executando independentemente", appName)

	return nil
}

func main() {
	// Inicializar sistema de logs antes de qualquer operação
	if err := logs.Init(); err != nil {
		fmt.Printf("⚠️ Erro ao inicializar logs: %v\n", err)
		// Continuar sem logs se necessário
	} else {
		logs.LogLauncher("📋 Sistema de logs do launcher inicializado")
	}

	launcher, err := NewLauncher()
	if err != nil {
		fmt.Printf("❌ Erro ao inicializar launcher: %v\n", err)
		logs.LogError(logs.CategoryLauncher, "❌ Erro ao inicializar launcher: %v", err)
		fmt.Printf("\nPressione Enter para sair...")
		fmt.Scanln()
		os.Exit(1)
	}

	if err := launcher.run(); err != nil {
		fmt.Printf("\n%v\n", err)
		fmt.Printf("💡 Tente executar %s diretamente se o problema persistir\n", appExecutable)
		logs.LogError(logs.CategoryLauncher, "%v", err)
		logs.LogLauncher("💡 Sugestão: executar %s diretamente se problema persistir", appExecutable)
		fmt.Printf("\nPressione Enter para sair...")
		fmt.Scanln()
		os.Exit(1)
	}

	// Aguardar um pouco antes de fechar
	fmt.Printf("🔄 Launcher será fechado em 3 segundos...\n")
	logs.LogLauncher("🔄 Launcher encerrando em 3 segundos...")
	time.Sleep(3 * time.Second)
	logs.LogLauncher("👋 Launcher finalizado com sucesso")
}
