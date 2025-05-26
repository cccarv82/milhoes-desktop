package data

import (
	"encoding/json"
	"fmt"
	"lottery-optimizer-gui/internal/lottery"
	"os"
	"path/filepath"
	"time"
)

// CacheEntry representa uma entrada do cache
type CacheEntry struct {
	LotteryType lottery.LotteryType `json:"lotteryType"`
	Draws       []lottery.Draw      `json:"draws"`
	CachedAt    time.Time           `json:"cachedAt"`
	Count       int                 `json:"count"`
}

// CacheManager gerencia o cache de dados de loteria
type CacheManager struct {
	cacheDir string
}

// NewCacheManager cria um novo gerenciador de cache
func NewCacheManager() *CacheManager {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".lottery-optimizer", "cache")

	// Criar diretório se não existir
	os.MkdirAll(cacheDir, 0755)

	return &CacheManager{
		cacheDir: cacheDir,
	}
}

// SaveToCache salva dados no cache
func (cm *CacheManager) SaveToCache(ltype lottery.LotteryType, draws []lottery.Draw, count int) error {
	entry := CacheEntry{
		LotteryType: ltype,
		Draws:       draws,
		CachedAt:    time.Now(),
		Count:       count,
	}

	filename := filepath.Join(cm.cacheDir, fmt.Sprintf("%s.json", string(ltype)))

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar cache: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadFromCache carrega dados do cache se válidos (menos de 1 mês)
func (cm *CacheManager) LoadFromCache(ltype lottery.LotteryType, count int) ([]lottery.Draw, bool) {
	filename := filepath.Join(cm.cacheDir, fmt.Sprintf("%s.json", string(ltype)))

	// Verificar se arquivo existe
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, false
	}

	// Ler arquivo
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, false
	}

	// Decodificar entrada
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	// Verificar se cache é válido (menos de 1 mês)
	if time.Since(entry.CachedAt) > 30*24*time.Hour {
		return nil, false
	}

	// Verificar se temos dados suficientes
	if len(entry.Draws) < count {
		// Se não temos dados suficientes, retornar o que temos
		return entry.Draws, len(entry.Draws) > 0
	}

	// Retornar quantidade solicitada
	return entry.Draws[:count], true
}

// IsCacheValid verifica se o cache é válido para um tipo de loteria
func (cm *CacheManager) IsCacheValid(ltype lottery.LotteryType) bool {
	filename := filepath.Join(cm.cacheDir, fmt.Sprintf("%s.json", string(ltype)))

	// Verificar se arquivo existe
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	// Ler apenas timestamp
	data, err := os.ReadFile(filename)
	if err != nil {
		return false
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return false
	}

	// Verificar se menos de 1 mês
	return time.Since(entry.CachedAt) <= 30*24*time.Hour
}

// CleanOldCache remove caches antigos (mais de 1 mês)
func (cm *CacheManager) CleanOldCache() error {
	files, err := os.ReadDir(cm.cacheDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		fullPath := filepath.Join(cm.cacheDir, file.Name())

		// Ler e verificar idade
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		var entry CacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		// Remover se mais de 1 mês
		if time.Since(entry.CachedAt) > 30*24*time.Hour {
			os.Remove(fullPath)
		}
	}

	return nil
}

// GetCacheInfo retorna informações sobre o cache
func (cm *CacheManager) GetCacheInfo(ltype lottery.LotteryType) (time.Time, int, bool) {
	filename := filepath.Join(cm.cacheDir, fmt.Sprintf("%s.json", string(ltype)))

	data, err := os.ReadFile(filename)
	if err != nil {
		return time.Time{}, 0, false
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return time.Time{}, 0, false
	}

	return entry.CachedAt, len(entry.Draws), true
}
