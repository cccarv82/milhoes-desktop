package updater

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

// UpdateInfo informações sobre atualização disponível
type UpdateInfo struct {
	Available    bool   `json:"available"`
	Version      string `json:"version"`
	DownloadURL  string `json:"downloadUrl"`
	ReleaseNotes string `json:"releaseNotes"`
	Size         int64  `json:"size"`
	PublishedAt  string `json:"publishedAt"`
	Message      string `json:"message"`
	ReleaseURL   string `json:"releaseUrl"`
}

// GitHubRelease estrutura da resposta da API do GitHub
type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Prerelease  bool   `json:"prerelease"`
	Assets      []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
		Size        int64  `json:"size"`
	} `json:"assets"`
	HTMLURL string `json:"html_url"`
}

// Updater gerenciador de atualizações
type Updater struct {
	currentVersion string
	githubRepo     string
	client         *http.Client
}

// NewUpdater cria novo updater
func NewUpdater(currentVersion, githubRepo string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
		githubRepo:     githubRepo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CheckForUpdates verifica se há atualizações disponíveis
func (u *Updater) CheckForUpdates(ctx context.Context) (*UpdateInfo, error) {
	log.Printf("🔄 Iniciando verificação de atualizações...")
	log.Printf("📂 Repositório: %s", u.githubRepo)
	log.Printf("📱 Versão atual: %s", u.currentVersion)

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.githubRepo)
	log.Printf("🌐 URL da API: %s", url)

	// Criar requisição com timeout
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("❌ Erro ao criar requisição: %v", err)
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	// Adicionar headers
	req.Header.Set("User-Agent", fmt.Sprintf("Milhoes-Desktop/%s", u.currentVersion))
	req.Header.Set("Accept", "application/vnd.github+json")

	// Fazer requisição
	log.Printf("📡 Enviando requisição para GitHub API...")
	resp, err := u.client.Do(req)
	if err != nil {
		// Verificar se é erro de timeout
		if ctx.Err() == context.DeadlineExceeded {
			log.Printf("⏰ Timeout na requisição")
			return nil, fmt.Errorf("timeout na verificação de atualizações")
		}
		log.Printf("❌ Erro na requisição HTTP: %v", err)
		return nil, fmt.Errorf("erro na conexão com GitHub: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("📊 Status da resposta: %d", resp.StatusCode)

	// Ler corpo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ Erro ao ler resposta: %v", err)
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	// Tratar diferentes códigos de status
	switch resp.StatusCode {
	case 200:
		log.Printf("✅ Resposta recebida com sucesso")
		// Continuar com processamento normal
	case 404:
		log.Printf("🔍 Repositório retornou 404")
		// Verificar se é repositório sem releases ou repositório privado
		if strings.Contains(u.githubRepo, "milhoes-releases") {
			log.Printf("📦 Repositório de releases ainda não possui releases publicadas")
			return &UpdateInfo{
				Available: false,
				Message:   "Repositório de releases configurado, mas ainda não possui releases. A verificação funcionará após a primeira release ser publicada.",
			}, nil
		} else {
			log.Printf("🔒 Repositório privado ou não encontrado")
			return &UpdateInfo{
				Available: false,
				Message:   "Auto-updates não disponível para repositórios privados. Verifique manualmente em: https://github.com/" + u.githubRepo + "/releases",
			}, nil
		}
	case 403:
		log.Printf("🚫 Rate limit do GitHub (403)")
		return nil, fmt.Errorf("rate limit do GitHub atingido. Tente novamente em alguns minutos")
	default:
		log.Printf("❌ Status inesperado: %d", resp.StatusCode)
		log.Printf("📄 Corpo da resposta: %s", string(body))
		return nil, fmt.Errorf("GitHub API retornou status %d", resp.StatusCode)
	}

	// Parse da resposta JSON
	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		log.Printf("❌ Erro ao fazer parse do JSON: %v", err)
		log.Printf("📄 JSON recebido: %s", string(body))
		return nil, fmt.Errorf("erro ao processar resposta do GitHub: %w", err)
	}

	log.Printf("🏷️  Última versão disponível: %s", release.TagName)
	log.Printf("📅 Data de publicação: %s", release.PublishedAt)

	// Comparar versões
	currentVer := strings.TrimPrefix(u.currentVersion, "v")
	latestVer := strings.TrimPrefix(release.TagName, "v")

	log.Printf("🔄 Comparando versões:")
	log.Printf("   📱 Versão atual original: '%s'", u.currentVersion)
	log.Printf("   🏷️ Tag do GitHub original: '%s'", release.TagName)
	log.Printf("   📱 Versão atual limpa: '%s'", currentVer)
	log.Printf("   🏷️ Versão GitHub limpa: '%s'", latestVer)

	// Verificar se há atualização disponível
	isNewer, err := u.isVersionNewer(currentVer, latestVer)
	if err != nil {
		log.Printf("⚠️  Erro ao comparar versões: %v", err)
		log.Printf("   🔧 Tentando comparação simples de strings...")
		// Fallback: comparação simples
		isNewer = latestVer > currentVer
		log.Printf("   📊 Resultado da comparação simples: %t ('%s' > '%s')", isNewer, latestVer, currentVer)
	} else {
		log.Printf("✅ Comparação semver bem-sucedida: isNewer = %t", isNewer)
	}

	if !isNewer {
		log.Printf("✅ Aplicativo está atualizado")
		return &UpdateInfo{
			Available:   false,
			Version:     latestVer,
			DownloadURL: "",
			Message:     "Você já está usando a versão mais recente",
		}, nil
	}

	// Buscar asset para download
	var downloadURL string
	var assetSize int64
	for _, asset := range release.Assets {
		// Aceitar tanto ZIP quanto EXE para Windows
		if strings.Contains(asset.Name, "windows") ||
			strings.Contains(asset.Name, "Setup.exe") ||
			strings.Contains(asset.Name, ".exe") {
			downloadURL = asset.DownloadURL
			assetSize = asset.Size
			log.Printf("📦 Asset encontrado: %s (%d bytes)", asset.Name, asset.Size)
			break
		}
	}

	if downloadURL == "" {
		log.Printf("⚠️  Nenhum asset compatível encontrado")
		return &UpdateInfo{
			Available:   true,
			Version:     latestVer,
			DownloadURL: "",
			Message:     "Nova versão disponível, mas sem instalador. Baixe manualmente em: " + release.HTMLURL,
		}, nil
	}

	log.Printf("🎉 Nova versão disponível: %s -> %s", currentVer, latestVer)
	log.Printf("📥 URL de download: %s", downloadURL)

	return &UpdateInfo{
		Available:    true,
		Version:      latestVer,
		DownloadURL:  downloadURL,
		ReleaseURL:   release.HTMLURL,
		Size:         assetSize,
		PublishedAt:  release.PublishedAt,
		ReleaseNotes: release.Body,
		Message:      fmt.Sprintf("Nova versão %s disponível!", latestVer),
	}, nil
}

// getAssetName retorna o nome do asset baseado na plataforma
func (u *Updater) getAssetName() string {
	switch runtime.GOOS {
	case "windows":
		return "MilhoesSetup.exe" // Instalador do Windows
	case "darwin":
		return "milhoes-darwin"
	case "linux":
		return "milhoes-linux"
	default:
		return "milhoes"
	}
}

// DownloadUpdate baixa a atualização
func (u *Updater) DownloadUpdate(ctx context.Context, updateInfo *UpdateInfo, progressCallback func(downloaded, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, "GET", updateInfo.DownloadURL, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar requisição de download: %w", err)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao baixar atualização: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download falhou com status %d", resp.StatusCode)
	}

	// Criar arquivo temporário
	tempDir := os.TempDir()
	fileName := filepath.Base(updateInfo.DownloadURL)
	tempFile := filepath.Join(tempDir, fileName)

	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo temporário: %w", err)
	}
	defer file.Close()

	// Download com progresso
	var downloaded int64
	total := resp.ContentLength

	buffer := make([]byte, 32*1024) // 32KB buffer
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := resp.Body.Read(buffer)
			if n > 0 {
				if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
					return fmt.Errorf("erro ao escrever arquivo: %w", writeErr)
				}
				downloaded += int64(n)

				if progressCallback != nil {
					progressCallback(downloaded, total)
				}
			}

			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("erro durante download: %w", err)
			}
		}
	}
}

// InstallUpdate instala a atualização baixada (preparação, não forçar fechamento)
func (u *Updater) InstallUpdate(updateInfo *UpdateInfo) error {
	tempDir := os.TempDir()
	fileName := filepath.Base(updateInfo.DownloadURL)
	installerPath := filepath.Join(tempDir, fileName)

	// Verificar se arquivo existe
	if _, err := os.Stat(installerPath); os.IsNotExist(err) {
		return fmt.Errorf("arquivo de instalação não encontrado: %s", installerPath)
	}

	switch runtime.GOOS {
	case "windows":
		return u.prepareWindowsInstall(installerPath)
	case "darwin":
		return u.installMacOS(installerPath)
	case "linux":
		return u.installLinux(installerPath)
	default:
		return fmt.Errorf("plataforma não suportada: %s", runtime.GOOS)
	}
}

// prepareWindowsInstall prepara a instalação no Windows (sem forçar fechamento)
func (u *Updater) prepareWindowsInstall(installerPath string) error {
	log.Printf("🔧 Preparando atualização no Windows: %s", installerPath)

	// Verificar se é ZIP ou EXE
	if strings.HasSuffix(strings.ToLower(installerPath), ".zip") {
		log.Printf("📦 Arquivo ZIP detectado, preparando extração...")
		return u.prepareZipInstall(installerPath)
	} else if strings.HasSuffix(strings.ToLower(installerPath), ".exe") {
		log.Printf("🚀 Executável detectado, instalador pronto para execução...")
		return u.prepareExeInstall(installerPath)
	} else {
		return fmt.Errorf("formato de arquivo não suportado: %s", installerPath)
	}
}

// prepareZipInstall prepara extração ZIP (sem executar ainda)
func (u *Updater) prepareZipInstall(zipPath string) error {
	log.Printf("📦 Preparando extração de arquivo ZIP: %s", zipPath)

	// Verificar se ZIP é válido
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("erro ao verificar ZIP: %w", err)
	}
	reader.Close()

	log.Printf("✅ Arquivo ZIP válido e pronto para extração quando usuário reiniciar")
	return nil
}

// prepareExeInstall prepara executável (verificação apenas)
func (u *Updater) prepareExeInstall(exePath string) error {
	log.Printf("🚀 Verificando instalador executável: %s", exePath)

	// Verificar se arquivo é executável válido
	if stat, err := os.Stat(exePath); err != nil {
		return fmt.Errorf("erro ao verificar executável: %w", err)
	} else {
		log.Printf("✅ Instalador executável válido (%d bytes) e pronto para execução quando usuário reiniciar", stat.Size())
	}

	return nil
}

// installWindows instala no Windows usando o instalador
func (u *Updater) installWindows(installerPath string) error {
	log.Printf("🔧 Iniciando instalação no Windows: %s", installerPath)

	// Verificar se é ZIP ou EXE
	if strings.HasSuffix(strings.ToLower(installerPath), ".zip") {
		log.Printf("📦 Arquivo ZIP detectado, extraindo...")
		return u.installFromZip(installerPath)
	} else if strings.HasSuffix(strings.ToLower(installerPath), ".exe") {
		log.Printf("🚀 Executável detectado, executando instalador...")
		return u.installFromExe(installerPath)
	} else {
		return fmt.Errorf("formato de arquivo não suportado: %s", installerPath)
	}
}

// installFromZip extrai ZIP e substitui executável atual
func (u *Updater) installFromZip(zipPath string) error {
	log.Printf("📦 Extraindo arquivo ZIP: %s", zipPath)

	// Abrir arquivo ZIP
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("erro ao abrir ZIP: %w", err)
	}
	defer reader.Close()

	// Obter caminho do executável atual
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("erro ao obter caminho do executável: %w", err)
	}

	tempDir := filepath.Join(os.TempDir(), "milhoes_update")

	// Criar diretório temporário
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório temporário: %w", err)
	}

	// Extrair arquivos
	for _, file := range reader.File {
		log.Printf("📄 Extraindo: %s", file.Name)

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("erro ao abrir arquivo no ZIP: %w", err)
		}

		destPath := filepath.Join(tempDir, file.Name)

		// Criar diretórios se necessário
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			rc.Close()
			return fmt.Errorf("erro ao criar diretório: %w", err)
		}

		// Extrair arquivo
		destFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("erro ao criar arquivo: %w", err)
		}

		_, err = io.Copy(destFile, rc)
		destFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("erro ao extrair arquivo: %w", err)
		}
	}

	// Encontrar novo executável
	newExePath := filepath.Join(tempDir, "milhoes.exe")
	if _, err := os.Stat(newExePath); os.IsNotExist(err) {
		return fmt.Errorf("executável não encontrado no ZIP: %s", newExePath)
	}

	log.Printf("✅ Extração concluída. Preparando para substituir executável...")

	// Criar script de atualização
	scriptPath := filepath.Join(os.TempDir(), "update_milhoes.bat")
	scriptContent := fmt.Sprintf(`@echo off
echo Aguardando fechamento do aplicativo...
timeout /t 2 /nobreak > nul
echo Fazendo backup...
move "%s" "%s.bak" 2>nul
echo Copiando nova versao...
copy "%s" "%s"
if errorlevel 1 (
    echo Erro na atualizacao, restaurando backup...
    move "%s.bak" "%s"
    echo Falha na atualizacao
) else (
    echo Limpando backup...
    del "%s.bak" 2>nul
    echo Atualizacao concluida com sucesso!
    echo Reiniciando aplicativo...
    start "" "%s"
)
echo Limpando arquivos temporarios...
rmdir /s /q "%s" 2>nul
del "%%~f0"
`, currentExe, currentExe, newExePath, currentExe, currentExe, currentExe, currentExe, currentExe, tempDir)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("erro ao criar script de atualização: %w", err)
	}

	log.Printf("🚀 Executando script de atualização...")

	// Executar script em background
	cmd := exec.Command("cmd", "/C", scriptPath)
	cmd.Dir = os.TempDir()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erro ao executar script de atualização: %w", err)
	}

	// Sair da aplicação para permitir atualização
	log.Printf("👋 Encerrando aplicação para permitir atualização...")
	go func() {
		time.Sleep(1 * time.Second)
		os.Exit(0)
	}()

	return nil
}

// installFromExe executa instalador EXE
func (u *Updater) installFromExe(exePath string) error {
	log.Printf("🚀 Executando instalador: %s", exePath)

	// Executar instalador silencioso
	cmd := exec.Command(exePath, "/SILENT", "/CLOSEAPPLICATIONS", "/RESTARTAPPLICATIONS")

	// Executar em background e fechar aplicação atual
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erro ao executar instalador: %w", err)
	}

	// Sair da aplicação para permitir atualização
	log.Printf("👋 Encerrando aplicação para instalação...")
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	return nil
}

// installMacOS instala no macOS
func (u *Updater) installMacOS(installerPath string) error {
	// Implementar instalação para macOS
	return fmt.Errorf("instalação macOS não implementada ainda")
}

// installLinux instala no Linux
func (u *Updater) installLinux(installerPath string) error {
	// Implementar instalação para Linux
	return fmt.Errorf("instalação Linux não implementada ainda")
}

// ScheduleUpdateCheck agenda verificação automática
func (u *Updater) ScheduleUpdateCheck(interval time.Duration, callback func(*UpdateInfo, error)) {
	ticker := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				updateInfo, err := u.CheckForUpdates(ctx)
				cancel()

				if callback != nil {
					callback(updateInfo, err)
				}
			}
		}
	}()
}

// GetCurrentVersion retorna versão atual
func (u *Updater) GetCurrentVersion() string {
	return u.currentVersion
}

// isVersionNewer compara duas versões
func (u *Updater) isVersionNewer(currentVer, latestVer string) (bool, error) {
	log.Printf("🔬 isVersionNewer: Comparando '%s' vs '%s'", currentVer, latestVer)

	currentSemver, err := semver.NewVersion(currentVer)
	if err != nil {
		log.Printf("❌ Erro ao parsear versão atual '%s': %v", currentVer, err)
		return false, fmt.Errorf("versão atual inválida: %w", err)
	}
	log.Printf("✅ Versão atual parseada: %s", currentSemver.String())

	latestSemver, err := semver.NewVersion(latestVer)
	if err != nil {
		log.Printf("❌ Erro ao parsear versão do GitHub '%s': %v", latestVer, err)
		return false, fmt.Errorf("versão da release inválida: %w", err)
	}
	log.Printf("✅ Versão do GitHub parseada: %s", latestSemver.String())

	result := latestSemver.GreaterThan(currentSemver)
	log.Printf("🔍 Resultado da comparação semver: %s.GreaterThan(%s) = %t",
		latestSemver.String(), currentSemver.String(), result)

	return result, nil
}

// ExecuteInstall executa a instalação real (chamado quando usuário escolhe reiniciar)
func (u *Updater) ExecuteInstall(updateInfo *UpdateInfo) error {
	tempDir := os.TempDir()
	fileName := filepath.Base(updateInfo.DownloadURL)
	installerPath := filepath.Join(tempDir, fileName)

	// Verificar se arquivo ainda existe
	if _, err := os.Stat(installerPath); os.IsNotExist(err) {
		return fmt.Errorf("arquivo de instalação não encontrado: %s", installerPath)
	}

	switch runtime.GOOS {
	case "windows":
		return u.installWindows(installerPath)
	case "darwin":
		return u.installMacOS(installerPath)
	case "linux":
		return u.installLinux(installerPath)
	default:
		return fmt.Errorf("plataforma não suportada: %s", runtime.GOOS)
	}
}

// InstallSilently instala a atualização de forma silenciosa em background
func (u *Updater) InstallSilently(updateInfo *UpdateInfo) error {
	tempDir := os.TempDir()
	fileName := filepath.Base(updateInfo.DownloadURL)
	installerPath := filepath.Join(tempDir, fileName)

	// Verificar se arquivo existe
	if _, err := os.Stat(installerPath); os.IsNotExist(err) {
		return fmt.Errorf("arquivo de instalação não encontrado: %s", installerPath)
	}

	switch runtime.GOOS {
	case "windows":
		return u.installWindowsSilently(installerPath)
	case "darwin":
		return u.installMacOSSilently(installerPath)
	case "linux":
		return u.installLinuxSilently(installerPath)
	default:
		return fmt.Errorf("plataforma não suportada: %s", runtime.GOOS)
	}
}

// installWindowsSilently instala no Windows de forma silenciosa
func (u *Updater) installWindowsSilently(installerPath string) error {
	log.Printf("🔧 Instalação silenciosa no Windows: %s", installerPath)

	// Verificar se é ZIP ou EXE
	if strings.HasSuffix(strings.ToLower(installerPath), ".zip") {
		log.Printf("📦 Arquivo ZIP detectado, extraindo silenciosamente...")
		return u.installFromZipSilently(installerPath)
	} else if strings.HasSuffix(strings.ToLower(installerPath), ".exe") {
		log.Printf("🚀 Executável detectado, preparando instalação silenciosa...")
		return u.installFromExeSilently(installerPath)
	} else {
		return fmt.Errorf("formato de arquivo não suportado: %s", installerPath)
	}
}

// installFromZipSilently extrai ZIP e prepara substituição do executável para próxima execução
func (u *Updater) installFromZipSilently(zipPath string) error {
	log.Printf("📦 Extração silenciosa de arquivo ZIP: %s", zipPath)

	// Abrir arquivo ZIP
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("erro ao abrir ZIP: %w", err)
	}
	defer reader.Close()

	// Obter caminho do executável atual
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("erro ao obter caminho do executável: %w", err)
	}

	currentDir := filepath.Dir(currentExe)
	updateDir := filepath.Join(currentDir, ".update")

	// Criar diretório de atualização
	os.RemoveAll(updateDir)
	if err := os.MkdirAll(updateDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório de atualização: %w", err)
	}

	// Extrair arquivos
	for _, file := range reader.File {
		log.Printf("📄 Extraindo silenciosamente: %s", file.Name)

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("erro ao abrir arquivo no ZIP: %w", err)
		}

		destPath := filepath.Join(updateDir, file.Name)

		// Criar diretórios se necessário
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			rc.Close()
			return fmt.Errorf("erro ao criar diretório: %w", err)
		}

		// Extrair arquivo
		destFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("erro ao criar arquivo: %w", err)
		}

		_, err = io.Copy(destFile, rc)
		destFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("erro ao extrair arquivo: %w", err)
		}
	}

	// Encontrar novo executável
	newExePath := filepath.Join(updateDir, "milhoes.exe")
	if _, err := os.Stat(newExePath); os.IsNotExist(err) {
		return fmt.Errorf("executável não encontrado no ZIP: %s", newExePath)
	}

	log.Printf("✅ Extração silenciosa concluída. Criando script de atualização...")

	// Criar script de atualização que será executado na próxima inicialização
	scriptPath := filepath.Join(currentDir, "apply_update.bat")
	scriptContent := fmt.Sprintf(`@echo off
REM Script de atualização silenciosa - executado na próxima inicialização
echo Aplicando atualização silenciosa...

REM Fazer backup da versão atual
if exist "%s" (
    echo Fazendo backup da versão atual...
    move "%s" "%s.bak" 2>nul
)

REM Copiar nova versão
if exist "%s" (
    echo Instalando nova versão...
    copy "%s" "%s" >nul 2>&1
    if errorlevel 1 (
        echo Erro na atualização, restaurando backup...
        if exist "%s.bak" (
            move "%s.bak" "%s" 2>nul
        )
        echo Falha na atualização silenciosa
        goto cleanup
    ) else (
        echo Atualização silenciosa concluída com sucesso!
        REM Limpar backup
        if exist "%s.bak" (
            del "%s.bak" 2>nul
        )
    )
) else (
    echo Arquivo de atualização não encontrado!
    if exist "%s.bak" (
        move "%s.bak" "%s" 2>nul
    )
)

:cleanup
echo Limpando arquivos temporários...
if exist "%s" (
    rmdir /s /q "%s" 2>nul
)
echo Removendo script de atualização...
del "%%~f0" 2>nul
`, currentExe, currentExe, currentExe, newExePath, newExePath, currentExe, currentExe, currentExe, currentExe, currentExe, currentExe, currentExe, currentExe, currentExe, updateDir, updateDir)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("erro ao criar script de atualização: %w", err)
	}

	log.Printf("✅ Instalação silenciosa preparada. Atualização será aplicada na próxima execução do app.")
	return nil
}

// installFromExeSilently prepara instalador EXE para execução silenciosa
func (u *Updater) installFromExeSilently(exePath string) error {
	log.Printf("🚀 Preparando instalação silenciosa via EXE: %s", exePath)

	// Obter caminho do executável atual
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("erro ao obter caminho do executável: %w", err)
	}

	currentDir := filepath.Dir(currentExe)

	// Criar script que será executado na próxima inicialização
	scriptPath := filepath.Join(currentDir, "apply_update.bat")
	scriptContent := fmt.Sprintf(`@echo off
REM Script de instalação silenciosa - executado na próxima inicialização
echo Executando instalação silenciosa...

REM Executar instalador silencioso
"%s" /SILENT /SUPPRESSMSGBOXES /NORESTART >nul 2>&1

if errorlevel 1 (
    echo Erro na instalação silenciosa
) else (
    echo Instalação silenciosa concluída com sucesso!
)

echo Removendo script de atualização...
del "%%~f0" 2>nul
`, exePath)

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("erro ao criar script de atualização: %w", err)
	}

	log.Printf("✅ Instalação silenciosa preparada. Atualização será aplicada na próxima execução do app.")
	return nil
}

// installMacOSSilently instala no macOS de forma silenciosa
func (u *Updater) installMacOSSilently(installerPath string) error {
	// Implementar instalação silenciosa para macOS
	return fmt.Errorf("instalação silenciosa macOS não implementada ainda")
}

// installLinuxSilently instala no Linux de forma silenciosa
func (u *Updater) installLinuxSilently(installerPath string) error {
	// Implementar instalação silenciosa para Linux
	return fmt.Errorf("instalação silenciosa Linux não implementada ainda")
}

// CheckAndApplyPendingUpdate verifica e aplica atualizações pendentes (chamado na inicialização)
func (u *Updater) CheckAndApplyPendingUpdate() error {
	// Obter caminho do executável atual
	currentExe, err := os.Executable()
	if err != nil {
		return err
	}

	currentDir := filepath.Dir(currentExe)
	scriptPath := filepath.Join(currentDir, "apply_update.bat")

	// Verificar se existe script de atualização pendente
	if _, err := os.Stat(scriptPath); err == nil {
		log.Printf("🔄 Script de atualização pendente encontrado, aplicando...")

		// Executar script de atualização
		cmd := exec.Command("cmd", "/C", scriptPath)
		cmd.Dir = currentDir

		// Executar e aguardar conclusão
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("❌ Erro ao executar script de atualização: %v", err)
			log.Printf("📄 Output: %s", string(output))
			return fmt.Errorf("erro ao aplicar atualização pendente: %w", err)
		}

		log.Printf("✅ Atualização pendente aplicada com sucesso!")
		log.Printf("📄 Output: %s", string(output))
		return nil
	}

	// Sem atualizações pendentes
	return nil
}

// InstallImmediate aplica a atualização imediatamente sem encerrar o processo atual
func (u *Updater) InstallImmediate(updateInfo *UpdateInfo) error {
	tempDir := os.TempDir()
	fileName := filepath.Base(updateInfo.DownloadURL)
	installerPath := filepath.Join(tempDir, fileName)

	// Verificar se arquivo existe
	if _, err := os.Stat(installerPath); os.IsNotExist(err) {
		return fmt.Errorf("arquivo de instalação não encontrado: %s", installerPath)
	}

	switch runtime.GOOS {
	case "windows":
		return u.installWindowsImmediate(installerPath)
	case "darwin":
		return u.installMacOSImmediate(installerPath)
	case "linux":
		return u.installLinuxImmediate(installerPath)
	default:
		return fmt.Errorf("plataforma não suportada: %s", runtime.GOOS)
	}
}

// installWindowsImmediate instala no Windows imediatamente sem encerrar processo
func (u *Updater) installWindowsImmediate(installerPath string) error {
	log.Printf("🔧 Instalação imediata no Windows: %s", installerPath)

	// Verificar se é ZIP ou EXE
	if strings.HasSuffix(strings.ToLower(installerPath), ".zip") {
		log.Printf("📦 Arquivo ZIP detectado, extraindo imediatamente...")
		return u.installFromZipImmediate(installerPath)
	} else if strings.HasSuffix(strings.ToLower(installerPath), ".exe") {
		log.Printf("🚀 Executável detectado, aplicando instalação imediata...")
		return u.installFromExeImmediate(installerPath)
	} else {
		return fmt.Errorf("formato de arquivo não suportado: %s", installerPath)
	}
}

// installFromZipImmediate extrai ZIP e substitui o executável principal sem afetar o launcher
func (u *Updater) installFromZipImmediate(zipPath string) error {
	log.Printf("📦 Extração imediata de arquivo ZIP: %s", zipPath)

	// Abrir arquivo ZIP
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("erro ao abrir ZIP: %w", err)
	}
	defer reader.Close()

	// Obter caminho do diretório do executável
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("erro ao obter caminho do executável: %w", err)
	}

	installDir := filepath.Dir(currentExe)
	tempDir := filepath.Join(os.TempDir(), "milhoes_update_immediate")

	// Criar diretório temporário
	os.RemoveAll(tempDir)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório temporário: %w", err)
	}

	// Extrair arquivos
	log.Printf("📄 Extraindo arquivos para: %s", tempDir)
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("erro ao abrir arquivo no ZIP: %w", err)
		}

		destPath := filepath.Join(tempDir, file.Name)

		// Criar diretórios se necessário
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			rc.Close()
			return fmt.Errorf("erro ao criar diretório: %w", err)
		}

		// Extrair arquivo
		destFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("erro ao criar arquivo: %w", err)
		}

		_, err = io.Copy(destFile, rc)
		destFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("erro ao extrair arquivo: %w", err)
		}
	}

	// Encontrar novo executável principal (milhoes.exe)
	newExePath := filepath.Join(tempDir, "milhoes.exe")
	if _, err := os.Stat(newExePath); os.IsNotExist(err) {
		return fmt.Errorf("executável principal não encontrado no ZIP: %s", newExePath)
	}

	// Caminho do executável principal atual
	currentMainExe := filepath.Join(installDir, "milhoes.exe")
	backupPath := currentMainExe + ".backup"

	log.Printf("🔄 Substituindo executável principal: %s", currentMainExe)

	// Fazer backup da versão atual (se existir)
	if _, err := os.Stat(currentMainExe); err == nil {
		log.Printf("💾 Fazendo backup: %s -> %s", currentMainExe, backupPath)
		if err := os.Rename(currentMainExe, backupPath); err != nil {
			return fmt.Errorf("erro ao fazer backup: %w", err)
		}
	}

	// Copiar nova versão
	log.Printf("📁 Copiando nova versão: %s -> %s", newExePath, currentMainExe)
	if err := copyFile(newExePath, currentMainExe); err != nil {
		// Restaurar backup em caso de erro
		if _, backupExists := os.Stat(backupPath); backupExists == nil {
			os.Rename(backupPath, currentMainExe)
		}
		return fmt.Errorf("erro ao copiar nova versão: %w", err)
	}

	// Limpar backup antigo
	if _, err := os.Stat(backupPath); err == nil {
		os.Remove(backupPath)
		log.Printf("🗑️ Backup removido: %s", backupPath)
	}

	// Limpar arquivos temporários
	os.RemoveAll(tempDir)
	log.Printf("🗑️ Arquivos temporários removidos: %s", tempDir)

	log.Printf("✅ Atualização imediata concluída com sucesso!")
	return nil
}

// installFromExeImmediate executa instalador EXE imediatamente (não implementado)
func (u *Updater) installFromExeImmediate(exePath string) error {
	log.Printf("⚠️ Instalação imediata via EXE não implementada: %s", exePath)
	return fmt.Errorf("instalação imediata via EXE não suportada")
}

// installMacOSImmediate - placeholder para macOS
func (u *Updater) installMacOSImmediate(installerPath string) error {
	return fmt.Errorf("instalação imediata no macOS não implementada")
}

// installLinuxImmediate - placeholder para Linux
func (u *Updater) installLinuxImmediate(installerPath string) error {
	return fmt.Errorf("instalação imediata no Linux não implementada")
}

// copyFile copia um arquivo de origem para destino
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
