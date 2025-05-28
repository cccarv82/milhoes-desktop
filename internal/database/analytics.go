package database

import (
	"fmt"
	"lottery-optimizer-gui/internal/models"
)

// Global database instance for analytics
var globalDB *SavedGamesDB

// SetGlobalDB define a instância global do database para analytics
func SetGlobalDB(db *SavedGamesDB) {
	globalDB = db
}

// GetAllSavedGames busca todos os jogos salvos (função global para analytics)
func GetAllSavedGames() ([]models.SavedGame, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database não inicializado")
	}
	return globalDB.GetAllSavedGames()
}

// GetGamesByLottery busca jogos por tipo de loteria (função global para analytics)
func GetGamesByLottery(lotteryType string) ([]models.SavedGame, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("database não inicializado")
	}
	return globalDB.GetGamesByLottery(lotteryType)
}

// SavedGame alias for analytics compatibility
type SavedGame = models.SavedGame
