package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"lottery-optimizer-gui/internal/ai"
	"lottery-optimizer-gui/internal/analytics"
	"lottery-optimizer-gui/internal/config"
	"lottery-optimizer-gui/internal/data"
	"lottery-optimizer-gui/internal/database"
	"lottery-optimizer-gui/internal/logs"
	"lottery-optimizer-gui/internal/lottery"
	"lottery-optimizer-gui/internal/models"
	"lottery-optimizer-gui/internal/notifications"
	"lottery-optimizer-gui/internal/services"
	"lottery-optimizer-gui/internal/strategy"
	"lottery-optimizer-gui/internal/updater"

	"gopkg.in/yaml.v3"
)

var (
	githubRepo   = "cccarv82/milhoes-releases" // Repositório público para releases
	logFile      *os.File
	logDir       string
	customLogger *CustomLogger
)

// CustomLogger - Logger personalizado que garante escrita em arquivo
type CustomLogger struct {
	file *os.File
}

// Printf escreve tanto no console quanto no arquivo com flush imediato
func (cl *CustomLogger) Printf(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(message, "\n") {
		message += "\n"
	}

	timestamp := time.Now().Format("2006/01/02 15:04:05.000000")
	fullMessage := fmt.Sprintf("%s %s", timestamp, message)

	// Escrever no console
	fmt.Print(fullMessage)

	// Escrever no arquivo
	if cl.file != nil {
		cl.file.WriteString(fullMessage)
		cl.file.Sync() // Flush imediato para garantir escrita
	}
}

// Close fecha o arquivo de log
func (cl *CustomLogger) Close() {
	if cl.file != nil {
		cl.Printf("🚀 =================================")
		cl.Printf("🚀 LOTTERY OPTIMIZER FINALIZADO")
		cl.Printf("🚀 =================================")
		cl.file.Close()
	}
}

// App struct - Bridge entre Frontend e Backend
type App struct {
	ctx              context.Context
	dataClient       *data.Client
	aiClient         *ai.ClaudeClient
	updater          *updater.Updater
	savedGamesDB     *database.SavedGamesDB
	resultChecker    *services.ResultChecker
	contestPredictor *services.ContestPredictor // Nova feature: Preditor de Concursos Quentes
	updateStatus     *UpdateStatus              // Status de atualização para o frontend
	pendingUpdate    *updater.UpdateInfo        // Informações da atualização pendente
}

// UpdateStatus representa o status atual da atualização
type UpdateStatus struct {
	Status  string `json:"status"`  // "none", "checking", "downloading", "installed_silently", "download_failed", "install_failed"
	Message string `json:"message"` // Mensagem detalhada para o usuário
	Version string `json:"version"` // Nova versão disponível
}

// NewApp creates a new App application struct
func NewApp() *App {
	// Inicializar logging em arquivo PRIMEIRO
	fmt.Println("🚀 Iniciando Lottery Optimizer...")
	fmt.Printf("🚀 Versão: %s\n", version)

	if err := initCustomLogging(); err != nil {
		fmt.Printf("⚠️ Erro ao inicializar logging em arquivo: %v\n", err)
		fmt.Println("⚠️ Continuando sem logging em arquivo - apenas console")
	} else {
		fmt.Println("✅ Sistema de logging em arquivo inicializado com sucesso!")
		// Teste adicional após inicialização
		customLogger.Printf("🧪 TESTE PÓS-INICIALIZAÇÃO - NewApp iniciado com logging funcional")
	}

	customLogger.Printf("🔧 Inicializando aplicação...")

	// Inicializar clientes
	dataClient := data.NewClient()
	customLogger.Printf("✅ Cliente de dados inicializado")

	// Inicializar banco de dados de jogos salvos
	var savedGamesDB *database.SavedGamesDB
	var resultChecker *services.ResultChecker

	// Determinar caminho do banco de dados
	execPath, err := os.Executable()
	if err != nil {
		execPath, _ = os.Getwd()
		customLogger.Printf("⚠️ Erro ao obter executável, usando diretório atual: %v", err)
	}

	dataDir := filepath.Join(filepath.Dir(execPath), "data")
	dbPath := filepath.Join(dataDir, "saved_games.db")

	customLogger.Printf("📁 Caminho do banco de dados: %s", dbPath)

	// Criar diretório se não existir
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		customLogger.Printf("❌ Erro ao criar diretório de dados: %v", err)
	} else {
		customLogger.Printf("✅ Diretório de dados criado/verificado")

		// Tentar inicializar banco
		if db, err := database.NewSavedGamesDB(dbPath); err != nil {
			customLogger.Printf("❌ Erro ao inicializar banco de dados: %v", err)
		} else {
			savedGamesDB = db
			customLogger.Printf("✅ Banco de dados de jogos salvos inicializado")

			// Definir instância global para analytics
			if savedGamesDB != nil {
				database.SetGlobalDB(savedGamesDB)
				logs.LogMain("✅ Instância global do database definida para analytics")
			}

			// Inicializar sistema de notificações
			notifications.InitNotificationManager()
			logs.LogMain("🔔 Sistema de notificações inicializado")

			// Inicializar verificador de resultados
			resultChecker = services.NewResultChecker(dataClient, savedGamesDB)
			customLogger.Printf("✅ Verificador de resultados inicializado")
		}
	}

	// Carregar configuração existente
	loadExistingConfig()

	// Inicializar preditor de concursos quentes
	contestPredictor := services.NewContestPredictor(dataClient)
	customLogger.Printf("🔮 Preditor de Concursos Quentes inicializado")

	customLogger.Printf("✅ App inicializado com sucesso - Versão %s", version)

	return &App{
		dataClient:       dataClient,
		aiClient:         ai.NewClaudeClient(),
		updater:          updater.NewUpdater(version, githubRepo),
		savedGamesDB:     savedGamesDB,
		resultChecker:    resultChecker,
		contestPredictor: contestPredictor,
		updateStatus:     &UpdateStatus{},
		pendingUpdate:    nil,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	customLogger.Printf("🚀 =================================")
	customLogger.Printf("🚀 APP STARTUP INICIADO")
	customLogger.Printf("🚀 =================================")

	customLogger.Printf("✅ Context salvo com sucesso")

	// Verificação de atualizações na inicialização (em background)
	go func() {
		customLogger.Printf("🔍 Verificando atualizações na inicialização...")
		updateInfo, err := a.CheckForUpdates()
		if err != nil {
			customLogger.Printf("⚠️ Erro na verificação inicial de atualizações: %v", err)
		} else if updateInfo != nil && updateInfo.Available {
			customLogger.Printf("🚀 NOVA VERSÃO DISPONÍVEL: %s -> %s", version, updateInfo.Version)
			customLogger.Printf("📦 Download: %s", updateInfo.DownloadURL)
			// Salvar informações da atualização para o frontend
			a.pendingUpdate = updateInfo
			a.updateStatus.Status = "available"
			a.updateStatus.Message = fmt.Sprintf("Nova versão %s disponível", updateInfo.Version)
			a.updateStatus.Version = updateInfo.Version
		} else {
			customLogger.Printf("✅ Aplicativo está atualizado")
			a.updateStatus.Status = "up_to_date"
			a.updateStatus.Message = "Aplicativo está atualizado"
		}
	}()

	customLogger.Printf("🚀 =================================")
	customLogger.Printf("🚀 APP STARTUP CONCLUÍDO")
	customLogger.Printf("🚀 =================================")
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

// getConfigPath retorna o caminho do arquivo de configuração com estratégia de fallback
func getConfigPath() (string, error) {
	configFileName := "lottery-optimizer.yaml"

	customLogger.Printf("🔍 getConfigPath iniciado - procurando por: %s", configFileName)

	// ESTRATÉGIA 1: Diretório de dados do usuário (APPDATA no Windows)
	userConfigDir, err := os.UserConfigDir()
	customLogger.Printf("📁 ESTRATÉGIA 1 - UserConfigDir: %s, err: %v", userConfigDir, err)

	if err == nil {
		appDataDir := filepath.Join(userConfigDir, "lottery-optimizer")
		appDataConfigPath := filepath.Join(appDataDir, configFileName)

		customLogger.Printf("📁 Tentando APPDATA: %s", appDataConfigPath)

		// Criar diretório se não existir
		if err := os.MkdirAll(appDataDir, 0755); err == nil {
			customLogger.Printf("✅ Diretório APPDATA criado/existe: %s", appDataDir)

			// Verificar se pode escrever
			testFile := filepath.Join(appDataDir, "write_test.tmp")
			if err := os.WriteFile(testFile, []byte("test"), 0644); err == nil {
				os.Remove(testFile)
				customLogger.Printf("✅ APPDATA é writável - usando: %s", appDataConfigPath)

				// MIGRAÇÃO AUTOMÁTICA: Se arquivo existe no diretório do executável, copiar para APPDATA
				if _, err := os.Stat(appDataConfigPath); os.IsNotExist(err) {
					customLogger.Printf("🔍 Arquivo não existe em APPDATA, verificando migração...")
					if exePath, err := os.Executable(); err == nil {
						oldConfigPath := filepath.Join(filepath.Dir(exePath), configFileName)
						customLogger.Printf("🔍 Verificando arquivo antigo em: %s", oldConfigPath)
						if _, err := os.Stat(oldConfigPath); err == nil {
							customLogger.Printf("📁 Arquivo encontrado no local antigo, migrando...")
							if content, err := os.ReadFile(oldConfigPath); err == nil {
								if err := os.WriteFile(appDataConfigPath, content, 0644); err == nil {
									customLogger.Printf("🔄 Migração automática CONCLUÍDA: %s -> %s", oldConfigPath, appDataConfigPath)
								} else {
									customLogger.Printf("❌ Erro na migração - escrita: %v", err)
								}
							} else {
								customLogger.Printf("❌ Erro na migração - leitura: %v", err)
							}
						} else {
							customLogger.Printf("📁 Arquivo antigo não encontrado em: %s", oldConfigPath)
						}
					}
				} else {
					customLogger.Printf("✅ Arquivo já existe em APPDATA")
				}

				return appDataConfigPath, nil
			} else {
				customLogger.Printf("❌ APPDATA não é writável: %v", err)
			}
		} else {
			customLogger.Printf("❌ Erro ao criar diretório APPDATA: %v", err)
		}
	}

	// ESTRATÉGIA 2: Diretório do executável (fallback)
	customLogger.Printf("🔍 ESTRATÉGIA 2 - Tentando diretório do executável...")
	exePath, err := os.Executable()
	if err != nil {
		customLogger.Printf("❌ Erro ao obter caminho do executável: %v", err)
		customLogger.Printf("🔍 ESTRATÉGIA 3 - Usando diretório atual como último recurso")
		return configFileName, err // Fallback para diretório atual
	}

	exeDir := filepath.Dir(exePath)
	exeConfigPath := filepath.Join(exeDir, configFileName)

	customLogger.Printf("📁 Testando diretório do executável: %s", exeConfigPath)

	// Verificar se pode escrever no diretório do executável
	testFile := filepath.Join(exeDir, "write_test.tmp")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err == nil {
		os.Remove(testFile)
		customLogger.Printf("⚠️ USANDO diretório do executável (fallback): %s", exeConfigPath)
		return exeConfigPath, nil
	} else {
		customLogger.Printf("❌ Diretório do executável não é writável: %v", err)
	}

	// ESTRATÉGIA 3: Diretório atual (último recurso)
	customLogger.Printf("⚠️ USANDO diretório atual (último recurso): %s", configFileName)
	return configFileName, nil
}

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
		draws, err := a.dataClient.GetLatestDraws(ltype, 250) // 250 sorteios POR LOTERIA
		if err != nil {
			failedLotteries = append(failedLotteries, ltype)
			continue
		}

		allDraws = append(allDraws, draws...)
		allRules = append(allRules, lottery.GetRules(ltype))
		availableLotteries = append(availableLotteries, ltype)

		// Log para confirmar quantos dados foram carregados por loteria
		if config.IsVerbose() {
			customLogger.Printf("✅ Carregados %d sorteios históricos de %s", len(draws), ltype)
		}
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
		customLogger.Printf("🎯 IA gerou %d jogos com custo total R$ %.2f", len(response.Strategy.Games), response.Strategy.TotalCost)
		for i, game := range response.Strategy.Games {
			customLogger.Printf("   Jogo %d: %s - %v - R$ %.2f", i+1, game.Type, game.Numbers, game.Cost)
		}
	}

	// VALIDAÇÃO OBRIGATÓRIA: Validar e ajustar estratégia da IA
	customLogger.Printf("🔍 Validando estratégia da IA...")
	validatedStrategy := strategy.ValidateAndAdjustStrategy(&response.Strategy, *internalPrefs)

	// Recalcular totalCost corretamente baseado nos jogos validados
	totalCost := 0.0
	for _, game := range validatedStrategy.Games {
		totalCost += game.Cost
	}
	validatedStrategy.TotalCost = totalCost

	// Log após validação
	customLogger.Printf("✅ Após validação: %d jogos válidos com custo total R$ %.2f", len(validatedStrategy.Games), validatedStrategy.TotalCost)

	// VALIDAÇÃO CRÍTICA: Garantir que não excede o orçamento
	if totalCost > internalPrefs.Budget {
		customLogger.Printf("⚠️ Custo total R$ %.2f excede orçamento R$ %.2f - removendo jogos mais baratos", totalCost, internalPrefs.Budget)

		// ESTRATÉGIA CORRETA: Remover jogos para ficar dentro do orçamento
		validGames := optimizeBudgetUsage(validatedStrategy.Games, internalPrefs.Budget)
		currentCost := 0.0
		for _, game := range validGames {
			currentCost += game.Cost
		}

		validatedStrategy.Games = validGames
		validatedStrategy.TotalCost = currentCost

		customLogger.Printf("✅ Orçamento ajustado: %d jogos por R$ %.2f (%.1f%% do orçamento)",
			len(validGames), currentCost, (currentCost/internalPrefs.Budget)*100)

		// Atualizar reasoning para explicar o ajuste
		if validatedStrategy.Reasoning != "" {
			validatedStrategy.Reasoning += fmt.Sprintf("\n\n⚠️ AJUSTE DE ORÇAMENTO: A estratégia original custaria R$ %.2f, mas foi ajustada para R$ %.2f (%.1f%% do seu orçamento de R$ %.2f) removendo os jogos mais baratos para manter apenas os jogos de maior qualidade dentro do orçamento disponível.", totalCost, currentCost, (currentCost/internalPrefs.Budget)*100, internalPrefs.Budget)
		}
	} else {
		// Orçamento OK - aceitar que nem todo orçamento precisa ser usado
		remainingBudget := internalPrefs.Budget - totalCost
		customLogger.Printf("✅ Orçamento respeitado: R$ %.2f usado de R$ %.2f (%.1f%% - R$ %.2f restantes)",
			totalCost, internalPrefs.Budget, (totalCost/internalPrefs.Budget)*100, remainingBudget)

		if remainingBudget > 0 {
			customLogger.Printf("💡 Orçamento restante de R$ %.2f é normal - priorizamos qualidade dos jogos gerados pela IA", remainingBudget)
		}
	}

	// Debug: mostrar jogos após "validação"
	if config.IsVerbose() {
		customLogger.Printf("✅ Após validação: %d jogos com custo total R$ %.2f", len(validatedStrategy.Games), validatedStrategy.TotalCost)
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

	// Buscar dados para estatísticas (usando mais dados para melhor precisão)
	megaDraws, err := a.dataClient.GetLatestDraws(lottery.MegaSena, 50)
	if err == nil {
		result["megasena"] = map[string]interface{}{
			"totalDraws": len(megaDraws),
			"lastDraw":   megaDraws[0].Number,
		}
	}

	lotofacilDraws, err := a.dataClient.GetLatestDraws(lottery.Lotofacil, 50)
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
func (a *App) GetCurrentConfig() map[string]interface{} {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000000")
	customLogger.Printf("📖 [%s] GetCurrentConfig INICIADO", timestamp)

	// DEBUG: Verificar estado atual da GlobalConfig
	customLogger.Printf("🔍 [%s] DEBUG GlobalConfig - APIKey length: %d", timestamp, len(config.GlobalConfig.Claude.APIKey))
	customLogger.Printf("🔍 [%s] DEBUG GlobalConfig - Model: %s", timestamp, config.GlobalConfig.Claude.Model)
	customLogger.Printf("🔍 [%s] DEBUG GlobalConfig - MaxTokens: %d", timestamp, config.GlobalConfig.Claude.MaxTokens)
	customLogger.Printf("🔍 [%s] DEBUG GetClaudeAPIKey() length: %d", timestamp, len(config.GetClaudeAPIKey()))

	// PRIORIDADE 1: Usar configuração já carregada na memória (config.GlobalConfig)
	if config.GetClaudeAPIKey() != "" {
		customLogger.Printf("✅ [%s] GetCurrentConfig: Usando configuração da MEMÓRIA (GlobalConfig)", timestamp)

		result := map[string]interface{}{
			"exists":       true,
			"claudeApiKey": config.GetClaudeAPIKey(),
			"claudeModel":  config.GetClaudeModel(),
			"maxTokens":    config.GetMaxTokens(),
			"timeoutSec":   config.GlobalConfig.Claude.TimeoutSec,
			"verbose":      config.IsVerbose(),
			"source":       "memory", // Debug: indicar fonte
			"debug": map[string]interface{}{
				"source":    "GlobalConfig",
				"apiKeyLen": len(config.GetClaudeAPIKey()),
			},
		}

		customLogger.Printf("✅ [%s] GetCurrentConfig: Retornando da MEMÓRIA - APIKey length=%d",
			timestamp, len(config.GetClaudeAPIKey()))
		customLogger.Printf("🔍 [%s] RETORNO COMPLETO: %+v", timestamp, result)
		flushLogs()
		return result
	}

	// PRIORIDADE 2: Fallback para leitura do arquivo (se GlobalConfig estiver vazio)
	customLogger.Printf("⚠️ [%s] GetCurrentConfig: GlobalConfig vazio, fazendo fallback para ARQUIVO", timestamp)

	configPath, err := getConfigPath()
	if err != nil {
		customLogger.Printf("❌ [%s] GetCurrentConfig: Erro ao determinar caminho: %v", timestamp, err)
		flushLogs()
		return map[string]interface{}{
			"success": false,
			"error":   "Erro ao determinar caminho da configuração: " + err.Error(),
		}
	}

	customLogger.Printf("📁 [%s] GetCurrentConfig: Tentando ler arquivo: %s", timestamp, configPath)

	// Verificar se arquivo existe
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		customLogger.Printf("⚠️ [%s] GetCurrentConfig: Arquivo não existe, retornando configuração padrão", timestamp)
		flushLogs()
		return map[string]interface{}{
			"exists":       false,
			"claudeApiKey": "",
			"claudeModel":  "claude-opus-4-20250514",
			"maxTokens":    8000,
			"timeoutSec":   60,
			"verbose":      false,
			"source":       "default", // Debug: indicar fonte
		}
	}

	customLogger.Printf("✅ [%s] GetCurrentConfig: Arquivo existe, lendo conteúdo...", timestamp)

	// Ler arquivo
	data, err := os.ReadFile(configPath)
	if err != nil {
		customLogger.Printf("❌ [%s] GetCurrentConfig: Erro ao ler arquivo: %v", timestamp, err)
		flushLogs()
		return map[string]interface{}{
			"success": false,
			"error":   "Erro ao ler arquivo de configuração: " + err.Error(),
		}
	}

	customLogger.Printf("📖 Arquivo lido (%d bytes)", len(data))

	// Parse YAML
	var configStruct struct {
		App struct {
			Verbose bool `yaml:"verbose"`
		} `yaml:"app"`
		Claude struct {
			APIKey     string `yaml:"api_key"`
			Model      string `yaml:"model"`
			MaxTokens  int    `yaml:"max_tokens"`
			TimeoutSec int    `yaml:"timeout_sec"`
		} `yaml:"claude"`
	}

	if err := yaml.Unmarshal(data, &configStruct); err != nil {
		customLogger.Printf("❌ [%s] GetCurrentConfig: Erro ao fazer parse do YAML: %v", timestamp, err)
		flushLogs()
		return map[string]interface{}{
			"success": false,
			"error":   "Erro ao fazer parse da configuração: " + err.Error(),
		}
	}

	customLogger.Printf("✅ [%s] GetCurrentConfig: Parse realizado - APIKey length=%d, Model=%s",
		timestamp, len(configStruct.Claude.APIKey), configStruct.Claude.Model)

	// Atualizar configuração global se a chave estiver definida (sincronizar arquivo -> memória)
	if configStruct.Claude.APIKey != "" {
		config.GlobalConfig.Claude.APIKey = configStruct.Claude.APIKey
		config.GlobalConfig.Claude.Model = configStruct.Claude.Model
		config.GlobalConfig.Claude.MaxTokens = configStruct.Claude.MaxTokens
		config.GlobalConfig.Claude.TimeoutSec = configStruct.Claude.TimeoutSec

		customLogger.Printf("✅ CONFIGURAÇÃO SINCRONIZADA: Arquivo -> Memória - APIKey length=%d",
			len(configStruct.Claude.APIKey))
	} else {
		customLogger.Printf("⚠️ Arquivo de configuração existe mas não contém chave Claude API")
	}

	result := map[string]interface{}{
		"exists":       true,
		"claudeApiKey": configStruct.Claude.APIKey,
		"claudeModel":  configStruct.Claude.Model,
		"maxTokens":    configStruct.Claude.MaxTokens,
		"timeoutSec":   configStruct.Claude.TimeoutSec,
		"verbose":      configStruct.App.Verbose,
		"source":       "file", // Debug: indicar fonte
		"debug": map[string]interface{}{
			"configPath": configPath,
			"fileSize":   len(data),
			"apiKeyLen":  len(configStruct.Claude.APIKey),
			"source":     "file",
		},
	}

	customLogger.Printf("✅ [%s] GetCurrentConfig: Retornando do ARQUIVO - APIKey presente: %t",
		timestamp, configStruct.Claude.APIKey != "")

	flushLogs()

	return result
}

// SaveConfig salva a configuração
func (a *App) SaveConfig(configData ConfigData) map[string]interface{} {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000000")
	customLogger.Printf("🔧 [%s] SaveConfig INICIADO - Dados recebidos: APIKey length=%d, Model=%s",
		timestamp, len(configData.ClaudeAPIKey), configData.ClaudeModel)

	// Validar dados
	if configData.ClaudeAPIKey == "" {
		customLogger.Printf("❌ [%s] Erro: Chave da API do Claude é obrigatória", timestamp)
		flushLogs()
		return map[string]interface{}{
			"success": false,
			"error":   "Chave da API do Claude é obrigatória",
		}
	}

	if configData.TimeoutSec < 10 || configData.TimeoutSec > 300 {
		customLogger.Printf("❌ [%s] Erro: Timeout inválido: %d", timestamp, configData.TimeoutSec)
		flushLogs()
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

	customLogger.Printf("📦 [%s] Estrutura de configuração criada - APIKey length=%d", timestamp, len(configStruct.Claude.APIKey))

	configPath, err := getConfigPath()
	if err != nil {
		customLogger.Printf("❌ [%s] Erro ao determinar caminho da configuração: %v", timestamp, err)
		flushLogs()
		return map[string]interface{}{
			"success": false,
			"error":   "Erro ao determinar caminho da configuração: " + err.Error(),
		}
	}

	customLogger.Printf("📁 [%s] Caminho da configuração: %s", timestamp, configPath)
	configDir := filepath.Dir(configPath)
	customLogger.Printf("📁 [%s] Diretório da configuração: %s", timestamp, configDir)

	// Verificar se diretório é writável
	testPath := filepath.Join(configDir, "write_test_temp.txt")
	if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
		customLogger.Printf("❌ [%s] Diretório não é writável: %v", timestamp, err)
		flushLogs()
		return map[string]interface{}{
			"success": false,
			"error":   "Diretório não é writável: " + err.Error(),
		}
	}
	os.Remove(testPath)
	customLogger.Printf("✅ [%s] Diretório é writável", timestamp)

	// Serializar para YAML
	yamlData, err := yaml.Marshal(configStruct)
	if err != nil {
		customLogger.Printf("❌ [%s] Erro ao serializar configuração: %v", timestamp, err)
		flushLogs()
		return map[string]interface{}{
			"success": false,
			"error":   "Erro ao serializar configuração: " + err.Error(),
		}
	}

	customLogger.Printf("📝 [%s] YAML gerado (%d bytes)", timestamp, len(yamlData))

	// Salvar arquivo
	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		customLogger.Printf("❌ [%s] Erro ao salvar arquivo: %v", timestamp, err)
		flushLogs()
		return map[string]interface{}{
			"success": false,
			"error":   "Erro ao salvar arquivo: " + err.Error(),
		}
	}

	customLogger.Printf("✅ [%s] Arquivo salvo com sucesso", timestamp)

	// Atualizar configuração global diretamente
	config.GlobalConfig.Claude.APIKey = configData.ClaudeAPIKey
	config.GlobalConfig.Claude.Model = configData.ClaudeModel
	config.GlobalConfig.Claude.MaxTokens = configData.MaxTokens
	config.GlobalConfig.Claude.TimeoutSec = configData.TimeoutSec

	customLogger.Printf("✅ [%s] GlobalConfig atualizado", timestamp)

	// Recriar clientes com nova configuração
	a.aiClient = ai.NewClaudeClient()
	a.dataClient = data.NewClient()

	customLogger.Printf("✅ [%s] Clientes recriados", timestamp)

	// Flush final para garantir que tudo foi escrito
	flushLogs()

	return map[string]interface{}{
		"success": true,
		"message": "Configuração salva com sucesso em: " + configPath,
		"debug": map[string]interface{}{
			"configPath": configPath,
			"yamlSize":   len(yamlData),
			"apiKeyLen":  len(configData.ClaudeAPIKey),
		},
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
		ClaudeModel:  "claude-opus-4-20250514",
		TimeoutSec:   60,
		MaxTokens:    8000,
		Verbose:      false,
	}
}

// CheckForUpdates verifica se há atualizações disponíveis
func (a *App) CheckForUpdates() (*updater.UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return a.updater.CheckForUpdates(ctx)
}

// GetCurrentVersion retorna a versão atual do aplicativo
func (a *App) GetCurrentVersion() string {
	return version
}

// ===============================
// MÉTODOS PARA JOGOS SALVOS
// ===============================

// SaveGame salva um jogo para verificação posterior
func (a *App) SaveGame(request models.SaveGameRequest) map[string]interface{} {
	logs.LogDatabase("🎯 Tentativa de salvar jogo: %s com %d números", request.LotteryType, len(request.Numbers))
	logs.LogDatabase("📊 Detalhes: Data=%s, Concurso=%d, Números=%v", request.ExpectedDraw, request.ContestNumber, request.Numbers)

	if a.savedGamesDB == nil {
		logs.LogError(logs.CategoryDatabase, "❌ Banco de dados de jogos salvos não disponível")
		return map[string]interface{}{
			"success": false,
			"error":   "Banco de dados de jogos salvos não disponível",
		}
	}

	// Validações básicas
	if request.LotteryType == "" {
		logs.LogError(logs.CategoryDatabase, "❌ Tipo de loteria não informado")
		return map[string]interface{}{
			"success": false,
			"error":   "Tipo de loteria não informado",
		}
	}

	if len(request.Numbers) == 0 {
		logs.LogError(logs.CategoryDatabase, "❌ Nenhum número informado")
		return map[string]interface{}{
			"success": false,
			"error":   "Nenhum número informado",
		}
	}

	if request.ExpectedDraw == "" {
		logs.LogError(logs.CategoryDatabase, "❌ Data do sorteio não informada")
		return map[string]interface{}{
			"success": false,
			"error":   "Data do sorteio não informada",
		}
	}

	if request.ContestNumber <= 0 {
		logs.LogError(logs.CategoryDatabase, "❌ Número do concurso inválido: %d", request.ContestNumber)
		return map[string]interface{}{
			"success": false,
			"error":   "Número do concurso inválido",
		}
	}

	// Tentar salvar no banco
	logs.LogDatabase("💾 Salvando no banco de dados...")
	game, err := a.savedGamesDB.SaveGame(request)
	if err != nil {
		logs.LogError(logs.CategoryDatabase, "❌ Erro ao salvar jogo no banco: %v", err)
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao salvar jogo: %v", err),
		}
	}

	logs.LogDatabase("✅ Jogo salvo com sucesso! ID: %s", game.ID)

	return map[string]interface{}{
		"success": true,
		"game":    game,
		"message": "Jogo salvo com sucesso!",
	}
}

// SaveManualGame salva um jogo adicionado manualmente pelo usuário
func (a *App) SaveManualGame(request models.SaveGameRequest) map[string]interface{} {
	logs.LogDatabase("🖐️ Tentativa de salvar jogo MANUAL: %s com %d números", request.LotteryType, len(request.Numbers))
	logs.LogDatabase("📊 Detalhes manuais: Data=%s, Concurso=%d, Números=%v", request.ExpectedDraw, request.ContestNumber, request.Numbers)

	if a.savedGamesDB == nil {
		logs.LogError(logs.CategoryDatabase, "❌ Banco de dados de jogos salvos não disponível")
		return map[string]interface{}{
			"success": false,
			"error":   "Banco de dados de jogos salvos não disponível",
		}
	}

	// Validações específicas para jogos manuais
	if request.LotteryType == "" {
		logs.LogError(logs.CategoryDatabase, "❌ Tipo de loteria não informado")
		return map[string]interface{}{
			"success": false,
			"error":   "Tipo de loteria é obrigatório",
		}
	}

	// Validar tipo de loteria
	if request.LotteryType != "mega-sena" && request.LotteryType != "lotofacil" {
		logs.LogError(logs.CategoryDatabase, "❌ Tipo de loteria inválido: %s", request.LotteryType)
		return map[string]interface{}{
			"success": false,
			"error":   "Tipo de loteria deve ser 'mega-sena' ou 'lotofacil'",
		}
	}

	if len(request.Numbers) == 0 {
		logs.LogError(logs.CategoryDatabase, "❌ Nenhum número informado")
		return map[string]interface{}{
			"success": false,
			"error":   "Pelo menos um número deve ser informado",
		}
	}

	// Validações específicas por loteria
	if request.LotteryType == "mega-sena" {
		if len(request.Numbers) < 6 || len(request.Numbers) > 15 {
			logs.LogError(logs.CategoryDatabase, "❌ Mega-Sena: números inválidos (%d), deve ter entre 6 e 15", len(request.Numbers))
			return map[string]interface{}{
				"success": false,
				"error":   "Mega-Sena deve ter entre 6 e 15 números",
			}
		}
		// Verificar se números estão no range 1-60
		for _, num := range request.Numbers {
			if num < 1 || num > 60 {
				logs.LogError(logs.CategoryDatabase, "❌ Mega-Sena: número %d fora do range (1-60)", num)
				return map[string]interface{}{
					"success": false,
					"error":   fmt.Sprintf("Mega-Sena: número %d deve estar entre 1 e 60", num),
				}
			}
		}
	} else if request.LotteryType == "lotofacil" {
		if len(request.Numbers) < 15 || len(request.Numbers) > 20 {
			logs.LogError(logs.CategoryDatabase, "❌ Lotofácil: números inválidos (%d), deve ter entre 15 e 20", len(request.Numbers))
			return map[string]interface{}{
				"success": false,
				"error":   "Lotofácil deve ter entre 15 e 20 números",
			}
		}
		// Verificar se números estão no range 1-25
		for _, num := range request.Numbers {
			if num < 1 || num > 25 {
				logs.LogError(logs.CategoryDatabase, "❌ Lotofácil: número %d fora do range (1-25)", num)
				return map[string]interface{}{
					"success": false,
					"error":   fmt.Sprintf("Lotofácil: número %d deve estar entre 1 e 25", num),
				}
			}
		}
	}

	// Verificar duplicatas
	seen := make(map[int]bool)
	for _, num := range request.Numbers {
		if seen[num] {
			logs.LogError(logs.CategoryDatabase, "❌ Número duplicado: %d", num)
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Número %d está duplicado", num),
			}
		}
		seen[num] = true
	}

	if request.ExpectedDraw == "" {
		logs.LogError(logs.CategoryDatabase, "❌ Data do sorteio não informada")
		return map[string]interface{}{
			"success": false,
			"error":   "Data do sorteio é obrigatória",
		}
	}

	if request.ContestNumber <= 0 {
		logs.LogError(logs.CategoryDatabase, "❌ Número do concurso inválido: %d", request.ContestNumber)
		return map[string]interface{}{
			"success": false,
			"error":   "Número do concurso deve ser maior que zero",
		}
	}

	// Tentar salvar no banco
	logs.LogDatabase("💾 Salvando jogo manual no banco de dados...")
	game, err := a.savedGamesDB.SaveGame(request)
	if err != nil {
		logs.LogError(logs.CategoryDatabase, "❌ Erro ao salvar jogo manual no banco: %v", err)
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao salvar jogo: %v", err),
		}
	}

	logs.LogDatabase("✅ Jogo manual salvo com sucesso! ID: %s", game.ID)

	return map[string]interface{}{
		"success": true,
		"game":    game,
		"message": "Jogo adicionado manualmente com sucesso!",
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

	return map[string]interface{}{
		"success": true,
		"games":   games,
		"total":   len(games),
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

// ===============================
// ANALYTICS & PERFORMANCE DASHBOARD - V2.0.0
// ===============================

// GetPerformanceMetrics retorna todas as métricas de performance do usuário
func (a *App) GetPerformanceMetrics() map[string]interface{} {
	metrics, err := analytics.CalculatePerformanceMetrics()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao calcular métricas: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
		"metrics": metrics,
	}
}

// GetNumberFrequencyAnalysis retorna análise de frequência de números
func (a *App) GetNumberFrequencyAnalysis(lotteryType string) map[string]interface{} {
	frequencies, err := analytics.GetNumberFrequencyAnalysis(lotteryType)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao analisar frequência: %v", err),
		}
	}

	return map[string]interface{}{
		"success":      true,
		"frequencies":  frequencies,
		"lotteryType":  lotteryType,
		"totalNumbers": len(frequencies),
	}
}

// GetROICalculator retorna cálculos detalhados de ROI
func (a *App) GetROICalculator(investment float64, timeframe string) map[string]interface{} {
	metrics, err := analytics.CalculatePerformanceMetrics()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao calcular ROI: %v", err),
		}
	}

	// Determinar período baseado no timeframe
	var periodMetrics analytics.PeriodMetrics
	switch timeframe {
	case "30d":
		periodMetrics = metrics.Last30Days
	case "90d":
		periodMetrics = metrics.Last90Days
	case "365d":
		periodMetrics = metrics.Last365Days
	default:
		// Usar dados totais
		periodMetrics = analytics.PeriodMetrics{
			Games:      metrics.TotalGames,
			Investment: metrics.TotalInvestment,
			Winnings:   metrics.TotalWinnings,
			ROI:        metrics.ROIPercentage,
			WinRate:    metrics.WinRate,
		}
	}

	// Projeções baseadas no investimento fornecido
	projectedWinnings := 0.0
	projectedROI := 0.0

	if periodMetrics.Investment > 0 {
		winRate := periodMetrics.Winnings / periodMetrics.Investment
		projectedWinnings = investment * winRate
		projectedROI = ((projectedWinnings - investment) / investment) * 100
	}

	return map[string]interface{}{
		"success": true,
		"calculation": map[string]interface{}{
			"investment":        investment,
			"timeframe":         timeframe,
			"projectedWinnings": projectedWinnings,
			"projectedROI":      projectedROI,
			"projectedProfit":   projectedWinnings - investment,
			"historicalROI":     periodMetrics.ROI,
			"historicalWinRate": periodMetrics.WinRate,
			"basedOnGames":      periodMetrics.Games,
			"confidence":        getConfidenceLevel(periodMetrics.Games),
			"recommendation":    getROIRecommendation(projectedROI, periodMetrics.Games),
		},
	}
}

// getConfidenceLevel retorna nível de confiança baseado no número de jogos
func getConfidenceLevel(games int) string {
	if games >= 100 {
		return "Alta"
	} else if games >= 50 {
		return "Média"
	} else if games >= 20 {
		return "Baixa"
	}
	return "Muito Baixa"
}

// getROIRecommendation retorna recomendação baseada no ROI projetado
func getROIRecommendation(roi float64, games int) string {
	if games < 10 {
		return "Dados insuficientes para recomendação precisa. Continue jogando para obter análises mais confiáveis."
	}

	if roi > 0 {
		return fmt.Sprintf("Performance positiva! ROI de %.2f%% indica estratégia promissora.", roi)
	} else if roi > -20 {
		return "ROI ligeiramente negativo. Considere ajustar estratégia ou aguardar mais resultados."
	} else {
		return "ROI significativamente negativo. Recomenda-se revisar estratégia ou reduzir investimento."
	}
}

// GetDashboardSummary retorna resumo executivo para o dashboard
func (a *App) GetDashboardSummary() map[string]interface{} {
	metrics, err := analytics.CalculatePerformanceMetrics()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao gerar resumo: %v", err),
		}
	}

	// Determinar tendência
	trend := "neutral"
	if len(metrics.MonthlyTrends) >= 2 {
		current := metrics.MonthlyTrends[len(metrics.MonthlyTrends)-1]
		previous := metrics.MonthlyTrends[len(metrics.MonthlyTrends)-2]

		if current.ROI > previous.ROI {
			trend = "up"
		} else if current.ROI < previous.ROI {
			trend = "down"
		}
	}

	// Calcular streak atual
	currentStreak := map[string]interface{}{
		"type":  "none",
		"count": 0,
	}

	if metrics.CurrentWinStreak > 0 {
		currentStreak["type"] = "win"
		currentStreak["count"] = metrics.CurrentWinStreak
	} else if metrics.CurrentLossStreak > 0 {
		currentStreak["type"] = "loss"
		currentStreak["count"] = metrics.CurrentLossStreak
	}

	return map[string]interface{}{
		"success": true,
		"summary": map[string]interface{}{
			"totalGames":      metrics.TotalGames,
			"totalInvestment": metrics.TotalInvestment,
			"totalWinnings":   metrics.TotalWinnings,
			"currentROI":      metrics.ROIPercentage,
			"winRate":         metrics.WinRate * 100,
			"biggestWin":      metrics.BiggestWin,
			"averageWin":      metrics.AverageWinAmount,
			"trend":           trend,
			"currentStreak":   currentStreak,
			"last30Days": map[string]interface{}{
				"games":      metrics.Last30Days.Games,
				"investment": metrics.Last30Days.Investment,
				"winnings":   metrics.Last30Days.Winnings,
				"roi":        metrics.Last30Days.ROI,
			},
			"performance": map[string]interface{}{
				"level":       getPerformanceLevel(metrics.ROIPercentage),
				"description": getPerformanceDescription(metrics.ROIPercentage, metrics.WinRate),
			},
		},
	}
}

// getPerformanceLevel retorna nível de performance baseado no ROI
func getPerformanceLevel(roi float64) string {
	if roi > 20 {
		return "Excelente"
	} else if roi > 0 {
		return "Boa"
	} else if roi > -20 {
		return "Regular"
	} else {
		return "Baixa"
	}
}

// getPerformanceDescription retorna descrição da performance
func getPerformanceDescription(roi float64, winRate float64) string {
	if roi > 10 && winRate > 0.3 {
		return "Estratégia muito eficaz com ROI positivo e boa taxa de acerto!"
	} else if roi > 0 {
		return "Performance positiva! Continue com a estratégia atual."
	} else if roi > -10 {
		return "Performance neutra. Considere ajustes na estratégia."
	} else {
		return "Performance abaixo do esperado. Recomenda-se revisão da estratégia."
	}
}

// ===============================
// NOTIFICAÇÕES - V2.0.0
// ===============================

// GetNotifications retorna notificações do usuário
func (a *App) GetNotifications(limit int, onlyUnread bool) map[string]interface{} {
	if notifications.GlobalNotificationManager == nil {
		return map[string]interface{}{
			"success":       false,
			"error":         "Sistema de notificações não inicializado",
			"notifications": []interface{}{},
			"total":         0,
		}
	}

	notificationsList := notifications.GlobalNotificationManager.GetNotifications(limit, onlyUnread)

	return map[string]interface{}{
		"success":       true,
		"notifications": notificationsList,
		"total":         len(notificationsList),
	}
}

// MarkNotificationAsRead marca notificação como lida
func (a *App) MarkNotificationAsRead(notificationID string) map[string]interface{} {
	if notifications.GlobalNotificationManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Sistema de notificações não inicializado",
		}
	}

	err := notifications.GlobalNotificationManager.MarkAsRead(notificationID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao marcar notificação: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": "Notificação marcada como lida",
	}
}

// ClearOldNotifications limpa notificações antigas
func (a *App) ClearOldNotifications(daysOld int) map[string]interface{} {
	if notifications.GlobalNotificationManager == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Sistema de notificações não inicializado",
			"cleared": 0,
		}
	}

	duration := time.Duration(daysOld) * 24 * time.Hour
	cleared := notifications.GlobalNotificationManager.ClearNotifications(duration)

	return map[string]interface{}{
		"success": true,
		"cleared": cleared,
		"message": fmt.Sprintf("Removidas %d notificações antigas", cleared),
	}
}

// GetAppInfo retorna informações do aplicativo
func (a *App) GetAppInfo() map[string]interface{} {
	return map[string]interface{}{
		"success":           true,
		"version":           version,
		"platform":          "windows",
		"repository":        "cccarv82/milhoes-desktop",
		"buildDate":         time.Now().Format("2006-01-02"),
		"autoUpdateEnabled": true,
		"features": []string{
			"Performance Analytics",
			"ROI Calculator",
			"Smart Notifications",
			"Historical Analysis",
			"Number Frequency Analysis",
		},
	}
}

// ===============================
// SISTEMA DE LOGGING EM ARQUIVO
// ===============================

// initCustomLogging inicializa o sistema de logging em arquivo
func initCustomLogging() error {
	// Determinar diretório de logs
	exePath, err := os.Executable()
	if err != nil {
		logDir = "logs"
		fmt.Printf("⚠️ Erro ao obter executável, usando diretório atual: %v\n", err)
	} else {
		logDir = filepath.Join(filepath.Dir(exePath), "logs")
		fmt.Printf("📁 Diretório do executável: %s\n", filepath.Dir(exePath))
	}

	fmt.Printf("📁 Diretório de logs determinado: %s\n", logDir)

	// Criar diretório de logs
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("❌ Erro ao criar diretório de logs: %v\n", err)
		return fmt.Errorf("erro ao criar diretório de logs: %v", err)
	}

	fmt.Printf("✅ Diretório de logs criado/existe: %s\n", logDir)

	// Nome do arquivo de log com data
	logFileName := fmt.Sprintf("lottery-optimizer-%s.log", time.Now().Format("2006-01-02"))
	logFilePath := filepath.Join(logDir, logFileName)

	fmt.Printf("📝 Tentando abrir arquivo de log: %s\n", logFilePath)

	// Abrir arquivo de log
	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("❌ Erro ao abrir arquivo de log: %v\n", err)
		return fmt.Errorf("erro ao abrir arquivo de log: %v", err)
	}

	fmt.Printf("✅ Arquivo de log aberto com sucesso\n")

	// Criar logger personalizado
	customLogger = &CustomLogger{file: logFile}

	// Log inicial
	customLogger.Printf("🚀 =================================")
	customLogger.Printf("🚀 LOTTERY OPTIMIZER %s INICIADO", version)
	customLogger.Printf("🚀 VERSÃO 2.0.0 - ANALYTICS DASHBOARD")
	customLogger.Printf("🚀 =================================")
	customLogger.Printf("📁 Diretório de logs: %s", logDir)
	customLogger.Printf("📝 Arquivo de log: %s", logFilePath)

	fmt.Printf("✅ Logs iniciais escritos e sincronizados\n")

	return nil
}

// flushLogs força a sincronização dos logs para o disco
func flushLogs() {
	if customLogger != nil && customLogger.file != nil {
		customLogger.file.Sync()
	}
}

// closeFileLogging fecha o arquivo de log
func closeFileLogging() {
	if customLogger != nil {
		customLogger.Close()
	}
}

// loadExistingConfig carrega configuração existente na inicialização
func loadExistingConfig() {
	customLogger.Printf("🔧 CARREGANDO CONFIGURAÇÃO EXISTENTE NA INICIALIZAÇÃO...")

	configPath, err := getConfigPath()
	if err != nil {
		customLogger.Printf("⚠️ Erro ao determinar caminho da configuração: %v", err)
		return
	}

	customLogger.Printf("📁 Verificando configuração em: %s", configPath)

	// Verificar se arquivo existe
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		customLogger.Printf("📝 Arquivo de configuração não existe - primeira execução")
		return
	}

	customLogger.Printf("✅ Arquivo de configuração encontrado, carregando...")

	// Ler arquivo
	data, err := os.ReadFile(configPath)
	if err != nil {
		customLogger.Printf("❌ Erro ao ler arquivo de configuração: %v", err)
		return
	}

	customLogger.Printf("📖 Arquivo lido (%d bytes)", len(data))

	// Parse YAML
	var configStruct struct {
		App struct {
			Verbose bool `yaml:"verbose"`
		} `yaml:"app"`
		Claude struct {
			APIKey     string `yaml:"api_key"`
			Model      string `yaml:"model"`
			MaxTokens  int    `yaml:"max_tokens"`
			TimeoutSec int    `yaml:"timeout_sec"`
		} `yaml:"claude"`
	}

	if err := yaml.Unmarshal(data, &configStruct); err != nil {
		customLogger.Printf("❌ Erro ao fazer parse do YAML: %v", err)
		return
	}

	// Atualizar configuração global se a chave estiver definida
	if configStruct.Claude.APIKey != "" {
		config.GlobalConfig.Claude.APIKey = configStruct.Claude.APIKey
		config.GlobalConfig.Claude.Model = configStruct.Claude.Model
		config.GlobalConfig.Claude.MaxTokens = configStruct.Claude.MaxTokens
		config.GlobalConfig.Claude.TimeoutSec = configStruct.Claude.TimeoutSec

		customLogger.Printf("✅ CONFIGURAÇÃO CARREGADA: APIKey length=%d, Model=%s, MaxTokens=%d",
			len(configStruct.Claude.APIKey), configStruct.Claude.Model, configStruct.Claude.MaxTokens)
	} else {
		customLogger.Printf("⚠️ Arquivo de configuração existe mas não contém chave Claude API")
	}
}

// ===============================
// VERIFICAÇÃO DE RESULTADOS
// ===============================

// CheckGameResult verifica o resultado de um jogo específico
func (a *App) CheckGameResult(gameID string) map[string]interface{} {
	logs.LogDatabase("🎯 Verificando resultado do jogo %s", gameID)

	if a.savedGamesDB == nil || a.resultChecker == nil {
		logs.LogError(logs.CategoryDatabase, "❌ Sistema de verificação não disponível")
		return map[string]interface{}{
			"success": false,
			"error":   "Sistema de verificação não disponível",
		}
	}

	// Buscar todos os jogos e filtrar por ID
	filter := models.SavedGamesFilter{}
	games, err := a.savedGamesDB.GetSavedGames(filter)
	if err != nil {
		logs.LogError(logs.CategoryDatabase, "❌ Erro ao buscar jogos: %v", err)
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao buscar jogos: %v", err),
		}
	}

	// Filtrar pelo ID específico
	var game models.SavedGame
	found := false
	for _, g := range games {
		if g.ID == gameID {
			game = g
			found = true
			break
		}
	}

	if !found {
		logs.LogError(logs.CategoryDatabase, "❌ Jogo não encontrado: %s", gameID)
		return map[string]interface{}{
			"success": false,
			"error":   "Jogo não encontrado",
		}
	}

	// Verificar se já foi checado
	if game.Status == "checked" {
		logs.LogDatabase("ℹ️ Jogo já foi verificado anteriormente")
		return map[string]interface{}{
			"success": true,
			"message": "Jogo já foi verificado",
			"result":  game.Result,
		}
	}

	logs.LogDatabase("🔍 Verificando resultado para: %s - Concurso %d", game.LotteryType, game.ContestNumber)

	// Verificar resultado
	result, err := a.resultChecker.CheckGameResult(game)
	if err != nil {
		logs.LogError(logs.CategoryDatabase, "❌ Erro ao verificar resultado: %v", err)
		// Marcar como erro
		a.savedGamesDB.UpdateGameStatus(gameID, "error")
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao verificar resultado: %v", err),
		}
	}

	if result == nil {
		logs.LogDatabase("⏳ Sorteio ainda não aconteceu")
		return map[string]interface{}{
			"success": true,
			"message": "Sorteio ainda não aconteceu",
			"result":  nil,
		}
	}

	logs.LogDatabase("✅ Resultado encontrado: %d acertos, prêmio: %s", result.HitCount, result.Prize)

	// Persistir o resultado no banco de dados
	err = a.savedGamesDB.UpdateGameResult(gameID, result)
	if err != nil {
		logs.LogError(logs.CategoryDatabase, "❌ Erro ao salvar resultado no banco: %v", err)
		// Marcar como erro mas retornar o resultado mesmo assim
		a.savedGamesDB.UpdateGameStatus(gameID, "error")
		return map[string]interface{}{
			"success": true,
			"result":  result,
			"message": "Resultado verificado, mas houve erro ao salvar no banco",
			"warning": fmt.Sprintf("Erro ao salvar: %v", err),
		}
	}

	logs.LogDatabase("🎉 Resultado verificado e salvo com sucesso!")

	return map[string]interface{}{
		"success": true,
		"result":  result,
		"message": "Resultado verificado com sucesso",
	}
}

// CheckAllPendingResults verifica todos os jogos pendentes
func (a *App) CheckAllPendingResults() map[string]interface{} {
	logs.LogDatabase("🔄 Iniciando verificação de todos os jogos pendentes")

	if a.savedGamesDB == nil || a.resultChecker == nil {
		logs.LogError(logs.CategoryDatabase, "❌ Sistema de verificação não disponível")
		return map[string]interface{}{
			"success": false,
			"error":   "Sistema de verificação não disponível",
		}
	}

	// Buscar jogos pendentes
	filter := models.SavedGamesFilter{Status: "pending"}
	games, err := a.savedGamesDB.GetSavedGames(filter)
	if err != nil {
		logs.LogError(logs.CategoryDatabase, "❌ Erro ao buscar jogos pendentes: %v", err)
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao buscar jogos pendentes: %v", err),
		}
	}

	if len(games) == 0 {
		logs.LogDatabase("ℹ️ Nenhum jogo pendente para verificar")
		return map[string]interface{}{
			"success": true,
			"message": "Nenhum jogo pendente para verificar",
			"checked": 0,
		}
	}

	logs.LogDatabase("📊 Encontrados %d jogos pendentes para verificar", len(games))

	checked := 0
	errors := []string{}
	sorteiosNaoAconteceram := 0

	for _, game := range games {
		logs.LogDatabase("🎯 Verificando jogo %s (%s - Concurso %d)", game.ID, game.LotteryType, game.ContestNumber)

		result, err := a.resultChecker.CheckGameResult(game)
		if err != nil {
			errorMsg := fmt.Sprintf("Jogo %s: %v", game.ID, err)
			logs.LogError(logs.CategoryDatabase, "❌ %s", errorMsg)
			errors = append(errors, errorMsg)
			a.savedGamesDB.UpdateGameStatus(game.ID, "error")
			continue
		}

		if result == nil {
			// Sorteio ainda não aconteceu
			logs.LogDatabase("⏳ Jogo %s: sorteio ainda não aconteceu", game.ID)
			sorteiosNaoAconteceram++
			continue
		}

		// Persistir o resultado no banco
		err = a.savedGamesDB.UpdateGameResult(game.ID, result)
		if err != nil {
			errorMsg := fmt.Sprintf("Jogo %s: erro ao salvar resultado - %v", game.ID, err)
			logs.LogError(logs.CategoryDatabase, "❌ %s", errorMsg)
			errors = append(errors, errorMsg)
			a.savedGamesDB.UpdateGameStatus(game.ID, "error")
			continue
		}

		logs.LogDatabase("✅ Jogo %s verificado: %d acertos, prêmio: %s", game.ID, result.HitCount, result.Prize)
		checked++
	}

	logs.LogDatabase("🎉 Verificação concluída: %d verificados, %d sorteios pendentes, %d erros", checked, sorteiosNaoAconteceram, len(errors))

	result := map[string]interface{}{
		"success": true,
		"checked": checked,
		"total":   len(games),
		"pending": sorteiosNaoAconteceram,
		"message": fmt.Sprintf("Verificados %d de %d jogos (%d ainda pendentes)", checked, len(games), sorteiosNaoAconteceram),
	}

	if len(errors) > 0 {
		result["errors"] = errors
		result["error_count"] = len(errors)
	}

	return result
}

// ===============================
// PREDITOR DE CONCURSOS QUENTES - FASE 1
// ===============================

// GetContestTemperatureAnalysis retorna análise de temperatura de todos os concursos
func (a *App) GetContestTemperatureAnalysis() map[string]interface{} {
	customLogger.Printf("🌡️ Frontend solicitou análise de temperatura dos concursos")

	if a.contestPredictor == nil {
		customLogger.Printf("❌ Preditor não inicializado")
		return map[string]interface{}{
			"success": false,
			"error":   "Preditor de concursos não disponível",
		}
	}

	summary, err := a.contestPredictor.GetTemperatureAnalysis()
	if err != nil {
		customLogger.Printf("❌ Erro ao obter análise de temperatura: %v", err)
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao obter análise: %v", err),
		}
	}

	customLogger.Printf("✅ Análise de temperatura obtida: %s mais quente (%d%% confiança)",
		summary.HottestLottery, int(summary.OverallConfidence))

	return map[string]interface{}{
		"success": true,
		"data":    summary,
	}
}

// GetPredictorMetrics retorna métricas de performance do preditor
func (a *App) GetPredictorMetrics() map[string]interface{} {
	customLogger.Printf("📊 Frontend solicitou métricas do preditor")

	if a.contestPredictor == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Preditor de concursos não disponível",
		}
	}

	metrics, err := a.contestPredictor.GetPredictorMetrics()
	if err != nil {
		customLogger.Printf("❌ Erro ao obter métricas: %v", err)
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao obter métricas: %v", err),
		}
	}

	customLogger.Printf("✅ Métricas obtidas: %.1f%% precisão", metrics.AccuracyPercentage)

	return map[string]interface{}{
		"success": true,
		"data":    metrics,
	}
}

// ===============================
// FUNÇÕES DE OTIMIZAÇÃO DE ORÇAMENTO
// ===============================

// optimizeBudgetUsage implementa algoritmo de maximização de uso do orçamento
func optimizeBudgetUsage(games []lottery.Game, budget float64) []lottery.Game {
	if len(games) == 0 {
		return games
	}

	// Implementar algoritmo de mochila (knapsack) simplificado
	// Ordenar jogos por valor/custo (eficiência)
	gamesCopy := make([]lottery.Game, len(games))
	copy(gamesCopy, games)

	// Calcular eficiência de cada jogo (números por real)
	type gameWithEfficiency struct {
		game       lottery.Game
		efficiency float64
	}

	gamesWithEff := make([]gameWithEfficiency, len(gamesCopy))
	for i, game := range gamesCopy {
		efficiency := float64(len(game.Numbers)) / game.Cost
		gamesWithEff[i] = gameWithEfficiency{game: game, efficiency: efficiency}
	}

	// Ordenar por eficiência decrescente
	sort.Slice(gamesWithEff, func(i, j int) bool {
		return gamesWithEff[i].efficiency > gamesWithEff[j].efficiency
	})

	// Selecionar jogos que maximizam o uso do orçamento
	var selectedGames []lottery.Game
	currentCost := 0.0

	// Primeira passada: pegar jogos mais eficientes
	for _, gameEff := range gamesWithEff {
		if currentCost+gameEff.game.Cost <= budget {
			selectedGames = append(selectedGames, gameEff.game)
			currentCost += gameEff.game.Cost
		}
	}

	// Segunda passada: tentar trocar jogos para usar mais orçamento
	remainingBudget := budget - currentCost
	if remainingBudget >= 3.0 {
		// Tentar substituir jogos baratos por mais caros se possível
		for i := len(selectedGames) - 1; i >= 0; i-- {
			currentGame := selectedGames[i]
			availableBudget := remainingBudget + currentGame.Cost

			// Procurar um jogo mais caro que caiba no orçamento disponível
			for _, gameEff := range gamesWithEff {
				if gameEff.game.Cost > currentGame.Cost && gameEff.game.Cost <= availableBudget {
					// Verificar se este jogo já não está selecionado
					alreadySelected := false
					for _, selected := range selectedGames {
						if gameEff.game.Type == selected.Type &&
							len(gameEff.game.Numbers) == len(selected.Numbers) &&
							gameEff.game.Cost == selected.Cost {
							// Comparar números para ver se é o mesmo jogo
							same := true
							for j, num := range gameEff.game.Numbers {
								if j >= len(selected.Numbers) || num != selected.Numbers[j] {
									same = false
									break
								}
							}
							if same {
								alreadySelected = true
								break
							}
						}
					}

					if !alreadySelected {
						// Substituir o jogo
						selectedGames[i] = gameEff.game
						remainingBudget = availableBudget - gameEff.game.Cost
						break
					}
				}
			}
		}
	}

	return selectedGames
}
