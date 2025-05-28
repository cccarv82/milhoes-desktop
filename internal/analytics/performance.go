package analytics

import (
	"fmt"
	"lottery-optimizer-gui/internal/database"
	"lottery-optimizer-gui/internal/logs"
	"math"
	"sort"
	"time"
)

// PerformanceMetrics representa as métricas de performance do usuário
type PerformanceMetrics struct {
	// Métricas Gerais
	TotalGames      int     `json:"totalGames"`
	TotalInvestment float64 `json:"totalInvestment"`
	TotalWinnings   float64 `json:"totalWinnings"`
	ROI             float64 `json:"roi"`
	ROIPercentage   float64 `json:"roiPercentage"`

	// Métricas de Acerto
	GamesWithWins    int     `json:"gamesWithWins"`
	WinRate          float64 `json:"winRate"`
	AverageWinAmount float64 `json:"averageWinAmount"`
	BiggestWin       float64 `json:"biggestWin"`

	// Streaks
	CurrentWinStreak  int `json:"currentWinStreak"`
	LongestWinStreak  int `json:"longestWinStreak"`
	CurrentLossStreak int `json:"currentLossStreak"`
	LongestLossStreak int `json:"longestLossStreak"`

	// Por Período
	Last30Days  PeriodMetrics `json:"last30Days"`
	Last90Days  PeriodMetrics `json:"last90Days"`
	Last365Days PeriodMetrics `json:"last365Days"`

	// Por Loteria
	MegaSena  LotteryMetrics `json:"megaSena"`
	Lotofacil LotteryMetrics `json:"lotofacil"`

	// Análise Temporal
	PerformanceHistory []DailyPerformance `json:"performanceHistory"`
	MonthlyTrends      []MonthlyTrend     `json:"monthlyTrends"`
}

// PeriodMetrics representa métricas de um período específico
type PeriodMetrics struct {
	Games      int     `json:"games"`
	Investment float64 `json:"investment"`
	Winnings   float64 `json:"winnings"`
	ROI        float64 `json:"roi"`
	WinRate    float64 `json:"winRate"`
}

// LotteryMetrics representa métricas específicas por loteria
type LotteryMetrics struct {
	Name            string  `json:"name"`
	Games           int     `json:"games"`
	Investment      float64 `json:"investment"`
	Winnings        float64 `json:"winnings"`
	ROI             float64 `json:"roi"`
	WinRate         float64 `json:"winRate"`
	AverageWin      float64 `json:"averageWin"`
	BestStrategy    string  `json:"bestStrategy"`
	FavoriteNumbers []int   `json:"favoriteNumbers"`
}

// DailyPerformance representa performance diária
type DailyPerformance struct {
	Date       time.Time `json:"date"`
	Games      int       `json:"games"`
	Investment float64   `json:"investment"`
	Winnings   float64   `json:"winnings"`
	ROI        float64   `json:"roi"`
}

// MonthlyTrend representa tendência mensal
type MonthlyTrend struct {
	Month      string  `json:"month"`
	Year       int     `json:"year"`
	Games      int     `json:"games"`
	Investment float64 `json:"investment"`
	Winnings   float64 `json:"winnings"`
	ROI        float64 `json:"roi"`
	Growth     float64 `json:"growth"`
}

// NumberFrequency representa frequência de números
type NumberFrequency struct {
	Number    int       `json:"number"`
	Frequency int       `json:"frequency"`
	WinRate   float64   `json:"winRate"`
	LastSeen  time.Time `json:"lastSeen"`
	IsHot     bool      `json:"isHot"`
	IsCold    bool      `json:"isCold"`
}

// CalculatePerformanceMetrics calcula todas as métricas de performance
func CalculatePerformanceMetrics() (*PerformanceMetrics, error) {
	logs.LogAnalytics("🚀 Iniciando cálculo de métricas de performance...")

	// Buscar todos os jogos salvos
	savedGames, err := database.GetAllSavedGames()
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar jogos salvos: %v", err)
	}

	if len(savedGames) == 0 {
		logs.LogAnalytics("⚠️ Nenhum jogo salvo encontrado")
		return &PerformanceMetrics{}, nil
	}

	logs.LogAnalytics("📊 Processando %d jogos salvos...", len(savedGames))

	metrics := &PerformanceMetrics{}

	// Calcular métricas gerais
	calculateGeneralMetrics(metrics, savedGames)

	// Calcular métricas de streak
	calculateStreakMetrics(metrics, savedGames)

	// Calcular métricas por período
	calculatePeriodMetrics(metrics, savedGames)

	// Calcular métricas por loteria
	calculateLotteryMetrics(metrics, savedGames)

	// Calcular histórico de performance
	calculatePerformanceHistory(metrics, savedGames)

	// Calcular tendências mensais
	calculateMonthlyTrends(metrics, savedGames)

	logs.LogAnalytics("✅ Métricas calculadas com sucesso!")
	logs.LogAnalytics("📈 ROI Total: %.2f%% | Taxa de Acerto: %.2f%%",
		metrics.ROIPercentage, metrics.WinRate*100)

	return metrics, nil
}

// calculateGeneralMetrics calcula métricas gerais
func calculateGeneralMetrics(metrics *PerformanceMetrics, games []database.SavedGame) {
	var totalInvestment, totalWinnings float64
	gamesWithWins := 0
	var winAmounts []float64
	biggestWin := 0.0

	for _, game := range games {
		totalInvestment += game.Cost

		if game.Status == "checked" && game.Prize > 0 {
			totalWinnings += game.Prize
			gamesWithWins++
			winAmounts = append(winAmounts, game.Prize)

			if game.Prize > biggestWin {
				biggestWin = game.Prize
			}
		}
	}

	metrics.TotalGames = len(games)
	metrics.TotalInvestment = totalInvestment
	metrics.TotalWinnings = totalWinnings
	metrics.GamesWithWins = gamesWithWins

	// Calcular ROI
	if totalInvestment > 0 {
		metrics.ROI = totalWinnings - totalInvestment
		metrics.ROIPercentage = (metrics.ROI / totalInvestment) * 100
	}

	// Calcular taxa de acerto
	if len(games) > 0 {
		metrics.WinRate = float64(gamesWithWins) / float64(len(games))
	}

	// Calcular média de prêmios
	if len(winAmounts) > 0 {
		var sum float64
		for _, amount := range winAmounts {
			sum += amount
		}
		metrics.AverageWinAmount = sum / float64(len(winAmounts))
	}

	metrics.BiggestWin = biggestWin
}

// calculateStreakMetrics calcula métricas de sequências
func calculateStreakMetrics(metrics *PerformanceMetrics, games []database.SavedGame) {
	if len(games) == 0 {
		return
	}

	// Ordenar jogos por data de criação
	sort.Slice(games, func(i, j int) bool {
		return games[i].CreatedAt.Before(games[j].CreatedAt)
	})

	currentWinStreak := 0
	longestWinStreak := 0
	currentLossStreak := 0
	longestLossStreak := 0

	for _, game := range games {
		if game.Status == "checked" {
			if game.Prize > 0 {
				// Vitória
				currentWinStreak++
				currentLossStreak = 0

				if currentWinStreak > longestWinStreak {
					longestWinStreak = currentWinStreak
				}
			} else {
				// Derrota
				currentLossStreak++
				currentWinStreak = 0

				if currentLossStreak > longestLossStreak {
					longestLossStreak = currentLossStreak
				}
			}
		}
	}

	metrics.CurrentWinStreak = currentWinStreak
	metrics.LongestWinStreak = longestWinStreak
	metrics.CurrentLossStreak = currentLossStreak
	metrics.LongestLossStreak = longestLossStreak
}

// calculatePeriodMetrics calcula métricas por período
func calculatePeriodMetrics(metrics *PerformanceMetrics, games []database.SavedGame) {
	now := time.Now()

	// Últimos 30 dias
	metrics.Last30Days = calculatePeriodStats(games, now.AddDate(0, 0, -30))

	// Últimos 90 dias
	metrics.Last90Days = calculatePeriodStats(games, now.AddDate(0, 0, -90))

	// Últimos 365 dias
	metrics.Last365Days = calculatePeriodStats(games, now.AddDate(0, 0, -365))
}

// calculatePeriodStats calcula estatísticas para um período
func calculatePeriodStats(games []database.SavedGame, since time.Time) PeriodMetrics {
	var periodGames []database.SavedGame

	for _, game := range games {
		if game.CreatedAt.After(since) {
			periodGames = append(periodGames, game)
		}
	}

	if len(periodGames) == 0 {
		return PeriodMetrics{}
	}

	var investment, winnings float64
	wins := 0

	for _, game := range periodGames {
		investment += game.Cost

		if game.Status == "checked" && game.Prize > 0 {
			winnings += game.Prize
			wins++
		}
	}

	var roi, winRate float64

	if investment > 0 {
		roi = ((winnings - investment) / investment) * 100
	}

	if len(periodGames) > 0 {
		winRate = float64(wins) / float64(len(periodGames))
	}

	return PeriodMetrics{
		Games:      len(periodGames),
		Investment: investment,
		Winnings:   winnings,
		ROI:        roi,
		WinRate:    winRate,
	}
}

// calculateLotteryMetrics calcula métricas por loteria
func calculateLotteryMetrics(metrics *PerformanceMetrics, games []database.SavedGame) {
	megaSenaGames := filterGamesByLottery(games, "Mega-Sena")
	lotofacilGames := filterGamesByLottery(games, "Lotofácil")

	metrics.MegaSena = calculateLotteryStats("Mega-Sena", megaSenaGames)
	metrics.Lotofacil = calculateLotteryStats("Lotofácil", lotofacilGames)
}

// filterGamesByLottery filtra jogos por tipo de loteria
func filterGamesByLottery(games []database.SavedGame, lotteryType string) []database.SavedGame {
	var filtered []database.SavedGame

	for _, game := range games {
		if game.LotteryType == lotteryType {
			filtered = append(filtered, game)
		}
	}

	return filtered
}

// calculateLotteryStats calcula estatísticas para uma loteria específica
func calculateLotteryStats(name string, games []database.SavedGame) LotteryMetrics {
	if len(games) == 0 {
		return LotteryMetrics{Name: name}
	}

	var investment, winnings float64
	wins := 0
	var winAmounts []float64
	numberFreq := make(map[int]int)

	for _, game := range games {
		investment += game.Cost

		if game.Status == "checked" && game.Prize > 0 {
			winnings += game.Prize
			wins++
			winAmounts = append(winAmounts, game.Prize)
		}

		// Contar frequência de números
		for _, num := range game.Numbers {
			numberFreq[num]++
		}
	}

	var roi, winRate, averageWin float64

	if investment > 0 {
		roi = ((winnings - investment) / investment) * 100
	}

	if len(games) > 0 {
		winRate = float64(wins) / float64(len(games))
	}

	if len(winAmounts) > 0 {
		var sum float64
		for _, amount := range winAmounts {
			sum += amount
		}
		averageWin = sum / float64(len(winAmounts))
	}

	// Encontrar números favoritos (mais frequentes)
	var favoriteNumbers []int
	type numberCount struct {
		number int
		count  int
	}

	var counts []numberCount
	for num, count := range numberFreq {
		counts = append(counts, numberCount{num, count})
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	// Pegar top 10 números favoritos
	limit := 10
	if len(counts) < limit {
		limit = len(counts)
	}

	for i := 0; i < limit; i++ {
		favoriteNumbers = append(favoriteNumbers, counts[i].number)
	}

	return LotteryMetrics{
		Name:            name,
		Games:           len(games),
		Investment:      investment,
		Winnings:        winnings,
		ROI:             roi,
		WinRate:         winRate,
		AverageWin:      averageWin,
		BestStrategy:    "Balanceada", // TODO: implementar análise de estratégia
		FavoriteNumbers: favoriteNumbers,
	}
}

// calculatePerformanceHistory calcula histórico de performance diária
func calculatePerformanceHistory(metrics *PerformanceMetrics, games []database.SavedGame) {
	// Agrupar jogos por data
	dailyStats := make(map[string][]database.SavedGame)

	for _, game := range games {
		dateKey := game.CreatedAt.Format("2006-01-02")
		dailyStats[dateKey] = append(dailyStats[dateKey], game)
	}

	// Calcular performance diária
	var history []DailyPerformance

	for dateStr, dayGames := range dailyStats {
		date, _ := time.Parse("2006-01-02", dateStr)

		var investment, winnings float64

		for _, game := range dayGames {
			investment += game.Cost

			if game.Status == "checked" && game.Prize > 0 {
				winnings += game.Prize
			}
		}

		var roi float64
		if investment > 0 {
			roi = ((winnings - investment) / investment) * 100
		}

		history = append(history, DailyPerformance{
			Date:       date,
			Games:      len(dayGames),
			Investment: investment,
			Winnings:   winnings,
			ROI:        roi,
		})
	}

	// Ordenar por data
	sort.Slice(history, func(i, j int) bool {
		return history[i].Date.Before(history[j].Date)
	})

	metrics.PerformanceHistory = history
}

// calculateMonthlyTrends calcula tendências mensais
func calculateMonthlyTrends(metrics *PerformanceMetrics, games []database.SavedGame) {
	// Agrupar jogos por mês
	monthlyStats := make(map[string][]database.SavedGame)

	for _, game := range games {
		monthKey := game.CreatedAt.Format("2006-01")
		monthlyStats[monthKey] = append(monthlyStats[monthKey], game)
	}

	// Calcular tendências mensais
	var trends []MonthlyTrend

	for monthStr, monthGames := range monthlyStats {
		date, _ := time.Parse("2006-01", monthStr)

		var investment, winnings float64

		for _, game := range monthGames {
			investment += game.Cost

			if game.Status == "checked" && game.Prize > 0 {
				winnings += game.Prize
			}
		}

		var roi float64
		if investment > 0 {
			roi = ((winnings - investment) / investment) * 100
		}

		trends = append(trends, MonthlyTrend{
			Month:      date.Format("January"),
			Year:       date.Year(),
			Games:      len(monthGames),
			Investment: investment,
			Winnings:   winnings,
			ROI:        roi,
			Growth:     0, // TODO: calcular crescimento vs mês anterior
		})
	}

	// Ordenar por data
	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Year < trends[j].Year ||
			(trends[i].Year == trends[j].Year && trends[i].Month < trends[j].Month)
	})

	// Calcular crescimento mês a mês
	for i := 1; i < len(trends); i++ {
		if trends[i-1].ROI != 0 {
			trends[i].Growth = ((trends[i].ROI - trends[i-1].ROI) / math.Abs(trends[i-1].ROI)) * 100
		}
	}

	metrics.MonthlyTrends = trends
}

// GetNumberFrequencyAnalysis retorna análise de frequência de números
func GetNumberFrequencyAnalysis(lotteryType string) ([]NumberFrequency, error) {
	logs.LogAnalytics("🔍 Analisando frequência de números para %s...", lotteryType)

	games, err := database.GetGamesByLottery(lotteryType)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar jogos: %v", err)
	}

	if len(games) == 0 {
		return []NumberFrequency{}, nil
	}

	// Contar frequência e calcular win rate por número
	numberStats := make(map[int]*NumberFrequency)

	for _, game := range games {
		for _, num := range game.Numbers {
			if numberStats[num] == nil {
				numberStats[num] = &NumberFrequency{
					Number:    num,
					Frequency: 0,
					WinRate:   0,
					LastSeen:  time.Time{},
					IsHot:     false,
					IsCold:    false,
				}
			}

			numberStats[num].Frequency++

			if game.CreatedAt.After(numberStats[num].LastSeen) {
				numberStats[num].LastSeen = game.CreatedAt
			}

			// Se o jogo teve prêmio, contribui para o win rate
			if game.Status == "checked" && game.Prize > 0 {
				// Implementar lógica de win rate por número
			}
		}
	}

	// Converter para slice e calcular hot/cold
	var frequencies []NumberFrequency
	var allFreqs []int

	for _, freq := range numberStats {
		frequencies = append(frequencies, *freq)
		allFreqs = append(allFreqs, freq.Frequency)
	}

	// Calcular média e desvio padrão para determinar hot/cold
	if len(allFreqs) > 0 {
		var sum float64
		for _, f := range allFreqs {
			sum += float64(f)
		}
		mean := sum / float64(len(allFreqs))

		var variance float64
		for _, f := range allFreqs {
			variance += math.Pow(float64(f)-mean, 2)
		}
		stdDev := math.Sqrt(variance / float64(len(allFreqs)))

		// Marcar números hot/cold baseado em desvio padrão
		for i := range frequencies {
			freq := float64(frequencies[i].Frequency)
			if freq > mean+stdDev {
				frequencies[i].IsHot = true
			} else if freq < mean-stdDev {
				frequencies[i].IsCold = true
			}
		}
	}

	// Ordenar por frequência
	sort.Slice(frequencies, func(i, j int) bool {
		return frequencies[i].Frequency > frequencies[j].Frequency
	})

	logs.LogAnalytics("✅ Análise concluída: %d números analisados", len(frequencies))

	return frequencies, nil
}
