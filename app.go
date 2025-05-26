package main

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"lottery-optimizer-gui/internal/ai"
	"lottery-optimizer-gui/internal/config"
	"lottery-optimizer-gui/internal/data"
	"lottery-optimizer-gui/internal/database"
	"lottery-optimizer-gui/internal/lottery"
	"lottery-optimizer-gui/internal/models"
	"lottery-optimizer-gui/internal/services"
	"lottery-optimizer-gui/internal/updater"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// version é a versão atual do aplicativo
var version = "1.0.21.6"

var (
	githubRepo = "cccarv82/milhoes-desktop" // Repositório correto
)

// App struct - Bridge entre Frontend e Backend
type App struct {
	ctx           context.Context
	dataClient    *data.Client
	aiClient      *ai.ClaudeClient
	updater       *updater.Updater
	savedGamesDB  *database.SavedGamesDB
	resultChecker *services.ResultChecker
}

// NewApp creates a new App application struct
func NewApp() *App {
	dataClient := data.NewClient()

	// Inicializar banco de dados de jogos salvos
	// Usar diretório absoluto baseado no executável
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Erro ao obter caminho do executável: %v\n", err)
		execPath, _ = os.Getwd() // Fallback para diretório atual
	}

	dataDir := filepath.Join(filepath.Dir(execPath), "data")
	dbPath := filepath.Join(dataDir, "saved_games.db")

	// Criar diretório se não existir com permissões adequadas
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("❌ Erro ao criar diretório de dados (%s): %v\n", dataDir, err)
	}

	fmt.Printf("📁 Inicializando banco de dados em: %s\n", dbPath)

	savedGamesDB, err := database.NewSavedGamesDB(dbPath)
	if err != nil {
		fmt.Printf("❌ ERRO ao inicializar banco de jogos salvos: %v\n", err)
		fmt.Printf("   📂 Diretório: %s\n", dataDir)
		fmt.Printf("   💾 Arquivo DB: %s\n", dbPath)

		// Verificar se o diretório existe
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			fmt.Printf("   ⚠️  Diretório não existe: %s\n", dataDir)
		} else {
			fmt.Printf("   ✅ Diretório existe: %s\n", dataDir)
		}

		// Verificar permissões
		if file, err := os.OpenFile(filepath.Join(dataDir, "test_write.tmp"), os.O_CREATE|os.O_WRONLY, 0644); err != nil {
			fmt.Printf("   ❌ Sem permissão de escrita no diretório: %v\n", err)
		} else {
			file.Close()
			os.Remove(filepath.Join(dataDir, "test_write.tmp"))
			fmt.Printf("   ✅ Permissão de escrita OK\n")
		}

		savedGamesDB = nil // Garantir que seja nil em caso de erro
	} else {
		fmt.Printf("✅ Banco de jogos salvos inicializado com sucesso!\n")
	}

	// Inicializar verificador de resultados usando o dataClient existente
	var resultChecker *services.ResultChecker
	if savedGamesDB != nil {
		resultChecker = services.NewResultChecker(dataClient, savedGamesDB)
		// Iniciar verificação automática
		resultChecker.ScheduleAutoCheck()
		fmt.Printf("✅ Verificador de resultados inicializado e agendado!\n")
	} else {
		fmt.Printf("⚠️  Verificador de resultados não inicializado (banco indisponível)\n")
	}

	return &App{
		dataClient:    dataClient,
		aiClient:      ai.NewClaudeClient(),
		updater:       updater.NewUpdater(version, githubRepo),
		savedGamesDB:  savedGamesDB,
		resultChecker: resultChecker,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Inicializar verificação automática de atualizações
	a.ScheduleUpdateCheck()

	// Verificar atualizações após 30 segundos (não bloqueante)
	go func() {
		time.Sleep(30 * time.Second)
		log.Println("🔄 Verificando atualizações na inicialização...")
		updateInfo, err := a.CheckForUpdates()
		if err != nil {
			log.Printf("❌ Erro ao verificar atualizações: %v", err)
		} else if updateInfo != nil && updateInfo.Available {
			log.Printf("🎉 Nova versão disponível: %s -> %s", version, updateInfo.Version)
		} else {
			log.Println("✅ App atualizado - versão mais recente já instalada")
		}
	}()
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

// GetCurrentVersion retorna a versão atual do aplicativo
func (a *App) GetCurrentVersion() string {
	return version
}

// ScheduleUpdateCheck agenda verificação automática de atualizações
func (a *App) ScheduleUpdateCheck() {
	if a.updater == nil {
		log.Println("❌ Updater não inicializado - auto-update desabilitado")
		return
	}

	log.Println("⏰ Iniciando verificação automática de atualizações (a cada 6 horas)")

	// Usar callback do updater para verificação automática
	a.updater.ScheduleUpdateCheck(6*time.Hour, func(updateInfo *updater.UpdateInfo, err error) {
		if err != nil {
			log.Printf("❌ Erro na verificação automática de updates: %v", err)
		} else if updateInfo != nil && updateInfo.Available {
			log.Printf("🚀 NOVA VERSÃO DISPONÍVEL: %s -> %s", version, updateInfo.Version)
			log.Printf("📦 Download: %s", updateInfo.DownloadURL)
			// Aqui você poderia implementar notificação para o usuário
		} else {
			log.Println("✅ Auto-update check: aplicativo já está na versão mais recente")
		}
	})
}

// ===============================
// MÉTODOS PARA JOGOS SALVOS
// ===============================

// SaveGame salva um jogo para verificação posterior
func (a *App) SaveGame(request models.SaveGameRequest) map[string]interface{} {
	if a.savedGamesDB == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Banco de dados de jogos salvos não disponível",
		}
	}

	game, err := a.savedGamesDB.SaveGame(request)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao salvar jogo: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
		"game":    game,
		"message": "Jogo salvo com sucesso!",
	}
}

// GetSavedGames busca jogos salvos com filtros opcionais
func (a *App) GetSavedGames(filter models.SavedGamesFilter) map[string]interface{} {
	if a.savedGamesDB == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Banco de dados de jogos salvos não disponível",
		}
	}

	games, err := a.savedGamesDB.GetSavedGames(filter)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao buscar jogos salvos: %v", err),
		}
	}

	// Adicionar resultados aos jogos que já foram verificados
	for i := range games {
		if games[i].Status == "checked" && a.resultChecker != nil {
			// Buscar resultado do jogo
			result, err := a.resultChecker.CheckSingleGame(games[i].ID)
			if err == nil && result != nil {
				games[i].Result = result
			}
		}
	}

	return map[string]interface{}{
		"success": true,
		"games":   games,
		"total":   len(games),
	}
}

// CheckGameResult verifica o resultado de um jogo específico
func (a *App) CheckGameResult(gameID string) map[string]interface{} {
	if a.resultChecker == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Verificador de resultados não disponível",
		}
	}

	result, err := a.resultChecker.CheckSingleGame(gameID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao verificar resultado: %v", err),
		}
	}

	if result == nil {
		return map[string]interface{}{
			"success": true,
			"pending": true,
			"message": "Sorteio ainda não foi realizado",
		}
	}

	return map[string]interface{}{
		"success": true,
		"result":  result,
		"message": fmt.Sprintf("Resultado verificado: %d acertos", result.HitCount),
	}
}

// CheckAllPendingResults verifica todos os jogos pendentes
func (a *App) CheckAllPendingResults() map[string]interface{} {
	if a.resultChecker == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Verificador de resultados não disponível",
		}
	}

	err := a.resultChecker.CheckPendingResults()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao verificar jogos pendentes: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": "Verificação de jogos pendentes concluída",
	}
}

// DeleteSavedGame remove um jogo salvo
func (a *App) DeleteSavedGame(gameID string) map[string]interface{} {
	if a.savedGamesDB == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Banco de dados de jogos salvos não disponível",
		}
	}

	err := a.savedGamesDB.DeleteGame(gameID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao deletar jogo: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": "Jogo removido com sucesso",
	}
}

// GetSavedGamesStats retorna estatísticas dos jogos salvos
func (a *App) GetSavedGamesStats() map[string]interface{} {
	if a.savedGamesDB == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Banco de dados de jogos salvos não disponível",
		}
	}

	stats, err := a.savedGamesDB.GetStats()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao buscar estatísticas: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
		"stats":   stats,
	}
}

// DebugSavedGamesDB retorna informações de diagnóstico do banco de dados
func (a *App) DebugSavedGamesDB() map[string]interface{} {
	// Obter informações do caminho do banco
	execPath, err := os.Executable()
	if err != nil {
		execPath, _ = os.Getwd()
	}

	dataDir := filepath.Join(filepath.Dir(execPath), "data")
	dbPath := filepath.Join(dataDir, "saved_games.db")

	debug := map[string]interface{}{
		"executablePath":           execPath,
		"dataDirectory":            dataDir,
		"databasePath":             dbPath,
		"dbInitialized":            a.savedGamesDB != nil,
		"resultCheckerInitialized": a.resultChecker != nil,
	}

	// Verificar se diretório existe
	if stat, err := os.Stat(dataDir); err != nil {
		debug["directoryExists"] = false
		debug["directoryError"] = err.Error()
	} else {
		debug["directoryExists"] = true
		debug["directoryMode"] = stat.Mode().String()
	}

	// Verificar se arquivo do banco existe
	if stat, err := os.Stat(dbPath); err != nil {
		debug["databaseFileExists"] = false
		debug["databaseFileError"] = err.Error()
	} else {
		debug["databaseFileExists"] = true
		debug["databaseFileSize"] = stat.Size()
		debug["databaseFileMode"] = stat.Mode().String()
	}

	// Testar permissões de escrita
	testFile := filepath.Join(dataDir, "test_write_permission.tmp")
	if file, err := os.OpenFile(testFile, os.O_CREATE|os.O_WRONLY, 0644); err != nil {
		debug["writePermission"] = false
		debug["writePermissionError"] = err.Error()
	} else {
		file.Close()
		os.Remove(testFile)
		debug["writePermission"] = true
	}

	// Tentar inicializar banco de dados se não estiver inicializado
	if a.savedGamesDB == nil {
		testDB, err := database.NewSavedGamesDB(dbPath)
		if err != nil {
			debug["reinitializationTest"] = false
			debug["reinitializationError"] = err.Error()
		} else {
			debug["reinitializationTest"] = true
			testDB.Close()
		}
	}

	return debug
}

// GetAppInfo retorna informações do aplicativo
func (a *App) GetAppInfo() map[string]interface{} {
	return map[string]interface{}{
		"success":           true,
		"version":           "1.0.21.6",
		"platform":          "windows",
		"repository":        "cccarv82/milhoes-desktop",
		"buildDate":         time.Now().Format("2006-01-02"),
		"autoUpdateEnabled": true,
	}
}
