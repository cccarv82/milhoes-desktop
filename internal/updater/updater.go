package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	// Buscar última release
	releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.githubRepo)

	req, err := http.NewRequestWithContext(ctx, "GET", releaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API retornou status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	// Ignorar prereleases por padrão
	if release.Prerelease {
		return &UpdateInfo{Available: false}, nil
	}

	// Comparar versões
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVer, err := semver.NewVersion(u.currentVersion)
	if err != nil {
		return nil, fmt.Errorf("versão atual inválida: %w", err)
	}

	latestVer, err := semver.NewVersion(latestVersion)
	if err != nil {
		return nil, fmt.Errorf("versão da release inválida: %w", err)
	}

	updateInfo := &UpdateInfo{
		Available:    latestVer.GreaterThan(currentVer),
		Version:      latestVersion,
		ReleaseNotes: release.Body,
		PublishedAt:  release.PublishedAt,
	}

	if updateInfo.Available {
		// Encontrar asset apropriado para a plataforma
		assetName := u.getAssetName()
		for _, asset := range release.Assets {
			if strings.Contains(asset.Name, assetName) {
				updateInfo.DownloadURL = asset.DownloadURL
				updateInfo.Size = asset.Size
				break
			}
		}

		if updateInfo.DownloadURL == "" {
			return nil, fmt.Errorf("asset não encontrado para plataforma %s", runtime.GOOS)
		}
	}

	return updateInfo, nil
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
	// Executar instalador silencioso
	cmd := exec.Command(installerPath, "/SILENT", "/CLOSEAPPLICATIONS", "/RESTARTAPPLICATIONS")

	// Executar em background e fechar aplicação atual
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erro ao executar instalador: %w", err)
	}

	// Sair da aplicação para permitir atualização
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
