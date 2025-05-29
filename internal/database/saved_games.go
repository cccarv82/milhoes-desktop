package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"lottery-optimizer-gui/internal/logs"
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
	// Primeiro, criar a tabela principal
	createTableQuery := `
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

	_, err := sg.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("erro ao criar tabela principal: %w", err)
	}

	// Verificar se colunas cost e prize existem e adicioná-las se necessário
	if err := sg.addColumnIfNotExists("cost", "REAL NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("erro ao adicionar coluna cost: %w", err)
	}

	if err := sg.addColumnIfNotExists("prize", "REAL NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("erro ao adicionar coluna prize: %w", err)
	}

	// Adicionar colunas para armazenar os resultados dos jogos
	if err := sg.addColumnIfNotExists("hit_count", "INTEGER DEFAULT NULL"); err != nil {
		return fmt.Errorf("erro ao adicionar coluna hit_count: %w", err)
	}

	if err := sg.addColumnIfNotExists("matches", "TEXT DEFAULT NULL"); err != nil {
		return fmt.Errorf("erro ao adicionar coluna matches: %w", err)
	}

	if err := sg.addColumnIfNotExists("drawn_numbers", "TEXT DEFAULT NULL"); err != nil {
		return fmt.Errorf("erro ao adicionar coluna drawn_numbers: %w", err)
	}

	if err := sg.addColumnIfNotExists("prize_description", "TEXT DEFAULT NULL"); err != nil {
		return fmt.Errorf("erro ao adicionar coluna prize_description: %w", err)
	}

	if err := sg.addColumnIfNotExists("prize_amount", "REAL DEFAULT NULL"); err != nil {
		return fmt.Errorf("erro ao adicionar coluna prize_amount: %w", err)
	}

	if err := sg.addColumnIfNotExists("is_winner", "INTEGER DEFAULT NULL"); err != nil {
		return fmt.Errorf("erro ao adicionar coluna is_winner: %w", err)
	}

	if err := sg.addColumnIfNotExists("contest_number_actual", "INTEGER DEFAULT NULL"); err != nil {
		return fmt.Errorf("erro ao adicionar coluna contest_number_actual: %w", err)
	}

	if err := sg.addColumnIfNotExists("draw_date", "TEXT DEFAULT NULL"); err != nil {
		return fmt.Errorf("erro ao adicionar coluna draw_date: %w", err)
	}

	return nil
}

// addColumnIfNotExists adiciona uma coluna se ela não existir
func (sg *SavedGamesDB) addColumnIfNotExists(columnName, columnDefinition string) error {
	// Verificar se a coluna já existe
	checkQuery := "PRAGMA table_info(saved_games)"
	rows, err := sg.db.Query(checkQuery)
	if err != nil {
		return fmt.Errorf("erro ao verificar estrutura da tabela: %w", err)
	}
	defer rows.Close()

	columnExists := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, dfltValue, pk interface{}

		err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
		if err != nil {
			return fmt.Errorf("erro ao ler informações da coluna: %w", err)
		}

		if name == columnName {
			columnExists = true
			break
		}
	}

	// Se a coluna não existe, adicioná-la
	if !columnExists {
		alterQuery := fmt.Sprintf("ALTER TABLE saved_games ADD COLUMN %s %s", columnName, columnDefinition)
		_, err := sg.db.Exec(alterQuery)
		if err != nil {
			return fmt.Errorf("erro ao adicionar coluna %s: %w", columnName, err)
		}
		logs.LogDatabase("✅ Coluna %s adicionada com sucesso", columnName)
	} else {
		logs.LogDatabase("ℹ️ Coluna %s já existe, pulando adição", columnName)
	}

	return nil
}

// SaveGame salva um novo jogo para verificação posterior
func (sg *SavedGamesDB) SaveGame(request models.SaveGameRequest) (*models.SavedGame, error) {
	logs.LogDatabase("🚀 Iniciando salvamento no banco de dados")
	logs.LogDatabase("📋 Request: %+v", request)

	game := &models.SavedGame{
		ID:            uuid.New().String(),
		LotteryType:   request.LotteryType,
		Numbers:       models.IntSlice(request.Numbers),
		ExpectedDraw:  request.ExpectedDraw,
		ContestNumber: request.ContestNumber,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	logs.LogDatabase("🎲 Objeto do jogo criado: ID=%s, Tipo=%s, Números=%v", game.ID, game.LotteryType, game.Numbers)

	query := `
		INSERT INTO saved_games (id, lottery_type, numbers, expected_draw, contest_number, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	logs.LogDatabase("📝 Executando query: %s", query)
	logs.LogDatabase("🔧 Parâmetros: ID=%s, Type=%s, Numbers=%v, Date=%s, Contest=%d, Status=%s",
		game.ID, game.LotteryType, game.Numbers, game.ExpectedDraw, game.ContestNumber, game.Status)

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
		logs.LogError(logs.CategoryDatabase, "❌ Erro no Exec da query: %v", err)
		return nil, fmt.Errorf("erro ao salvar jogo: %w", err)
	}

	logs.LogDatabase("✅ Jogo salvo com sucesso no banco! ID: %s", game.ID)

	return game, nil
}

// GetSavedGames busca jogos salvos com filtros opcionais
func (sg *SavedGamesDB) GetSavedGames(filter models.SavedGamesFilter) ([]models.SavedGame, error) {
	query := `SELECT id, lottery_type, numbers, expected_draw, contest_number, status, created_at, checked_at,
					 hit_count, matches, drawn_numbers, prize_description, prize_amount, is_winner, contest_number_actual, draw_date
			  FROM saved_games WHERE 1=1`
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

		// Campos do resultado
		var hitCount sql.NullInt64
		var matchesJSON sql.NullString
		var drawnNumbersJSON sql.NullString
		var prizeDescription sql.NullString
		var prizeAmount sql.NullFloat64
		var isWinner sql.NullInt64
		var contestNumberActual sql.NullInt64
		var drawDate sql.NullString

		err := rows.Scan(
			&game.ID,
			&game.LotteryType,
			&game.Numbers,
			&game.ExpectedDraw,
			&game.ContestNumber,
			&game.Status,
			&game.CreatedAt,
			&checkedAt,
			&hitCount,
			&matchesJSON,
			&drawnNumbersJSON,
			&prizeDescription,
			&prizeAmount,
			&isWinner,
			&contestNumberActual,
			&drawDate,
		)
		if err != nil {
			return nil, fmt.Errorf("erro ao fazer scan do jogo: %w", err)
		}

		if checkedAt.Valid {
			game.CheckedAt = &checkedAt.Time
		}

		// Se o jogo foi verificado e tem dados de resultado, carregar o resultado
		if game.Status == "checked" && hitCount.Valid {
			result := &models.GameResult{
				HitCount: int(hitCount.Int64),
			}

			// Deserializar matches
			if matchesJSON.Valid && matchesJSON.String != "" {
				var matches []int
				if err := json.Unmarshal([]byte(matchesJSON.String), &matches); err == nil {
					result.Matches = matches
				}
			}

			// Deserializar drawn numbers
			if drawnNumbersJSON.Valid && drawnNumbersJSON.String != "" {
				var drawnNumbers []int
				if err := json.Unmarshal([]byte(drawnNumbersJSON.String), &drawnNumbers); err == nil {
					result.DrawnNumbers = drawnNumbers
				}
			}

			if prizeDescription.Valid {
				result.Prize = prizeDescription.String
			}

			if prizeAmount.Valid {
				result.PrizeAmount = prizeAmount.Float64
			}

			if isWinner.Valid {
				result.IsWinner = isWinner.Int64 == 1
			}

			if contestNumberActual.Valid {
				result.ContestNumber = int(contestNumberActual.Int64)
			}

			if drawDate.Valid {
				result.DrawDate = drawDate.String
			}

			game.Result = result
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

// UpdateGameResult atualiza o resultado de um jogo verificado
func (sg *SavedGamesDB) UpdateGameResult(gameID string, result *models.GameResult) error {
	logs.LogDatabase("🎯 Atualizando resultado do jogo %s", gameID)
	logs.LogDatabase("📊 Resultado: %d acertos, prêmio: %s (R$ %.2f)", result.HitCount, result.Prize, result.PrizeAmount)

	// Converter slices para JSON
	matchesJSON, err := json.Marshal(result.Matches)
	if err != nil {
		return fmt.Errorf("erro ao serializar matches: %w", err)
	}

	drawnNumbersJSON, err := json.Marshal(result.DrawnNumbers)
	if err != nil {
		return fmt.Errorf("erro ao serializar drawn_numbers: %w", err)
	}

	query := `
		UPDATE saved_games 
		SET status = 'checked', 
			checked_at = ?,
			hit_count = ?,
			matches = ?,
			drawn_numbers = ?,
			prize_description = ?,
			prize_amount = ?,
			is_winner = ?,
			contest_number_actual = ?,
			draw_date = ?
		WHERE id = ?
	`

	isWinnerInt := 0
	if result.IsWinner {
		isWinnerInt = 1
	}

	_, err = sg.db.Exec(query,
		time.Now(),
		result.HitCount,
		string(matchesJSON),
		string(drawnNumbersJSON),
		result.Prize,
		result.PrizeAmount,
		isWinnerInt,
		result.ContestNumber,
		result.DrawDate,
		gameID,
	)

	if err != nil {
		logs.LogError(logs.CategoryDatabase, "❌ Erro ao atualizar resultado: %v", err)
		return fmt.Errorf("erro ao atualizar resultado do jogo: %w", err)
	}

	logs.LogDatabase("✅ Resultado atualizado com sucesso!")
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
	query := `SELECT id, lottery_type, numbers, expected_draw, contest_number, status, created_at, checked_at,
					 hit_count, matches, drawn_numbers, prize_description, prize_amount, is_winner, contest_number_actual, draw_date
			  FROM saved_games WHERE id = ?`

	var game models.SavedGame
	var checkedAt sql.NullTime

	// Campos do resultado
	var hitCount sql.NullInt64
	var matchesJSON sql.NullString
	var drawnNumbersJSON sql.NullString
	var prizeDescription sql.NullString
	var prizeAmount sql.NullFloat64
	var isWinner sql.NullInt64
	var contestNumberActual sql.NullInt64
	var drawDate sql.NullString

	err := sg.db.QueryRow(query, gameID).Scan(
		&game.ID,
		&game.LotteryType,
		&game.Numbers,
		&game.ExpectedDraw,
		&game.ContestNumber,
		&game.Status,
		&game.CreatedAt,
		&checkedAt,
		&hitCount,
		&matchesJSON,
		&drawnNumbersJSON,
		&prizeDescription,
		&prizeAmount,
		&isWinner,
		&contestNumberActual,
		&drawDate,
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

	// Se o jogo foi verificado e tem dados de resultado, carregar o resultado
	if game.Status == "checked" && hitCount.Valid {
		result := &models.GameResult{
			HitCount: int(hitCount.Int64),
		}

		// Deserializar matches
		if matchesJSON.Valid && matchesJSON.String != "" {
			var matches []int
			if err := json.Unmarshal([]byte(matchesJSON.String), &matches); err == nil {
				result.Matches = matches
			}
		}

		// Deserializar drawn numbers
		if drawnNumbersJSON.Valid && drawnNumbersJSON.String != "" {
			var drawnNumbers []int
			if err := json.Unmarshal([]byte(drawnNumbersJSON.String), &drawnNumbers); err == nil {
				result.DrawnNumbers = drawnNumbers
			}
		}

		if prizeDescription.Valid {
			result.Prize = prizeDescription.String
		}

		if prizeAmount.Valid {
			result.PrizeAmount = prizeAmount.Float64
		}

		if isWinner.Valid {
			result.IsWinner = isWinner.Int64 == 1
		}

		if contestNumberActual.Valid {
			result.ContestNumber = int(contestNumberActual.Int64)
		}

		if drawDate.Valid {
			result.DrawDate = drawDate.String
		}

		game.Result = result
	}

	return &game, nil
}

// GetAllSavedGames busca todos os jogos salvos (para analytics)
func (sg *SavedGamesDB) GetAllSavedGames() ([]models.SavedGame, error) {
	return sg.GetSavedGames(models.SavedGamesFilter{})
}

// GetGamesByLottery busca jogos por tipo de loteria (para analytics)
func (sg *SavedGamesDB) GetGamesByLottery(lotteryType string) ([]models.SavedGame, error) {
	return sg.GetSavedGames(models.SavedGamesFilter{LotteryType: lotteryType})
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
