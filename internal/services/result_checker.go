package services

import (
	"fmt"
	"log"
	"sort"
	"time"

	"lottery-optimizer-gui/internal/data"
	"lottery-optimizer-gui/internal/database"
	"lottery-optimizer-gui/internal/lottery"
	"lottery-optimizer-gui/internal/models"
)

type ResultChecker struct {
	dataClient *data.Client
	db         *database.SavedGamesDB
}

// NewResultChecker cria uma nova instância do verificador de resultados
func NewResultChecker(dataClient *data.Client, db *database.SavedGamesDB) *ResultChecker {
	return &ResultChecker{
		dataClient: dataClient,
		db:         db,
	}
}

// CheckPendingResults verifica todos os jogos pendentes automaticamente
func (rc *ResultChecker) CheckPendingResults() error {
	pendingGames, err := rc.db.GetPendingGames()
	if err != nil {
		return fmt.Errorf("erro ao buscar jogos pendentes: %w", err)
	}

	log.Printf("Verificando %d jogos pendentes...", len(pendingGames))

	for _, game := range pendingGames {
		result, err := rc.CheckGameResult(game)
		if err != nil {
			log.Printf("Erro ao verificar jogo %s: %v", game.ID, err)
			// Marcar como erro mas continuar com os outros
			rc.db.UpdateGameStatus(game.ID, "error")
			continue
		}

		if result != nil {
			// Resultado encontrado, marcar como verificado
			rc.db.UpdateGameStatus(game.ID, "checked")
			log.Printf("Jogo %s verificado: %d acertos", game.ID, result.HitCount)
		}
		// Se result for nil, significa que o sorteio ainda não aconteceu
	}

	return nil
}

// CheckGameResult verifica o resultado de um jogo específico
func (rc *ResultChecker) CheckGameResult(game models.SavedGame) (*models.GameResult, error) {
	// Converter tipo de loteria para o formato interno
	var lotteryType lottery.LotteryType
	switch game.LotteryType {
	case "mega-sena":
		lotteryType = lottery.MegaSena
	case "lotofacil":
		lotteryType = lottery.Lotofacil
	default:
		return nil, fmt.Errorf("tipo de loteria não suportado: %s", game.LotteryType)
	}

	// Buscar resultado do concurso específico
	draw, err := rc.dataClient.GetDrawByNumber(lotteryType, game.ContestNumber)
	if err != nil {
		// Se o erro for de concurso não encontrado, pode ser que ainda não houve o sorteio
		return nil, nil // Retorna nil sem erro - sorteio ainda não aconteceu
	}

	// Verificar se o sorteio já aconteceu
	if draw == nil {
		return nil, nil // Sorteio ainda não aconteceu
	}

	// Calcular acertos
	userNumbers := []int(game.Numbers)
	drawnNumbers := draw.Numbers.ToIntSlice()
	matches := findMatches(userNumbers, drawnNumbers)

	result := &models.GameResult{
		ContestNumber: draw.Number,
		DrawDate:      draw.Date.String(),
		DrawnNumbers:  drawnNumbers,
		Matches:       matches,
		HitCount:      len(matches),
		IsWinner:      false,
	}

	// Determinar premiação baseada no tipo de loteria e número de acertos
	result.Prize, result.PrizeAmount, result.IsWinner = rc.calculatePrize(game.LotteryType, result.HitCount, draw)

	return result, nil
}

// CheckSingleGame verifica um jogo específico pelo ID
func (rc *ResultChecker) CheckSingleGame(gameID string) (*models.GameResult, error) {
	game, err := rc.db.GetGameByID(gameID)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar jogo: %w", err)
	}

	result, err := rc.CheckGameResult(*game)
	if err != nil {
		// Marcar como erro
		rc.db.UpdateGameStatus(gameID, "error")
		return nil, err
	}

	if result != nil {
		// Marcar como verificado
		rc.db.UpdateGameStatus(gameID, "checked")
	}

	return result, nil
}

// findMatches encontra números que coincidem entre duas listas
func findMatches(userNumbers, drawnNumbers []int) []int {
	drawnSet := make(map[int]bool)
	for _, num := range drawnNumbers {
		drawnSet[num] = true
	}

	var matches []int
	for _, num := range userNumbers {
		if drawnSet[num] {
			matches = append(matches, num)
		}
	}

	sort.Ints(matches)
	return matches
}

// calculatePrize calcula a premiação baseada no tipo de loteria e acertos
func (rc *ResultChecker) calculatePrize(lotteryType string, hitCount int, draw *lottery.Draw) (string, float64, bool) {
	switch lotteryType {
	case "mega-sena":
		return rc.calculateMegaSenaPrize(hitCount, draw)
	case "lotofacil":
		return rc.calculateLotofacilPrize(hitCount, draw)
	default:
		return "Tipo não suportado", 0, false
	}
}

// calculateMegaSenaPrize calcula premiação da Mega-Sena
func (rc *ResultChecker) calculateMegaSenaPrize(hitCount int, draw *lottery.Draw) (string, float64, bool) {
	// Buscar prêmios nos ganhadores
	var senaValue, quinaValue, quadraValue float64

	for _, winner := range draw.Winners {
		switch winner.Description {
		case "Sena":
			senaValue = winner.Prize
		case "Quina":
			quinaValue = winner.Prize
		case "Quadra":
			quadraValue = winner.Prize
		}
	}

	switch hitCount {
	case 6:
		return "Sena (6 acertos)", senaValue, true
	case 5:
		return "Quina (5 acertos)", quinaValue, true
	case 4:
		return "Quadra (4 acertos)", quadraValue, true
	default:
		return fmt.Sprintf("%d acertos", hitCount), 0, false
	}
}

// calculateLotofacilPrize calcula premiação da Lotofácil
func (rc *ResultChecker) calculateLotofacilPrize(hitCount int, draw *lottery.Draw) (string, float64, bool) {
	// Buscar prêmios nos ganhadores
	prizeMap := make(map[int]float64)

	// LOG DEBUG: Mostrar todas as descriptions da API
	log.Printf("🔍 DEBUG - Descriptions da API para concurso %d:", draw.Number)
	for _, winner := range draw.Winners {
		log.Printf("  - Description: '%s', Winners: %d, Prize: R$ %.2f", winner.Description, winner.Winners, winner.Prize)
	}

	for _, winner := range draw.Winners {
		switch winner.Description {
		// Formato "X acertos"
		case "15 acertos":
			prizeMap[15] = winner.Prize
		case "14 acertos":
			prizeMap[14] = winner.Prize
		case "13 acertos":
			prizeMap[13] = winner.Prize
		case "12 acertos":
			prizeMap[12] = winner.Prize
		case "11 acertos":
			prizeMap[11] = winner.Prize
		// Formato "X pontos" (usado pela API da CAIXA)
		case "15 pontos":
			prizeMap[15] = winner.Prize
		case "14 pontos":
			prizeMap[14] = winner.Prize
		case "13 pontos":
			prizeMap[13] = winner.Prize
		case "12 pontos":
			prizeMap[12] = winner.Prize
		case "11 pontos":
			prizeMap[11] = winner.Prize
		// Formato com faixas
		case "Faixa 1 (15 pontos)":
			prizeMap[15] = winner.Prize
		case "Faixa 2 (14 pontos)":
			prizeMap[14] = winner.Prize
		case "Faixa 3 (13 pontos)":
			prizeMap[13] = winner.Prize
		case "Faixa 4 (12 pontos)":
			prizeMap[12] = winner.Prize
		case "Faixa 5 (11 pontos)":
			prizeMap[11] = winner.Prize
		default:
			log.Printf("⚠️ Description não mapeada: '%s'", winner.Description)
		}
	}

	if prize, exists := prizeMap[hitCount]; exists && hitCount >= 11 {
		log.Printf("✅ Prêmio encontrado para %d acertos: R$ %.2f", hitCount, prize)
		return fmt.Sprintf("%d acertos", hitCount), prize, true
	}

	log.Printf("❌ Nenhum prêmio encontrado para %d acertos", hitCount)
	return fmt.Sprintf("%d acertos", hitCount), 0, false
}

// ScheduleAutoCheck agenda verificações automáticas
func (rc *ResultChecker) ScheduleAutoCheck() {
	ticker := time.NewTicker(6 * time.Hour) // Verificar a cada 6 horas
	go func() {
		for range ticker.C {
			if err := rc.CheckPendingResults(); err != nil {
				log.Printf("Erro na verificação automática: %v", err)
			}
		}
	}()
}
