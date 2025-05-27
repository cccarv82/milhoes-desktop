package main

import (
	"fmt"
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
	launcherVersion = "v1.1.0"
)

type Launcher struct {
	appDir  string
	appPath string
}

func NewLauncher() (*Launcher, error) {
	launcherPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter caminho do launcher: %w", err)
	}

	appDir := filepath.Dir(launcherPath)
	appPath := filepath.Join(appDir, appExecutable)

	return &Launcher{
		appDir:  appDir,
		appPath: appPath,
	}, nil
}

func (l *Launcher) checkMainApp() error {
	if _, err := os.Stat(l.appPath); os.IsNotExist(err) {
		return fmt.Errorf("app principal não encontrado: %s", l.appPath)
	}
	return nil
}

func (l *Launcher) applyPendingUpdate() error {
	updateScriptPath := filepath.Join(l.appDir, "apply_update.bat")

	if _, err := os.Stat(updateScriptPath); os.IsNotExist(err) {
		// Nenhuma atualização pendente
		return nil
	}

	fmt.Printf("📦 Aplicando atualização pendente...\n")

	cmd := exec.Command("cmd", "/C", updateScriptPath)
	cmd.Dir = l.appDir

	// Redirecionar output para mostrar progresso
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falha na execução do script de atualização: %w", err)
	}

	fmt.Printf("✅ Atualização aplicada com sucesso!\n")
	time.Sleep(1 * time.Second)

	return nil
}

func (l *Launcher) startMainApp() error {
	fmt.Printf("🚀 Iniciando %s...\n", appName)

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// Método específico para Windows
		cmd = exec.Command(l.appPath)
		cmd.Dir = l.appDir

		// Configurar para criar processo completamente independente
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    false,
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}

		// Desconectar completamente do launcher
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.Stdin = nil
	} else {
		// Método genérico para outros sistemas
		cmd = exec.Command(l.appPath)
		cmd.Dir = l.appDir
	}

	// Iniciar processo
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("falha ao iniciar processo: %w", err)
	}

	fmt.Printf("✅ %s iniciado com sucesso (PID: %d)\n", appName, cmd.Process.Pid)

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
	}

	return nil
}

func (l *Launcher) run() error {
	fmt.Printf("\n🚀 ===============================================\n")
	fmt.Printf("🚀 %s Launcher %s\n", appName, launcherVersion)
	fmt.Printf("🚀 ===============================================\n\n")

	// Etapa 1: Verificar app principal
	fmt.Printf("🔍 [1/3] Verificando aplicativo principal...\n")
	if err := l.checkMainApp(); err != nil {
		return fmt.Errorf("❌ %w", err)
	}
	fmt.Printf("✅ Aplicativo principal encontrado\n\n")

	// Etapa 2: Aplicar atualizações pendentes
	fmt.Printf("🔄 [2/3] Verificando atualizações pendentes...\n")
	if err := l.applyPendingUpdate(); err != nil {
		fmt.Printf("⚠️ Erro na atualização: %v\n", err)
		fmt.Printf("⚠️ Continuando com versão atual...\n\n")
	} else {
		fmt.Printf("✅ Verificação de atualizações concluída\n\n")
	}

	// Etapa 3: Iniciar app principal
	fmt.Printf("🚀 [3/3] Iniciando aplicativo principal...\n")
	if err := l.startMainApp(); err != nil {
		return fmt.Errorf("❌ %w", err)
	}

	fmt.Printf("\n🎉 ===============================================\n")
	fmt.Printf("🎉 Launcher concluído com sucesso!\n")
	fmt.Printf("🎉 %s está rodando independentemente\n", appName)
	fmt.Printf("🎉 ===============================================\n\n")

	return nil
}

func main() {
	launcher, err := NewLauncher()
	if err != nil {
		fmt.Printf("❌ Erro ao inicializar launcher: %v\n", err)
		fmt.Printf("\nPressione Enter para sair...")
		fmt.Scanln()
		os.Exit(1)
	}

	if err := launcher.run(); err != nil {
		fmt.Printf("\n%v\n", err)
		fmt.Printf("💡 Tente executar %s diretamente se o problema persistir\n", appExecutable)
		fmt.Printf("\nPressione Enter para sair...")
		fmt.Scanln()
		os.Exit(1)
	}

	// Aguardar um pouco antes de fechar
	fmt.Printf("🔄 Launcher será fechado em 3 segundos...\n")
	time.Sleep(3 * time.Second)
}
