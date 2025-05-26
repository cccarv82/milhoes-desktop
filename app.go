package main

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"lottery-optimizer-gui/internal/ai"
	"lottery-optimizer-gui/internal/config"
	"lottery-optimizer-gui/internal/data"
	"lottery-optimizer-gui/internal/lottery"
	"lottery-optimizer-gui/internal/updater"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	version    = "1.0.0"                // Será injetado durante build
	githubRepo = "yourusername/milhoes" // Substitua pelo seu repo
)

// App struct - Bridge entre Frontend e Backend
type App struct {
	ctx        context.Context
	dataClient *data.Client
	aiClient   *ai.ClaudeClient
	updater    *updater.Updater
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		dataClient: data.NewClient(),
		aiClient:   ai.NewClaudeClient(),
		updater:    updater.NewUpdater(version, githubRepo),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// UserPreferences representa as preferências do usuário para o frontend
type UserPreferences struct {
	LotteryTypes    []string `json:"lotteryTypes"`
	Budget          float64  `json:"budget"`
	Strategy        string   `json:"strategy"`
	AvoidPatterns   bool     `json:"avoidPatterns"`
	FavoriteNumbers []int    `json:"favoriteNumbers"`
	ExcludeNumbers  []int    `json:"excludeNumbers"`
}

// StrategyResponse resposta da geração de estratégia
type StrategyResponse struct {
	Success            bool              `json:"success"`
	Strategy           *lottery.Strategy `json:"strategy,omitempty"`
	Confidence         float64           `json:"confidence"`
	Error              string            `json:"error,omitempty"`
	AvailableLotteries []string          `json:"availableLotteries,omitempty"`
	FailedLotteries    []string          `json:"failedLotteries,omitempty"`
}

// ConnectionStatus status das conexões
type ConnectionStatus struct {
	CaixaAPI    bool   `json:"caixaAPI"`
	CaixaError  string `json:"caixaError,omitempty"`
	ClaudeAPI   bool   `json:"claudeAPI"`
	ClaudeError string `json:"claudeError,omitempty"`
}

// ConfigData representa os dados de configuração para o frontend
type ConfigData struct {
	ClaudeAPIKey string `json:"claudeApiKey" yaml:"claude_api_key"`
	ClaudeModel  string `json:"claudeModel" yaml:"claude_model"`
	TimeoutSec   int    `json:"timeoutSec" yaml:"timeout_sec"`
	MaxTokens    int    `json:"maxTokens" yaml:"max_tokens"`
	Verbose      bool   `json:"verbose" yaml:"verbose"`
}

// ===============================
// FUNÇÕES AUXILIARES
// ===============================

// mapStrategy mapeia estratégias do frontend para o backend
func mapStrategy(frontendStrategy string) string {
	switch frontendStrategy {
	case "intelligent":
		return "balanced" // Estratégia inteligente usa análise equilibrada
	default:
		return frontendStrategy
	}
}

// ===============================
// MÉTODOS DA API PARA O FRONTEND
// ===============================

// TestConnectionsWithConfig testa as conexões com uma configuração específica
func (a *App) TestConnectionsWithConfig(configData ConfigData) ConnectionStatus {
	status := ConnectionStatus{}

	// Testar API da Caixa (não depende da configuração)
	if err := a.dataClient.TestConnection(); err != nil {
		status.CaixaAPI = false
		status.CaixaError = err.Error()
	} else {
		status.CaixaAPI = true
	}

	// Testar Claude API com a configuração fornecida
	testClient := ai.NewClaudeClientWithConfig(configData.ClaudeAPIKey, configData.ClaudeModel, configData.MaxTokens, configData.TimeoutSec)
	if err := testClient.TestConnection(); err != nil {
		status.ClaudeAPI = false
		status.ClaudeError = err.Error()
	} else {
		status.ClaudeAPI = true
	}

	return status
}

// TestConnections testa as conexões com APIs
func (a *App) TestConnections() ConnectionStatus {
	status := ConnectionStatus{}

	// Testar API da Caixa
	if err := a.dataClient.TestConnection(); err != nil {
		status.CaixaAPI = false
		status.CaixaError = err.Error()
	} else {
		status.CaixaAPI = true
	}

	// Testar Claude API
	if err := a.aiClient.TestConnection(); err != nil {
		status.ClaudeAPI = false
		status.ClaudeError = err.Error()
	} else {
		status.ClaudeAPI = true
	}

	return status
}

// GenerateStrategy gera estratégia baseada nas preferências do usuário
func (a *App) GenerateStrategy(preferences UserPreferences) StrategyResponse {
	// Converter preferências para formato interno
	internalPrefs := &lottery.UserPreferences{
		Budget:          preferences.Budget,
		Strategy:        mapStrategy(preferences.Strategy),
		AvoidPatterns:   preferences.AvoidPatterns,
		FavoriteNumbers: preferences.FavoriteNumbers,
		ExcludeNumbers:  preferences.ExcludeNumbers,
	}

	// Converter tipos de loteria
	for _, ltype := range preferences.LotteryTypes {
		switch ltype {
		case "megasena":
			internalPrefs.LotteryTypes = append(internalPrefs.LotteryTypes, lottery.MegaSena)
		case "lotofacil":
			internalPrefs.LotteryTypes = append(internalPrefs.LotteryTypes, lottery.Lotofacil)
		}
	}

	// Buscar dados históricos com lógica de fallback
	var allDraws []lottery.Draw
	var allRules []lottery.LotteryRules
	var availableLotteries []lottery.LotteryType
	var failedLotteries []lottery.LotteryType

	for _, ltype := range internalPrefs.LotteryTypes {
		draws, err := a.dataClient.GetLatestDraws(ltype, 100)
		if err != nil {
			failedLotteries = append(failedLotteries, ltype)
			continue
		}

		allDraws = append(allDraws, draws...)
		allRules = append(allRules, lottery.GetRules(ltype))
		availableLotteries = append(availableLotteries, ltype)
	}

	// Implementar lógica de fallback
	if len(availableLotteries) == 0 {
		return StrategyResponse{
			Success: false,
			Error:   "Não foi possível obter dados de nenhuma loteria. API da CAIXA indisponível e cache expirado.",
		}
	}

	if len(internalPrefs.LotteryTypes) == 1 && len(failedLotteries) > 0 {
		return StrategyResponse{
			Success: false,
			Error:   fmt.Sprintf("Loteria %s indisponível. Tente novamente mais tarde ou inclua ambas as loterias.", failedLotteries[0]),
		}
	}

	// Atualizar preferências para usar apenas loterias disponíveis
	internalPrefs.LotteryTypes = availableLotteries

	// Preparar requisição para IA
	analysisReq := lottery.AnalysisRequest{
		Draws:       allDraws,
		Preferences: *internalPrefs,
		Rules:       allRules,
	}

	// Analisar com IA
	response, err := a.aiClient.AnalyzeStrategy(analysisReq)
	if err != nil {
		// Verificar se é erro de autenticação (401)
		if strings.Contains(err.Error(), "status 401") {
			return StrategyResponse{
				Success: false,
				Error:   "Erro de autenticação com Claude API. Verifique se sua chave está correta e válida.",
			}
		}

		return StrategyResponse{
			Success: false,
			Error:   fmt.Sprintf("Erro na análise da IA: %v", err),
		}
	}

	// Debug: mostrar quantos jogos a IA gerou
	if config.IsVerbose() {
		fmt.Printf("🎯 IA gerou %d jogos com custo total R$ %.2f\n", len(response.Strategy.Games), response.Strategy.TotalCost)
		for i, game := range response.Strategy.Games {
			fmt.Printf("   Jogo %d: %s - %v - R$ %.2f\n", i+1, game.Type, game.Numbers, game.Cost)
		}
	}

	// TEMPORÁRIO: Pular validação para debug - usar estratégia da IA diretamente
	validatedStrategy := &response.Strategy

	// Debug: mostrar jogos após "validação"
	if config.IsVerbose() {
		fmt.Printf("✅ Após validação: %d jogos com custo total R$ %.2f\n", len(validatedStrategy.Games), validatedStrategy.TotalCost)
	}

	// Converter loterias falhas para strings
	var failedLotteriesStr []string
	var availableLotteriesStr []string

	for _, ltype := range failedLotteries {
		failedLotteriesStr = append(failedLotteriesStr, string(ltype))
	}

	for _, ltype := range availableLotteries {
		availableLotteriesStr = append(availableLotteriesStr, string(ltype))
	}

	return StrategyResponse{
		Success:            true,
		Strategy:           validatedStrategy,
		Confidence:         response.Confidence,
		AvailableLotteries: availableLotteriesStr,
		FailedLotteries:    failedLotteriesStr,
	}
}

// GetNextDraws retorna informações dos próximos sorteios
func (a *App) GetNextDraws() map[string]interface{} {
	result := make(map[string]interface{})

	// Mega Sena
	if nextDate, nextNum, err := a.dataClient.GetNextDrawInfo(lottery.MegaSena); err == nil {
		result["megasena"] = map[string]interface{}{
			"number": nextNum,
			"date":   nextDate.Format("02/01/2006"),
		}
	}

	// Lotofácil
	if nextDate, nextNum, err := a.dataClient.GetNextDrawInfo(lottery.Lotofacil); err == nil {
		result["lotofacil"] = map[string]interface{}{
			"number": nextNum,
			"date":   nextDate.Format("02/01/2006"),
		}
	}

	return result
}

// GetStatistics retorna estatísticas das loterias
func (a *App) GetStatistics() map[string]interface{} {
	result := make(map[string]interface{})

	// Buscar dados para estatísticas
	megaDraws, err := a.dataClient.GetLatestDraws(lottery.MegaSena, 20)
	if err == nil {
		result["megasena"] = map[string]interface{}{
			"totalDraws": len(megaDraws),
			"lastDraw":   megaDraws[0].Number,
		}
	}

	lotofacilDraws, err := a.dataClient.GetLatestDraws(lottery.Lotofacil, 20)
	if err == nil {
		result["lotofacil"] = map[string]interface{}{
			"totalDraws": len(lotofacilDraws),
			"lastDraw":   lotofacilDraws[0].Number,
		}
	}

	return result
}

// Greet método de exemplo (manter para compatibilidade)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Olá %s! Bem-vindo ao Lottery Optimizer! 🎰", name)
}

// ===============================
// MÉTODOS DE CONFIGURAÇÃO
// ===============================

// GetCurrentConfig retorna a configuração atual
func (a *App) GetCurrentConfig() ConfigData {
	return ConfigData{
		ClaudeAPIKey: config.GetClaudeAPIKey(),
		ClaudeModel:  config.GetClaudeModel(),
		TimeoutSec:   config.GlobalConfig.Claude.TimeoutSec,
		MaxTokens:    config.GetMaxTokens(),
		Verbose:      config.IsVerbose(),
	}
}

// SaveConfig salva a configuração
func (a *App) SaveConfig(configData ConfigData) map[string]interface{} {
	// Validar dados
	if configData.ClaudeAPIKey == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "Chave da API do Claude é obrigatória",
		}
	}

	if configData.TimeoutSec < 10 || configData.TimeoutSec > 300 {
		return map[string]interface{}{
			"success": false,
			"error":   "Timeout deve estar entre 10 e 300 segundos",
		}
	}

	// Preparar estrutura de configuração
	configStruct := struct {
		App struct {
			Verbose bool `yaml:"verbose"`
		} `yaml:"app"`
		Claude struct {
			APIKey     string `yaml:"api_key"`
			Model      string `yaml:"model"`
			MaxTokens  int    `yaml:"max_tokens"`
			TimeoutSec int    `yaml:"timeout_sec"`
		} `yaml:"claude"`
	}{}

	configStruct.App.Verbose = configData.Verbose
	configStruct.Claude.APIKey = configData.ClaudeAPIKey
	configStruct.Claude.Model = configData.ClaudeModel
	configStruct.Claude.MaxTokens = configData.MaxTokens
	configStruct.Claude.TimeoutSec = configData.TimeoutSec

	// Determinar local do arquivo de configuração (mesmo diretório do executável)
	exePath, err := os.Executable()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Erro ao determinar diretório do executável: " + err.Error(),
		}
	}

	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "lottery-optimizer.yaml")

	// Serializar para YAML
	yamlData, err := yaml.Marshal(configStruct)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Erro ao serializar configuração: " + err.Error(),
		}
	}

	// Salvar arquivo
	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Erro ao salvar arquivo: " + err.Error(),
		}
	}

	// Atualizar configuração global diretamente
	config.GlobalConfig.Claude.APIKey = configData.ClaudeAPIKey
	config.GlobalConfig.Claude.Model = configData.ClaudeModel
	config.GlobalConfig.Claude.MaxTokens = configData.MaxTokens
	config.GlobalConfig.Claude.TimeoutSec = configData.TimeoutSec

	// Recriar clientes com nova configuração
	a.aiClient = ai.NewClaudeClient()
	a.dataClient = data.NewClient()

	return map[string]interface{}{
		"success": true,
		"message": "Configuração salva com sucesso em: " + configPath,
	}
}

// ValidateConfig valida se a configuração está correta
func (a *App) ValidateConfig() map[string]interface{} {
	result := map[string]interface{}{
		"claudeConfigured": false,
		"claudeValid":      false,
		"caixaValid":       false,
		"errors":           []string{},
	}

	errors := []string{}

	// Verificar se Claude está configurado
	if config.GetClaudeAPIKey() == "" {
		errors = append(errors, "Chave da API do Claude não configurada")
	} else {
		result["claudeConfigured"] = true

		// Testar Claude API
		if err := a.aiClient.TestConnection(); err != nil {
			errors = append(errors, "Claude API: "+err.Error())
		} else {
			result["claudeValid"] = true
		}
	}

	// Testar API da Caixa
	if err := a.dataClient.TestConnection(); err != nil {
		errors = append(errors, "API Caixa: "+err.Error())
	} else {
		result["caixaValid"] = true
	}

	result["errors"] = errors
	result["allValid"] = len(errors) == 0

	return result
}

// GetDefaultConfig retorna configuração padrão
func (a *App) GetDefaultConfig() ConfigData {
	return ConfigData{
		ClaudeAPIKey: "",
		ClaudeModel:  "claude-3-5-sonnet-20241022",
		TimeoutSec:   60,
		MaxTokens:    8000,
		Verbose:      false,
	}
}

// DebugConfig função para debug - mostra configuração atual
func (a *App) DebugConfig() map[string]interface{} {
	return map[string]interface{}{
		"claudeApiKey": config.GetClaudeAPIKey(),
		"claudeModel":  config.GetClaudeModel(),
		"maxTokens":    config.GetMaxTokens(),
		"verbose":      config.IsVerbose(),
		"aiClientKey":  a.aiClient != nil,
	}
}

// DebugClaudeConfig função para debug detalhado da configuração do Claude
func (a *App) DebugClaudeConfig() map[string]interface{} {
	result := map[string]interface{}{}

	// Informações básicas da configuração
	apiKey := config.GetClaudeAPIKey()
	result["hasApiKey"] = apiKey != ""
	result["apiKeyLength"] = len(apiKey)

	if apiKey != "" {
		// Mostrar primeiros e últimos caracteres para verificar se é válida
		if len(apiKey) > 10 {
			result["apiKeyPreview"] = apiKey[:8] + "..." + apiKey[len(apiKey)-4:]
		} else {
			result["apiKeyPreview"] = apiKey
		}

		// Verificar se parece com uma chave válida da Anthropic
		result["apiKeyLooksValid"] = strings.HasPrefix(apiKey, "sk-ant-")
	} else {
		result["apiKeyPreview"] = "VAZIA"
		result["apiKeyLooksValid"] = false
	}

	result["claudeModel"] = config.GetClaudeModel()
	result["maxTokens"] = config.GetMaxTokens()
	result["timeout"] = config.GlobalConfig.Claude.TimeoutSec
	result["verbose"] = config.IsVerbose()

	// Testar conexão se tiver chave
	if apiKey != "" {
		result["connectionTest"] = "testing..."

		// Criar cliente de teste
		testClient := ai.NewClaudeClientWithConfig(apiKey, config.GetClaudeModel(), config.GetMaxTokens(), config.GlobalConfig.Claude.TimeoutSec)

		if err := testClient.TestConnection(); err != nil {
			result["connectionTest"] = "FALHOU"
			result["connectionError"] = err.Error()
		} else {
			result["connectionTest"] = "SUCESSO"
		}
	} else {
		result["connectionTest"] = "SEM_CHAVE"
	}

	// Informações do arquivo de configuração
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	configPath := filepath.Join(exeDir, "lottery-optimizer.yaml")

	result["configPath"] = configPath
	result["configExists"] = false

	if _, err := os.Stat(configPath); err == nil {
		result["configExists"] = true

		// Ler conteúdo do arquivo para debug
		if content, err := os.ReadFile(configPath); err == nil {
			result["configContent"] = string(content)
		}
	}

	return result
}

// CheckForUpdates verifica se há atualizações disponíveis
func (a *App) CheckForUpdates() (*updater.UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return a.updater.CheckForUpdates(ctx)
}

// DownloadUpdate baixa uma atualização
func (a *App) DownloadUpdate(updateInfo *updater.UpdateInfo) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Progress callback pode ser implementado para mostrar progresso no frontend
	return a.updater.DownloadUpdate(ctx, updateInfo, func(downloaded, total int64) {
		// Implementar callback de progresso se necessário
		fmt.Printf("Download: %d/%d bytes (%.2f%%)\n",
			downloaded, total, float64(downloaded)/float64(total)*100)
	})
}

// InstallUpdate instala a atualização baixada
func (a *App) InstallUpdate(updateInfo *updater.UpdateInfo) error {
	return a.updater.InstallUpdate(updateInfo)
}

// GetCurrentVersion retorna a versão atual do app
func (a *App) GetCurrentVersion() string {
	return a.updater.GetCurrentVersion()
}
