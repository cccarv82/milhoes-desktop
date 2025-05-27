package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	appExecutable = "milhoes.exe"
	appName       = "Lottery Optimizer"
	launcherVersion = "v1.0.0"
)

func main() {
	fmt.Printf("🚀 %s Launcher %s\n", appName, launcherVersion)
	fmt.Printf("🔍 Verificando atualizações pendentes...\n")
	
	// Obter diretório do launcher
	launcherPath, err := os.Executable()
	if err != nil {
		fmt.Printf("❌ Erro ao obter caminho do launcher: %v\n", err)
		os.Exit(1)
	}
	
	appDir := filepath.Dir(launcherPath)
	appPath := filepath.Join(appDir, appExecutable)
	updateScriptPath := filepath.Join(appDir, "apply_update.bat")
	
	// Verificar se app principal existe
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		fmt.Printf("❌ App principal não encontrado: %s\n", appPath)
		fmt.Printf("💡 Certifique-se que %s está no mesmo diretório do launcher\n", appExecutable)
		fmt.Printf("\nPressione Enter para sair...")
		fmt.Scanln()
		os.Exit(1)
	}
	
	// Verificar e aplicar atualizações pendentes
	if _, err := os.Stat(updateScriptPath); err == nil {
		fmt.Printf("📦 Atualização pendente encontrada! Aplicando...\n")
		
		if err := applyPendingUpdate(updateScriptPath, appDir); err != nil {
			fmt.Printf("❌ Erro ao aplicar atualização: %v\n", err)
			fmt.Printf("⚠️  Continuando com versão atual...\n")
		} else {
			fmt.Printf("✅ Atualização aplicada com sucesso!\n")
		}
	} else {
		fmt.Printf("✅ Nenhuma atualização pendente\n")
	}
	
	// Iniciar app principal
	fmt.Printf("🚀 Iniciando %s...\n", appName)
	
	if err := startMainApp(appPath); err != nil {
		fmt.Printf("❌ Erro ao iniciar app principal: %v\n", err)
		fmt.Printf("\nPressione Enter para sair...")
		fmt.Scanln()
		os.Exit(1)
	}
	
	fmt.Printf("👋 Launcher finalizado\n")
}

// applyPendingUpdate executa script de atualização pendente
func applyPendingUpdate(scriptPath, workingDir string) error {
	fmt.Printf("🔧 Executando script de atualização...\n")
	
	// Executar script de atualização
	cmd := exec.Command("cmd", "/C", scriptPath)
	cmd.Dir = workingDir
	
	// Redirecionar output para ver progresso
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Executar e aguardar conclusão
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falha na execução do script: %w", err)
	}
	
	// Aguardar um pouco para garantir que arquivos foram escritos
	time.Sleep(500 * time.Millisecond)
	
	return nil
}

// startMainApp inicia o aplicativo principal
func startMainApp(appPath string) error {
	// Verificar se app foi atualizado
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return fmt.Errorf("app principal não encontrado após atualização: %s", appPath)
	}
	
	// Preparar comando para executar app principal
	cmd := exec.Command(appPath)
	cmd.Dir = filepath.Dir(appPath)
	
	// No Windows, usar syscall para criar processo independente
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		// Configurar para criar processo independente
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		}
	}
	
	// Iniciar app principal como processo independente
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("falha ao iniciar processo: %w", err)
	}
	
	fmt.Printf("✅ %s iniciado (PID: %d)\n", appExecutable, cmd.Process.Pid)
	
	// Aguardar um pouco para garantir que app iniciou corretamente
	time.Sleep(1 * time.Second)
	
	// Verificar se processo ainda está rodando
	if cmd.Process != nil {
		// No Windows, não precisamos aguardar - o processo é independente
		fmt.Printf("✅ App principal rodando independentemente\n")
	}
	
	return nil
} 