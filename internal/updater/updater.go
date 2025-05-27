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
		Available:   true,
		Version:     latestVer,
		DownloadURL: downloadURL,
		ReleaseURL:  release.HTMLURL,
		Size:        assetSize,
		PublishedAt: release.PublishedAt,
		ReleaseNotes: release.Body,
		Message:     fmt.Sprintf("Nova versão %s disponível!", latestVer),
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

// InstallUpdate instala a atualização baixada
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
		return u.installWindows(installerPath)
	case "darwin":
		return u.installMacOS(installerPath)
	case "linux":
		return u.installLinux(installerPath)
	default:
		return fmt.Errorf("plataforma não suportada: %s", runtime.GOOS)
	}
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
