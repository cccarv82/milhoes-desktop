package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// SavedGame representa um jogo salvo pelo usuário para verificação posterior
type SavedGame struct {
	ID            string      `json:"id" db:"id"`
	LotteryType   string      `json:"lottery_type" db:"lottery_type"`     // "mega-sena", "lotofacil", etc.
	Numbers       IntSlice    `json:"numbers" db:"numbers"`               // Números apostados
	ExpectedDraw  string      `json:"expected_draw" db:"expected_draw"`   // Data esperada do sorteio (YYYY-MM-DD)
	ContestNumber int         `json:"contest_number" db:"contest_number"` // Número do concurso esperado
	Status        string      `json:"status" db:"status"`                 // "pending", "checked", "error"
	Cost          float64     `json:"cost" db:"cost"`                     // Custo do jogo
	Prize         float64     `json:"prize" db:"prize"`                   // Valor do prêmio (se ganhou)
	CreatedAt     time.Time   `json:"created_at" db:"created_at"`
	CheckedAt     *time.Time  `json:"checked_at,omitempty" db:"checked_at"`
	Result        *GameResult `json:"result,omitempty"` // Resultado da verificação (não armazenado no DB)
}

// GameResult representa o resultado da verificação de um jogo salvo
type GameResult struct {
	ContestNumber int     `json:"contest_number"` // Número do concurso real
	DrawDate      string  `json:"draw_date"`      // Data real do sorteio
	DrawnNumbers  []int   `json:"drawn_numbers"`  // Números sorteados
	Matches       []int   `json:"matches"`        // Números que o usuário acertou
	HitCount      int     `json:"hit_count"`      // Quantos números acertou
	Prize         string  `json:"prize"`          // Faixa de premiação ("quadra", "quina", "sena", etc.)
	PrizeAmount   float64 `json:"prize_amount"`   // Valor do prêmio (se ganhou)
	IsWinner      bool    `json:"is_winner"`      // Se ganhou algum prêmio
}

// IntSlice é um helper para serializar []int no SQLite
type IntSlice []int

// Value implementa driver.Valuer para SQLite
func (is IntSlice) Value() (driver.Value, error) {
	return json.Marshal(is)
}

// Scan implementa sql.Scanner para SQLite
func (is *IntSlice) Scan(value interface{}) error {
	if value == nil {
		*is = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, is)
	case string:
		return json.Unmarshal([]byte(v), is)
	}

	return nil
}

// SaveGameRequest representa a requisição para salvar um jogo
type SaveGameRequest struct {
	LotteryType   string `json:"lottery_type"`
	Numbers       []int  `json:"numbers"`
	ExpectedDraw  string `json:"expected_draw"`
	ContestNumber int    `json:"contest_number"`
}

// SavedGamesFilter representa filtros para buscar jogos salvos
type SavedGamesFilter struct {
	LotteryType string `json:"lottery_type,omitempty"`
	Status      string `json:"status,omitempty"`
	FromDate    string `json:"from_date,omitempty"`
	ToDate      string `json:"to_date,omitempty"`
}
