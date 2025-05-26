package database

import (
	"database/sql"
	"fmt"
	"time"

	"lottery-optimizer-gui/internal/models"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

type SavedGamesDB struct {
	db *sql.DB
}

// NewSavedGamesDB cria uma nova instância do banco de jogos salvos
func NewSavedGamesDB(dbPath string) (*SavedGamesDB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir banco de dados: %w", err)
	}

	sgdb := &SavedGamesDB{db: db}
	
	// Criar tabela se não existir
	if err := sgdb.createTables(); err != nil {
		return nil, fmt.Errorf("erro ao criar tabelas: %w", err)
	}

	return sgdb, nil
}

// createTables cria as tabelas necessárias
func (sg *SavedGamesDB) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS saved_games (
		id TEXT PRIMARY KEY,
		lottery_type TEXT NOT NULL,
		numbers TEXT NOT NULL,  -- JSON array de números
		expected_draw TEXT NOT NULL, -- Data esperada (YYYY-MM-DD)
		contest_number INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending', -- pending, checked, error
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		checked_at DATETIME NULL
	);

	CREATE INDEX IF NOT EXISTS idx_saved_games_lottery_type ON saved_games(lottery_type);
	CREATE INDEX IF NOT EXISTS idx_saved_games_status ON saved_games(status);
	CREATE INDEX IF NOT EXISTS idx_saved_games_expected_draw ON saved_games(expected_draw);
	`

	_, err := sg.db.Exec(query)
	return err
}

// SaveGame salva um novo jogo para verificação posterior
func (sg *SavedGamesDB) SaveGame(request models.SaveGameRequest) (*models.SavedGame, error) {
	game := &models.SavedGame{
		ID:            uuid.New().String(),
		LotteryType:   request.LotteryType,
		Numbers:       models.IntSlice(request.Numbers),
		ExpectedDraw:  request.ExpectedDraw,
		ContestNumber: request.ContestNumber,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	query := `
		INSERT INTO saved_games (id, lottery_type, numbers, expected_draw, contest_number, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := sg.db.Exec(query, 
		game.ID, 
		game.LotteryType, 
		game.Numbers, 
		game.ExpectedDraw, 
		game.ContestNumber, 
		game.Status, 
		game.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("erro ao salvar jogo: %w", err)
	}

	return game, nil
}

// GetSavedGames busca jogos salvos com filtros opcionais
func (sg *SavedGamesDB) GetSavedGames(filter models.SavedGamesFilter) ([]models.SavedGame, error) {
	query := "SELECT id, lottery_type, numbers, expected_draw, contest_number, status, created_at, checked_at FROM saved_games WHERE 1=1"
	args := []interface{}{}

	if filter.LotteryType != "" {
		query += " AND lottery_type = ?"
		args = append(args, filter.LotteryType)
	}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	if filter.FromDate != "" {
		query += " AND expected_draw >= ?"
		args = append(args, filter.FromDate)
	}

	if filter.ToDate != "" {
		query += " AND expected_draw <= ?"
		args = append(args, filter.ToDate)
	}

	query += " ORDER BY created_at DESC"

	rows, err := sg.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar jogos salvos: %w", err)
	}
	defer rows.Close()

	var games []models.SavedGame
	for rows.Next() {
		var game models.SavedGame
		var checkedAt sql.NullTime

		err := rows.Scan(
			&game.ID,
			&game.LotteryType,
			&game.Numbers,
			&game.ExpectedDraw,
			&game.ContestNumber,
			&game.Status,
			&game.CreatedAt,
			&checkedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao fazer scan do jogo: %w", err)
		}

		if checkedAt.Valid {
			game.CheckedAt = &checkedAt.Time
		}

		games = append(games, game)
	}

	return games, nil
}

// GetPendingGames busca jogos que ainda não foram verificados
func (sg *SavedGamesDB) GetPendingGames() ([]models.SavedGame, error) {
	filter := models.SavedGamesFilter{Status: "pending"}
	return sg.GetSavedGames(filter)
}

// UpdateGameStatus atualiza o status de um jogo
func (sg *SavedGamesDB) UpdateGameStatus(gameID string, status string) error {
	query := "UPDATE saved_games SET status = ?, checked_at = ? WHERE id = ?"
	_, err := sg.db.Exec(query, status, time.Now(), gameID)
	if err != nil {
		return fmt.Errorf("erro ao atualizar status do jogo: %w", err)
	}
	return nil
}

// DeleteGame remove um jogo salvo
func (sg *SavedGamesDB) DeleteGame(gameID string) error {
	query := "DELETE FROM saved_games WHERE id = ?"
	_, err := sg.db.Exec(query, gameID)
	if err != nil {
		return fmt.Errorf("erro ao deletar jogo: %w", err)
	}
	return nil
}

// GetGameByID busca um jogo específico pelo ID
func (sg *SavedGamesDB) GetGameByID(gameID string) (*models.SavedGame, error) {
	query := "SELECT id, lottery_type, numbers, expected_draw, contest_number, status, created_at, checked_at FROM saved_games WHERE id = ?"
	
	var game models.SavedGame
	var checkedAt sql.NullTime

	err := sg.db.QueryRow(query, gameID).Scan(
		&game.ID,
		&game.LotteryType,
		&game.Numbers,
		&game.ExpectedDraw,
		&game.ContestNumber,
		&game.Status,
		&game.CreatedAt,
		&checkedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("jogo não encontrado")
		}
		return nil, fmt.Errorf("erro ao buscar jogo: %w", err)
	}

	if checkedAt.Valid {
		game.CheckedAt = &checkedAt.Time
	}

	return &game, nil
}

// GetStats retorna estatísticas dos jogos salvos
func (sg *SavedGamesDB) GetStats() (map[string]interface{}, error) {
	query := `
		SELECT 
			lottery_type,
			status,
			COUNT(*) as count
		FROM saved_games 
		GROUP BY lottery_type, status
	`

	rows, err := sg.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar estatísticas: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]interface{})
	lotteryStats := make(map[string]map[string]int)

	for rows.Next() {
		var lotteryType, status string
		var count int

		err := rows.Scan(&lotteryType, &status, &count)
		if err != nil {
			return nil, fmt.Errorf("erro ao fazer scan das estatísticas: %w", err)
		}

		if lotteryStats[lotteryType] == nil {
			lotteryStats[lotteryType] = make(map[string]int)
		}
		lotteryStats[lotteryType][status] = count
	}

	stats["by_lottery_and_status"] = lotteryStats

	// Total geral
	totalQuery := "SELECT COUNT(*) FROM saved_games"
	var total int
	err = sg.db.QueryRow(totalQuery).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar total: %w", err)
	}
	stats["total"] = total

	return stats, nil
}

// Close fecha a conexão com o banco de dados
func (sg *SavedGamesDB) Close() error {
	return sg.db.Close()
} 