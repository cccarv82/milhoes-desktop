package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config estrutura de configuração da aplicação
type Config struct {
	Claude ClaudeConfig `yaml:"claude"`
	App    AppConfig    `yaml:"app"`
}

// ClaudeConfig configurações da API do Claude
type ClaudeConfig struct {
	APIKey     string `yaml:"api_key"`
	Model      string `yaml:"model"`
	MaxTokens  int    `yaml:"max_tokens"`
	TimeoutSec int    `yaml:"timeout_sec"`
}

// AppConfig configurações da aplicação
type AppConfig struct {
	CacheEnabled  bool   `yaml:"cache_enabled"`
	CacheDuration int    `yaml:"cache_duration_hours"`
	DefaultBudget int    `yaml:"default_budget"`
	LogLevel      string `yaml:"log_level"`
	DataSourceURL string `yaml:"data_source_url"`
}

var GlobalConfig *Config

// getConfigPath retorna o caminho do arquivo de configuração no diretório do executável
func getConfigPath() string {
	// Obter o diretório do executável
	exePath, err := os.Executable()
	if err != nil {
		// Fallback para diretório atual
		return "lottery-optimizer.yaml"
	}

	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "lottery-optimizer.yaml")
}

// Init inicializa a configuração global
func Init() {
	// Configurar Viper para ler do arquivo de configuração no diretório do executável
	configPath := getConfigPath()
	viper.SetConfigFile(configPath)

	// Tentar ler o arquivo de configuração
	if err := viper.ReadInConfig(); err != nil {
		// Se não conseguir ler, não é erro fatal - usar padrões
		fmt.Printf("Aviso: Não foi possível ler arquivo de configuração (%s): %v\n", configPath, err)
	}

	GlobalConfig = &Config{
		Claude: ClaudeConfig{
			APIKey:     getClaudeAPIKey(),
			Model:      viper.GetString("claude.model"),
			MaxTokens:  viper.GetInt("claude.max_tokens"),
			TimeoutSec: viper.GetInt("claude.timeout_sec"),
		},
		App: AppConfig{
			CacheEnabled:  viper.GetBool("app.cache_enabled"),
			CacheDuration: viper.GetInt("app.cache_duration_hours"),
			DefaultBudget: viper.GetInt("app.default_budget"),
			LogLevel:      viper.GetString("app.log_level"),
			DataSourceURL: viper.GetString("app.data_source_url"),
		},
	}

	// Configurações padrão
	setDefaults()
}

// getClaudeAPIKey obtém a chave da API do Claude
func getClaudeAPIKey() string {
	// Prioridade: flag -> env var -> config file
	if key := viper.GetString("api-key"); key != "" {
		return key
	}

	if key := os.Getenv("CLAUDE_API_KEY"); key != "" {
		return key
	}

	if key := viper.GetString("claude.api_key"); key != "" {
		return key
	}

	// Retornar string vazia - usuário deve fornecer sua própria chave
	return ""
}

// setDefaults define valores padrão para configurações
func setDefaults() {
	if GlobalConfig.Claude.Model == "" {
		GlobalConfig.Claude.Model = "claude-3-5-sonnet-20241022"
	}

	if GlobalConfig.Claude.MaxTokens == 0 {
		GlobalConfig.Claude.MaxTokens = 4000
	}

	if GlobalConfig.Claude.TimeoutSec == 0 {
		GlobalConfig.Claude.TimeoutSec = 30
	}

	if GlobalConfig.App.CacheDuration == 0 {
		GlobalConfig.App.CacheDuration = 24 // 24 horas
	}

	if GlobalConfig.App.DefaultBudget == 0 {
		GlobalConfig.App.DefaultBudget = 50 // R$ 50
	}

	if GlobalConfig.App.LogLevel == "" {
		GlobalConfig.App.LogLevel = "info"
	}

	if GlobalConfig.App.DataSourceURL == "" {
		GlobalConfig.App.DataSourceURL = "https://servicebus2.caixa.gov.br/portaldeloterias/api"
	}

	GlobalConfig.App.CacheEnabled = true
}

// ValidateConfig valida se a configuração está correta
func ValidateConfig() error {
	if GlobalConfig.Claude.APIKey == "" {
		return fmt.Errorf(`chave da API do Claude não configurada!

Para usar as funcionalidades de IA, configure sua chave da Claude:

💡 OPÇÕES DE CONFIGURAÇÃO:
   1. Variável de ambiente: export CLAUDE_API_KEY="sua-chave-aqui"
   2. Parâmetro da linha de comando: --api-key="sua-chave-aqui"
   3. Arquivo de configuração %s

🔑 OBTENHA SUA CHAVE:
   Visite: https://console.anthropic.com/
   
⚠️  SEM CHAVE: O app funcionará apenas com estratégias básicas (sem IA)`, getConfigPath())
	}

	if GlobalConfig.App.DefaultBudget <= 0 {
		return fmt.Errorf("orçamento padrão deve ser maior que zero")
	}

	return nil
}

// GetClaudeAPIKey retorna a chave da API do Claude
func GetClaudeAPIKey() string {
	// CORREÇÃO: Sempre tentar ler do arquivo primeiro para pegar configuração mais recente
	configPath := getConfigPath()
	if content, err := os.ReadFile(configPath); err == nil {
		// Parse rápido do YAML só para pegar a chave
		var configYAML struct {
			Claude struct {
				APIKey string `yaml:"api_key"`
			} `yaml:"claude"`
		}
		
		if err := yaml.Unmarshal(content, &configYAML); err == nil && configYAML.Claude.APIKey != "" {
			return configYAML.Claude.APIKey
		}
	}
	
	// Fallback para GlobalConfig se arquivo não disponível
	if GlobalConfig != nil {
		return GlobalConfig.Claude.APIKey
	}
	
	return ""
}

// GetClaudeModel retorna o modelo do Claude a ser usado
func GetClaudeModel() string {
	return GlobalConfig.Claude.Model
}

// GetMaxTokens retorna o número máximo de tokens para o Claude
func GetMaxTokens() int {
	return GlobalConfig.Claude.MaxTokens
}

// IsVerbose retorna se o modo verbose está ativo
func IsVerbose() bool {
	return viper.GetBool("verbose")
}
