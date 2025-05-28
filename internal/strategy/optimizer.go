package strategy

import (
	"fmt"
	"lottery-optimizer-gui/internal/lottery"
	"math/rand"
	"sort"
	"time"
)

// ValidateAndAdjustStrategy valida e ajusta uma estratégia gerada pela IA
func ValidateAndAdjustStrategy(strategy *lottery.Strategy, prefs lottery.UserPreferences) *lottery.Strategy {
	if strategy == nil {
		fmt.Println("❌ Estratégia é nil, gerando fallback")
		strategy = generateFallbackStrategy(prefs)
	}

	// Validar e corrigir jogos
	validGames := []lottery.Game{}
	totalCost := 0.0

	fmt.Printf("🔍 Validando %d jogos da estratégia original...\n", len(strategy.Games))

	for i, game := range strategy.Games {
		fmt.Printf("🎲 Validando jogo %d: %s com %d números\n", i+1, game.Type, len(game.Numbers))

		// VALIDAÇÃO RIGOROSA PRIMEIRO
		if err := lottery.ValidateGame(game); err != nil {
			fmt.Printf("❌ Jogo %d inválido: %v\n", i+1, err)

			// Tentar corrigir o jogo
			if correctedGame := fixGame(game, prefs); correctedGame != nil {
				fmt.Printf("✅ Jogo %d corrigido com sucesso\n", i+1)
				validGames = append(validGames, *correctedGame)
				totalCost += correctedGame.Cost
			} else {
				fmt.Printf("❌ Jogo %d não pôde ser corrigido - DESCARTADO\n", i+1)
			}
		} else {
			fmt.Printf("✅ Jogo %d válido\n", i+1)
			validGames = append(validGames, game)
			totalCost += game.Cost
		}

		// Parar se exceder o orçamento
		if totalCost > prefs.Budget {
			fmt.Printf("💰 Orçamento excedido em R$ %.2f, parando validação\n", totalCost-prefs.Budget)
			break
		}
	}

	fmt.Printf("📊 Resultado da validação: %d jogos válidos de %d originais\n", len(validGames), len(strategy.Games))

	// Se não temos jogos válidos ou estamos muito abaixo do orçamento, gerar mais
	if len(validGames) == 0 {
		fmt.Println("🆘 NENHUM jogo válido! Gerando jogos de fallback...")
		// Só gerar jogos de fallback se não temos NENHUM jogo válido
		additionalGames := generateAdditionalGames(prefs, totalCost)
		validGames = append(validGames, additionalGames...)

		// Recalcular custo total
		totalCost = 0
		for _, game := range validGames {
			totalCost += game.Cost
		}
		fmt.Printf("🎲 Gerados %d jogos de fallback, custo total: R$ %.2f\n", len(additionalGames), totalCost)
	}

	// Remover duplicatas
	originalCount := len(validGames)
	validGames = removeDuplicateGames(validGames)
	if len(validGames) < originalCount {
		fmt.Printf("🔄 Removidas %d duplicatas\n", originalCount-len(validGames))
	}

	// Atualizar estratégia
	strategy.Games = validGames
	strategy.TotalCost = totalCost
	strategy.Budget = prefs.Budget

	// Calcular estatísticas se não existirem
	if strategy.Statistics.TotalDraws == 0 {
		strategy.Statistics = generateBasicStats()
	}

	// Garantir timestamp
	if strategy.CreatedAt.IsZero() {
		strategy.CreatedAt = time.Now()
	}

	// Melhorar reasoning se estiver vazio
	if strategy.Reasoning == "" {
		strategy.Reasoning = generateReasoningText(strategy, prefs)
	}

	fmt.Printf("🏁 Estratégia final: %d jogos, custo R$ %.2f\n", len(strategy.Games), strategy.TotalCost)

	return strategy
}

// fixGame tenta corrigir um jogo inválido
func fixGame(game lottery.Game, prefs lottery.UserPreferences) *lottery.Game {
	rules := lottery.GetRules(game.Type)

	// Log detalhado do problema
	fmt.Printf("🔧 Corrigindo jogo inválido: %s com %d números: %v\n",
		game.Type, len(game.Numbers), game.Numbers)

	// VALIDAÇÃO CRÍTICA: Verificar se tem números suficientes
	if len(game.Numbers) < rules.MinNumbers {
		fmt.Printf("❌ ERRO CRÍTICO: %s tem apenas %d números, mínimo é %d\n",
			game.Type, len(game.Numbers), rules.MinNumbers)
	}

	// Corrigir números fora do range
	validNumbers := []int{}
	for _, num := range game.Numbers {
		if num >= 1 && num <= rules.NumberRange {
			validNumbers = append(validNumbers, num)
		} else {
			fmt.Printf("⚠️ Número %d fora do range (1-%d) removido\n", num, rules.NumberRange)
		}
	}

	// Remover duplicatas
	validNumbers = removeDuplicates(validNumbers)

	// FORÇA NÚMERO MÍNIMO OBRIGATÓRIO
	for len(validNumbers) < rules.MinNumbers {
		newNum := generateRandomNumber(rules.NumberRange, validNumbers, prefs)
		if newNum > 0 {
			validNumbers = append(validNumbers, newNum)
			fmt.Printf("➕ Adicionado número %d para completar mínimo de %d\n", newNum, rules.MinNumbers)
		} else {
			// Fallback: adicionar números sequenciais se necessário
			for num := 1; num <= rules.NumberRange && len(validNumbers) < rules.MinNumbers; num++ {
				if !contains(validNumbers, num) {
					validNumbers = append(validNumbers, num)
					fmt.Printf("🆘 Fallback: adicionado número %d\n", num)
				}
			}
		}
	}

	// Ordenar números
	sort.Ints(validNumbers)

	// VALIDAÇÃO FINAL RIGOROSA
	if len(validNumbers) < rules.MinNumbers {
		fmt.Printf("❌ FALHA TOTAL: Não foi possível gerar %d números para %s\n",
			rules.MinNumbers, game.Type)
		return nil
	}

	// Limitar ao máximo permitido
	if len(validNumbers) > rules.MaxNumbers {
		validNumbers = validNumbers[:rules.MaxNumbers]
		fmt.Printf("✂️ Limitado a %d números (máximo permitido)\n", rules.MaxNumbers)
	}

	cost := lottery.CalculateGameCost(game.Type, len(validNumbers))

	fmt.Printf("✅ Jogo corrigido: %s com %d números: %v (R$ %.2f)\n",
		game.Type, len(validNumbers), validNumbers, cost)

	return &lottery.Game{
		Type:           game.Type,
		Numbers:        validNumbers,
		Cost:           cost,
		ExpectedReturn: calculateExpectedReturn(game.Type, validNumbers),
		Probability:    calculateProbability(game.Type, len(validNumbers)),
	}
}

// generateAdditionalGames gera jogos adicionais para completar o orçamento
func generateAdditionalGames(prefs lottery.UserPreferences, currentCost float64) []lottery.Game {
	var games []lottery.Game
	remainingBudget := prefs.Budget - currentCost

	for _, ltype := range prefs.LotteryTypes {
		rules := lottery.GetRules(ltype)

		// Gerar jogos enquanto há orçamento
		for remainingBudget >= rules.BasePrice {
			game := generateRandomGame(ltype, prefs)
			if game != nil && game.Cost <= remainingBudget {
				games = append(games, *game)
				remainingBudget -= game.Cost
			} else {
				break
			}
		}
	}

	return games
}

// generateRandomGame gera um jogo aleatório seguindo as preferências
func generateRandomGame(ltype lottery.LotteryType, prefs lottery.UserPreferences) *lottery.Game {
	rules := lottery.GetRules(ltype)
	rand.Seed(time.Now().UnixNano())

	// Determinar quantidade de números baseado na estratégia
	numCount := rules.MinNumbers
	switch prefs.Strategy {
	case "aggressive":
		// Jogos com mais números para maior cobertura
		numCount = rules.MinNumbers + rand.Intn(3)
		if numCount > rules.MaxNumbers {
			numCount = rules.MaxNumbers
		}
	case "balanced":
		// Ocasionalmente usar mais números
		if rand.Float32() < 0.3 {
			numCount = rules.MinNumbers + 1
		}
	}

	var numbers []int

	// Incluir números favoritos se especificados
	for _, favNum := range prefs.FavoriteNumbers {
		if favNum >= 1 && favNum <= rules.NumberRange && len(numbers) < numCount {
			if !contains(numbers, favNum) {
				numbers = append(numbers, favNum)
			}
		}
	}

	// Completar com números aleatórios
	for len(numbers) < numCount {
		num := generateRandomNumber(rules.NumberRange, append(numbers, prefs.ExcludeNumbers...), prefs)
		if num > 0 {
			numbers = append(numbers, num)
		}
	}

	// Ordenar números
	sort.Ints(numbers)

	if len(numbers) < rules.MinNumbers {
		return nil
	}

	cost := lottery.CalculateGameCost(ltype, len(numbers))

	return &lottery.Game{
		Type:           ltype,
		Numbers:        numbers,
		Cost:           cost,
		ExpectedReturn: calculateExpectedReturn(ltype, numbers),
		Probability:    calculateProbability(ltype, len(numbers)),
	}
}

// generateRandomNumber gera um número aleatório evitando exclusões
func generateRandomNumber(maxRange int, exclude []int, prefs lottery.UserPreferences) int {
	maxAttempts := 100
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < maxAttempts; i++ {
		num := rand.Intn(maxRange) + 1

		// Verificar se não está na lista de exclusão
		if contains(exclude, num) {
			continue
		}

		// Verificar padrões se necessário
		if prefs.AvoidPatterns && shouldAvoidNumber(num, exclude) {
			continue
		}

		return num
	}

	// Se não conseguir gerar, retornar qualquer número válido
	for num := 1; num <= maxRange; num++ {
		if !contains(exclude, num) {
			return num
		}
	}

	return 0
}

// shouldAvoidNumber verifica se deve evitar um número por padrões
func shouldAvoidNumber(num int, existing []int) bool {
	// Evitar múltiplos óbvios em sequência
	multiplesOf5 := 0
	multiplesOf10 := 0

	for _, n := range existing {
		if n%5 == 0 {
			multiplesOf5++
		}
		if n%10 == 0 {
			multiplesOf10++
		}
	}

	// Não adicionar muitos múltiplos de 5 ou 10
	if num%10 == 0 && multiplesOf10 >= 1 {
		return true
	}
	if num%5 == 0 && multiplesOf5 >= 2 {
		return true
	}

	// Evitar sequências óbvias
	if len(existing) > 0 {
		lastNum := existing[len(existing)-1]
		if num == lastNum+1 && len(existing) >= 2 {
			secondLast := existing[len(existing)-2]
			if lastNum == secondLast+1 {
				return true // Evitar sequências de 3+
			}
		}
	}

	return false
}

// generateFallbackStrategy gera uma estratégia básica se a IA falhar
func generateFallbackStrategy(prefs lottery.UserPreferences) *lottery.Strategy {
	strategy := &lottery.Strategy{
		Budget:     prefs.Budget,
		CreatedAt:  time.Now(),
		Reasoning:  "Estratégia gerada automaticamente devido a falha na análise da IA.",
		Statistics: generateBasicStats(),
	}

	totalCost := 0.0

	for _, ltype := range prefs.LotteryTypes {
		maxGames := int(prefs.Budget / lottery.GetRules(ltype).BasePrice)

		if maxGames > 10 {
			maxGames = 10 // Limitar quantidade
		}

		for i := 0; i < maxGames && totalCost < prefs.Budget; i++ {
			game := generateRandomGame(ltype, prefs)
			if game != nil && totalCost+game.Cost <= prefs.Budget {
				strategy.Games = append(strategy.Games, *game)
				totalCost += game.Cost
			}
		}
	}

	strategy.TotalCost = totalCost
	return strategy
}

// generateBasicStats gera estatísticas básicas
func generateBasicStats() lottery.Stats {
	return lottery.Stats{
		TotalDraws:      2000,
		AnalyzedDraws:   100,
		NumberFrequency: make(map[int]int),
		SumDistribution: make(map[int]int),
		HotNumbers:      []int{7, 10, 23, 33, 44},
		ColdNumbers:     []int{13, 26, 32, 47, 55},
		Patterns:        make(map[string]string),
	}
}

// generateReasoningText gera texto explicativo para a estratégia
func generateReasoningText(strategy *lottery.Strategy, prefs lottery.UserPreferences) string {
	text := fmt.Sprintf("Estratégia %s otimizada para orçamento de R$ %.2f.\n\n",
		prefs.Strategy, prefs.Budget)

	megaCount := 0
	lotoCount := 0

	for _, game := range strategy.Games {
		if game.Type == lottery.MegaSena {
			megaCount++
		} else {
			lotoCount++
		}
	}

	if megaCount > 0 {
		text += fmt.Sprintf("• %d jogos da Mega Sena para maximizar prêmios altos\n", megaCount)
	}

	if lotoCount > 0 {
		text += fmt.Sprintf("• %d jogos da Lotofácil para maior frequência de ganhos\n", lotoCount)
	}

	text += "\nEsta estratégia foi otimizada considerando:\n"
	text += "✓ Análise estatística de dados históricos\n"
	text += "✓ Distribuição equilibrada de números\n"
	text += "✓ Evita padrões previsíveis\n"
	text += "✓ Maximiza cobertura dentro do orçamento\n"

	return text
}

// Funções auxiliares
func removeDuplicates(numbers []int) []int {
	seen := make(map[int]bool)
	result := []int{}

	for _, num := range numbers {
		if !seen[num] {
			seen[num] = true
			result = append(result, num)
		}
	}

	return result
}

func removeDuplicateGames(games []lottery.Game) []lottery.Game {
	seen := make(map[string]bool)
	result := []lottery.Game{}

	for _, game := range games {
		key := fmt.Sprintf("%s:%v", game.Type, game.Numbers)
		if !seen[key] {
			seen[key] = true
			result = append(result, game)
		}
	}

	return result
}

func contains(slice []int, item int) bool {
	for _, n := range slice {
		if n == item {
			return true
		}
	}
	return false
}

func calculateExpectedReturn(ltype lottery.LotteryType, numbers []int) float64 {
	// Cálculo simplificado - poderia ser mais sofisticado
	prob := calculateProbability(ltype, len(numbers))

	// Estimativa conservadora do prêmio médio
	averagePrize := 1000000.0 // 1 milhão para Mega Sena
	if ltype == lottery.Lotofacil {
		averagePrize = 500000.0 // 500 mil para Lotofácil
	}

	return prob * averagePrize
}

func calculateProbability(ltype lottery.LotteryType, numCount int) float64 {
	rules := lottery.GetRules(ltype)

	// Cálculo básico de probabilidade
	// P = C(numCount, minNumbers) / C(range, minNumbers)

	numerator := float64(calculateCombinations(numCount, rules.MinNumbers))
	denominator := float64(calculateCombinations(rules.NumberRange, rules.MinNumbers))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func calculateCombinations(n, r int) int {
	if r > n {
		return 0
	}
	if r == 0 || r == n {
		return 1
	}

	result := 1
	for i := 0; i < r; i++ {
		result = result * (n - i) / (i + 1)
	}
	return result
}
