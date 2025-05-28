package logs

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// LogCategory define os tipos de logs especializados
type LogCategory string

const (
	CategoryMain      LogCategory = "main"
	CategoryAI        LogCategory = "ai"
	CategoryData      LogCategory = "data"
	CategoryUpdater   LogCategory = "updater"
	CategoryDatabase  LogCategory = "database"
	CategoryLauncher  LogCategory = "launcher"
	CategoryAnalytics LogCategory = "analytics"
)

// SpecializedLogger logger especializado com categorias
type SpecializedLogger struct {
	loggers map[LogCategory]*log.Logger
	logDir  string
}

var GlobalLogger *SpecializedLogger

// Init inicializa o sistema de logs especializado
func Init() error {
	// Obter diretório do executável
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("erro ao obter caminho do executável: %w", err)
	}

	// Criar diretório de logs
	logDir := filepath.Join(filepath.Dir(exePath), "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório de logs: %w", err)
	}

	GlobalLogger = &SpecializedLogger{
		loggers: make(map[LogCategory]*log.Logger),
		logDir:  logDir,
	}

	// Inicializar loggers para cada categoria
	categories := []LogCategory{
		CategoryMain,
		CategoryAI,
		CategoryData,
		CategoryUpdater,
		CategoryDatabase,
		CategoryLauncher,
		CategoryAnalytics,
	}

	for _, category := range categories {
		if err := GlobalLogger.initCategoryLogger(category); err != nil {
			return fmt.Errorf("erro ao inicializar logger %s: %w", category, err)
		}
	}

	// Log inicial
	LogMain("🚀 Sistema de logs especializado inicializado")
	LogMain("📁 Diretório de logs: %s", logDir)

	return nil
}

// initCategoryLogger inicializa um logger para uma categoria específica
func (sl *SpecializedLogger) initCategoryLogger(category LogCategory) error {
	// Nome do arquivo com data atual
	today := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("lottery-optimizer-%s-%s.log", category, today)
	filepath := filepath.Join(sl.logDir, filename)

	// Abrir arquivo (criar se não existir, append se existir)
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("erro ao abrir arquivo de log %s: %w", filepath, err)
	}

	// Criar logger
	logger := log.New(file, "", log.LstdFlags|log.Lmicroseconds)
	sl.loggers[category] = logger

	return nil
}

// getLogger retorna o logger para uma categoria
func (sl *SpecializedLogger) getLogger(category LogCategory) *log.Logger {
	if logger, exists := sl.loggers[category]; exists {
		return logger
	}
	// Fallback para stderr se não encontrar
	return log.New(os.Stderr, fmt.Sprintf("[%s] ", category), log.LstdFlags)
}

// Funções de conveniência para cada categoria
func LogMain(format string, v ...interface{}) {
	if GlobalLogger != nil {
		msg := fmt.Sprintf(format, v...)
		GlobalLogger.getLogger(CategoryMain).Printf("[MAIN] %s", msg)
	}
}

func LogAI(format string, v ...interface{}) {
	if GlobalLogger != nil {
		msg := fmt.Sprintf(format, v...)
		GlobalLogger.getLogger(CategoryAI).Printf("[AI] %s", msg)
	}
}

func LogData(format string, v ...interface{}) {
	if GlobalLogger != nil {
		msg := fmt.Sprintf(format, v...)
		GlobalLogger.getLogger(CategoryData).Printf("[DATA] %s", msg)
	}
}

func LogUpdater(format string, v ...interface{}) {
	if GlobalLogger != nil {
		msg := fmt.Sprintf(format, v...)
		GlobalLogger.getLogger(CategoryUpdater).Printf("[UPDATER] %s", msg)
	}
}

func LogDatabase(format string, v ...interface{}) {
	if GlobalLogger != nil {
		msg := fmt.Sprintf(format, v...)
		GlobalLogger.getLogger(CategoryDatabase).Printf("[DATABASE] %s", msg)
	}
}

func LogLauncher(format string, v ...interface{}) {
	if GlobalLogger != nil {
		msg := fmt.Sprintf(format, v...)
		GlobalLogger.getLogger(CategoryLauncher).Printf("[LAUNCHER] %s", msg)
	}
}

func LogAnalytics(format string, v ...interface{}) {
	if GlobalLogger != nil {
		msg := fmt.Sprintf(format, v...)
		GlobalLogger.getLogger(CategoryAnalytics).Printf("[ANALYTICS] %s", msg)
	}
}

// LogError registra erro em qualquer categoria
func LogError(category LogCategory, format string, v ...interface{}) {
	if GlobalLogger != nil {
		msg := fmt.Sprintf(format, v...)
		GlobalLogger.getLogger(category).Printf("[ERROR] %s", msg)
	}
}

// LogDebug registra debug em qualquer categoria (apenas se verbose ativado)
func LogDebug(category LogCategory, format string, v ...interface{}) {
	if GlobalLogger != nil {
		msg := fmt.Sprintf(format, v...)
		GlobalLogger.getLogger(category).Printf("[DEBUG] %s", msg)
	}
}

// GetLogDir retorna o diretório de logs
func GetLogDir() string {
	if GlobalLogger != nil {
		return GlobalLogger.logDir
	}
	return ""
}

// ListLogFiles retorna lista de arquivos de log existentes
func ListLogFiles() ([]string, error) {
	if GlobalLogger == nil {
		return nil, fmt.Errorf("sistema de logs não inicializado")
	}

	files, err := os.ReadDir(GlobalLogger.logDir)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar arquivos de log: %w", err)
	}

	var logFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".log" {
			logFiles = append(logFiles, file.Name())
		}
	}

	return logFiles, nil
}
