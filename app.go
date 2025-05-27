package main

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
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
	"sort"
	"strings"
	"time"
)

var (
	githubRepo = "cccarv82/milhoes-releases" // Repositório público para releases
	logFile *os.File
	logDir  string
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
	ctx           context.Context
	dataClient    *data.Client
	aiClient      *ai.ClaudeClient
	updater       *updater.Updater
	savedGamesDB  *database.SavedGamesDB
	resultChecker *services.ResultChecker
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

	// CARREGAR CONFIGURAÇÃO EXISTENTE NA INICIALIZAÇÃO
	loadExistingConfig()

	dataClient := data.NewClient()

	// Inicializar banco de dados de jogos salvos
	// Usar diretório absoluto baseado no executável
	execPath, err := os.Executable()
	if err != nil {
		customLogger.Printf("Erro ao obter caminho do executável: %v", err)
		execPath, _ = os.Getwd() // Fallback para diretório atual
	}

	dataDir := filepath.Join(filepath.Dir(execPath), "data")
	dbPath := filepath.Join(dataDir, "saved_games.db")

	// Criar diretório se não existir com permissões adequadas
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		customLogger.Printf("❌ Erro ao criar diretório de dados (%s): %v", dataDir, err)
	}

	customLogger.Printf("📁 Inicializando banco de dados em: %s", dbPath)

	savedGamesDB, err := database.NewSavedGamesDB(dbPath)
	if err != nil {
		customLogger.Printf("❌ ERRO ao inicializar banco de jogos salvos: %v", err)
		customLogger.Printf("   📂 Diretório: %s", dataDir)
		customLogger.Printf("   💾 Arquivo DB: %s", dbPath)

		// Verificar se o diretório existe
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			customLogger.Printf("   ⚠️  Diretório não existe: %s", dataDir)
		} else {
			customLogger.Printf("   ✅ Diretório existe: %s", dataDir)
		}

		// Verificar permissões
		if file, err := os.OpenFile(filepath.Join(dataDir, "test_write.tmp"), os.O_CREATE|os.O_WRONLY, 0644); err != nil {
			customLogger.Printf("   ❌ Sem permissão de escrita no diretório: %v", err)
		} else {
			file.Close()
			os.Remove(filepath.Join(dataDir, "test_write.tmp"))
			customLogger.Printf("   ✅ Permissão de escrita OK")
		}

		savedGamesDB = nil // Garantir que seja nil em caso de erro
	} else {
		customLogger.Printf("✅ Banco de jogos salvos inicializado com sucesso!")
	}

	// Inicializar verificador de resultados usando o dataClient existente
	var resultChecker *services.ResultChecker
	if savedGamesDB != nil {
		resultChecker = services.NewResultChecker(dataClient, savedGamesDB)
		// Iniciar verificação automática
		resultChecker.ScheduleAutoCheck()
		customLogger.Printf("✅ Verificador de resultados inicializado e agendado!")
	} else {
		customLogger.Printf("⚠️  Verificador de resultados não inicializado (banco indisponível)")
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
		customLogger.Printf("🔄 Verificando atualizações na inicialização...")
		updateInfo, err := a.CheckForUpdates()
		if err != nil {
			customLogger.Printf("❌ Erro ao verificar atualizações: %v", err)
		} else if updateInfo != nil && updateInfo.Available {
			customLogger.Printf("🎉 Nova versão disponível: %s -> %s", version, updateInfo.Version)
		} else {
			customLogger.Printf("✅ App atualizado - versão mais recente já instalada")
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
		customLogger.Printf("🎯 IA gerou %d jogos com custo total R$ %.2f", len(response.Strategy.Games), response.Strategy.TotalCost)
		for i, game := range response.Strategy.Games {
			customLogger.Printf("   Jogo %d: %s - %v - R$ %.2f", i+1, game.Type, game.Numbers, game.Cost)
		}
	}

	// TEMPORÁRIO: Pular validação para debug - usar estratégia da IA diretamente
	validatedStrategy := &response.Strategy

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
func (a *App) GetCurrentConfig() map[string]interface{} {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000000")
	customLogger.Printf("📖 [%s] GetCurrentConfig INICIADO", timestamp)
	
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
			"exists":        false,
			"claudeApiKey":  "",
			"claudeModel":   "claude-3-sonnet-20240229",
			"maxTokens":     4096,
			"timeoutSec":    60,
			"verbose":       false,
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
	
	customLogger.Printf("📝 [%s] GetCurrentConfig: Arquivo lido (%d bytes):\n%s", timestamp, len(data), string(data))
	
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
	
	result := map[string]interface{}{
		"exists":       true,
		"claudeApiKey": configStruct.Claude.APIKey,
		"claudeModel":  configStruct.Claude.Model,
		"maxTokens":    configStruct.Claude.MaxTokens,
		"timeoutSec":   configStruct.Claude.TimeoutSec,
		"verbose":      configStruct.App.Verbose,
		"debug": map[string]interface{}{
			"configPath": configPath,
			"fileSize":   len(data),
			"apiKeyLen":  len(configStruct.Claude.APIKey),
		},
	}
	
	customLogger.Printf("✅ [%s] GetCurrentConfig: Retornando configuração - APIKey presente: %t", 
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
	
	customLogger.Printf("📝 [%s] YAML gerado (%d bytes):\n%s", timestamp, len(yamlData), string(yamlData))
	
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
	
	// Verificar se arquivo foi realmente salvo lendo de volta
	if savedContent, err := os.ReadFile(configPath); err != nil {
		customLogger.Printf("❌ [%s] Erro ao verificar arquivo salvo: %v", timestamp, err)
		flushLogs()
		return map[string]interface{}{
			"success": false,
			"error":   "Erro ao verificar arquivo salvo: " + err.Error(),
		}
	} else {
		customLogger.Printf("✅ [%s] Verificação: arquivo contém %d bytes", timestamp, len(savedContent))
		
		// Parse de volta para verificar
		var verifyStruct struct {
			Claude struct {
				APIKey string `yaml:"api_key"`
			} `yaml:"claude"`
		}
		
		if err := yaml.Unmarshal(savedContent, &verifyStruct); err != nil {
			customLogger.Printf("❌ [%s] Erro ao verificar YAML salvo: %v", timestamp, err)
		} else {
			customLogger.Printf("✅ [%s] Verificação: chave salva tem %d caracteres", timestamp, len(verifyStruct.Claude.APIKey))
		}
	}

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

// DebugConfigPath função para debug detalhado de caminhos e arquivos
func (a *App) DebugConfigPath() map[string]interface{} {
	result := map[string]interface{}{}

	// Caminho do executável
	exePath, err := os.Executable()
	if err != nil {
		result["executableError"] = err.Error()
		result["executablePath"] = "ERRO"
	} else {
		result["executablePath"] = exePath
		result["executableDir"] = filepath.Dir(exePath)
	}

	// Diretório de dados do usuário (APPDATA)
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		result["userConfigDirError"] = err.Error()
		result["userConfigDir"] = "ERRO"
	} else {
		result["userConfigDir"] = userConfigDir
		appDataDir := filepath.Join(userConfigDir, "lottery-optimizer")
		result["appDataDir"] = appDataDir
		
		// Verificar se diretório APPDATA existe
		if stat, err := os.Stat(appDataDir); err != nil {
			result["appDataDirExists"] = false
			result["appDataDirError"] = err.Error()
		} else {
			result["appDataDirExists"] = true
			result["appDataDirMode"] = stat.Mode().String()
		}
		
		// Testar permissões de escrita no APPDATA
		testFile := filepath.Join(appDataDir, "write_test.tmp")
		if err := os.MkdirAll(appDataDir, 0755); err != nil {
			result["appDataWritable"] = false
			result["appDataWriteError"] = err.Error()
		} else if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			result["appDataWritable"] = false
			result["appDataWriteError"] = err.Error()
		} else {
			result["appDataWritable"] = true
			os.Remove(testFile)
		}
	}

	// Caminho final resolvido
	configPath, err := getConfigPath()
	if err != nil {
		result["finalConfigPathError"] = err.Error()
		result["finalConfigPath"] = "ERRO"
	} else {
		result["finalConfigPath"] = configPath
		result["finalConfigDir"] = filepath.Dir(configPath)
	}

	// Verificar se arquivo final existe
	if configPath != "ERRO" {
		if stat, err := os.Stat(configPath); err != nil {
			result["configExists"] = false
			result["configError"] = err.Error()
		} else {
			result["configExists"] = true
			result["configSize"] = stat.Size()
			result["configModTime"] = stat.ModTime().Format("2006-01-02 15:04:05")
			result["configMode"] = stat.Mode().String()
		}

		// Tentar ler conteúdo
		if content, err := os.ReadFile(configPath); err != nil {
			result["readError"] = err.Error()
		} else {
			result["configContent"] = string(content)
			result["configLength"] = len(content)
		}

		// Testar permissões de escrita no diretório final
		configDir := filepath.Dir(configPath)
		if err := os.WriteFile(configPath+"_test", []byte("test"), 0644); err != nil {
			result["writePermissionError"] = err.Error()
			result["canWrite"] = false
		} else {
			result["canWrite"] = true
			os.Remove(configPath + "_test") // Limpar arquivo de teste
		}

		// Informações do diretório final
		if files, err := os.ReadDir(configDir); err != nil {
			result["dirListError"] = err.Error()
		} else {
			fileList := []string{}
			for _, file := range files {
				fileList = append(fileList, file.Name())
			}
			result["dirFiles"] = fileList
		}
	}

	// Estratégias testadas
	result["strategies"] = map[string]interface{}{
		"1_appdata":    result["appDataDir"],
		"2_executable": result["executableDir"],
		"3_current":    "lottery-optimizer.yaml",
		"final_chosen": result["finalConfigPath"],
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

	// Callback para progresso de download
	progressCallback := func(downloaded, total int64) {
		customLogger.Printf("Download: %d/%d bytes (%.2f%%)",
			downloaded, total, float64(downloaded)/float64(total)*100)
	}

	return a.updater.DownloadUpdate(ctx, updateInfo, progressCallback)
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
	go func() {
		ticker := time.NewTicker(24 * time.Hour) // Verificar a cada 24 horas
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				updateInfo, err := a.CheckForUpdates()
				if err != nil {
					customLogger.Printf("❌ Erro na verificação automática de updates: %v", err)
				} else if updateInfo != nil && updateInfo.Available {
					customLogger.Printf("🚀 NOVA VERSÃO DISPONÍVEL: %s -> %s", version, updateInfo.Version)
					customLogger.Printf("📦 Download: %s", updateInfo.DownloadURL)
				}
			}
		}
	}()
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
		"version":           version,
		"platform":          "windows",
		"repository":        "cccarv82/milhoes-desktop",
		"buildDate":         time.Now().Format("2006-01-02"),
		"autoUpdateEnabled": true,
		"logDirectory":      logDir,
	}
}

// ===============================
// MÉTODOS PARA GERENCIAR LOGS
// ===============================

// GetLogFiles retorna lista de arquivos de log disponíveis
func (a *App) GetLogFiles() map[string]interface{} {
	if logDir == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "Diretório de logs não inicializado",
		}
	}

	files, err := os.ReadDir(logDir)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao ler diretório de logs: %v", err),
		}
	}

	var logFiles []map[string]interface{}
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "lottery-optimizer-") && strings.HasSuffix(file.Name(), ".log") {
			info, err := file.Info()
			if err != nil {
				continue
			}

			logFiles = append(logFiles, map[string]interface{}{
				"name":     file.Name(),
				"size":     info.Size(),
				"modTime":  info.ModTime().Format("2006-01-02 15:04:05"),
				"path":     filepath.Join(logDir, file.Name()),
				"isToday":  file.Name() == fmt.Sprintf("lottery-optimizer-%s.log", time.Now().Format("2006-01-02")),
			})
		}
	}

	// Ordenar por data (mais recente primeiro)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i]["name"].(string) > logFiles[j]["name"].(string)
	})

	return map[string]interface{}{
		"success":   true,
		"logFiles":  logFiles,
		"logDir":    logDir,
		"totalFiles": len(logFiles),
	}
}

// GetLogContent retorna o conteúdo de um arquivo de log
func (a *App) GetLogContent(fileName string) map[string]interface{} {
	if logDir == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "Diretório de logs não inicializado",
		}
	}

	// Validar nome do arquivo por segurança
	if !strings.HasPrefix(fileName, "lottery-optimizer-") || !strings.HasSuffix(fileName, ".log") {
		return map[string]interface{}{
			"success": false,
			"error":   "Nome de arquivo inválido",
		}
	}

	filePath := filepath.Join(logDir, fileName)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao ler arquivo de log: %v", err),
		}
	}

	return map[string]interface{}{
		"success": true,
		"content": string(content),
		"size":    len(content),
		"file":    fileName,
	}
}

// GetTodayLogContent retorna o conteúdo do log de hoje
func (a *App) GetTodayLogContent() map[string]interface{} {
	todayFileName := fmt.Sprintf("lottery-optimizer-%s.log", time.Now().Format("2006-01-02"))
	return a.GetLogContent(todayFileName)
}

// OpenLogDirectory abre o diretório de logs no explorador
func (a *App) OpenLogDirectory() map[string]interface{} {
	if logDir == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "Diretório de logs não inicializado",
		}
	}

	// No Windows, usar o comando explorer
	// Nota: Esta função pode precisar de ajustes dependendo do sistema
	return map[string]interface{}{
		"success": true,
		"message": "Use o explorador de arquivos para navegar até: " + logDir,
		"path":    logDir,
	}
}

// ClearOldLogs remove logs antigos (mais de 7 dias)
func (a *App) ClearOldLogs() map[string]interface{} {
	if logDir == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "Diretório de logs não inicializado",
		}
	}

	files, err := os.ReadDir(logDir)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Erro ao ler diretório de logs: %v", err),
		}
	}

	var removedFiles []string
	cutoff := time.Now().AddDate(0, 0, -7) // 7 dias atrás

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "lottery-optimizer-") && strings.HasSuffix(file.Name(), ".log") {
			filePath := filepath.Join(logDir, file.Name())
			if info, err := file.Info(); err == nil {
				if info.ModTime().Before(cutoff) {
					if err := os.Remove(filePath); err == nil {
						removedFiles = append(removedFiles, file.Name())
						customLogger.Printf("🗑️ Log antigo removido: %s", file.Name())
					}
				}
			}
		}
	}

	return map[string]interface{}{
		"success":      true,
		"removedFiles": removedFiles,
		"totalRemoved": len(removedFiles),
		"message":      fmt.Sprintf("Removidos %d arquivos de log antigos", len(removedFiles)),
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

	// TESTE IMEDIATO - escrever logs para verificar
	fmt.Printf("🧪 Testando log no console...\n")
	
	// Log inicial
	customLogger.Printf("🚀 =================================")
	customLogger.Printf("🚀 LOTTERY OPTIMIZER %s INICIADO", version)
	customLogger.Printf("🚀 =================================")
	customLogger.Printf("📁 Diretório de logs: %s", logDir)
	customLogger.Printf("📝 Arquivo de log: %s", logFilePath)
	customLogger.Printf("🧪 TESTE DE LOGGING - Se você está vendo isso, o sistema funciona!")

	fmt.Printf("✅ Logs iniciais escritos e sincronizados\n")

	// Rotação de logs (manter últimos 7 dias)
	go rotateLogFiles()

	return nil
}

// rotateLogFiles remove logs antigos (manter últimos 7 dias)
func rotateLogFiles() {
	if logDir == "" {
		return
	}

	files, err := os.ReadDir(logDir)
	if err != nil {
		customLogger.Printf("❌ Erro ao ler diretório de logs: %v", err)
		return
	}

	cutoff := time.Now().AddDate(0, 0, -7) // 7 dias atrás

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "lottery-optimizer-") && strings.HasSuffix(file.Name(), ".log") {
			if info, err := file.Info(); err == nil && info.ModTime().Before(cutoff) {
				logPath := filepath.Join(logDir, file.Name())
				if err := os.Remove(logPath); err == nil {
					customLogger.Printf("🗑️ Log antigo removido: %s", file.Name())
				}
			}
		}
	}
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
