package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"lottery-optimizer-gui/internal/config"
	"lottery-optimizer-gui/internal/logs"
	"lottery-optimizer-gui/internal/lottery"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ClaudeClient cliente para API do Claude
type ClaudeClient struct {
	apiKey     string
	baseURL    string
	model      string
	maxTokens  int
	httpClient *http.Client
}

// ClaudeRequest estrutura da requisição para Claude
type ClaudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

// Message mensagem para Claude
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse resposta da API do Claude
type ClaudeResponse struct {
	Content []Content `json:"content"`
	Usage   Usage     `json:"usage"`
}

// Content conteúdo da resposta
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Usage uso de tokens
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewClaudeClient cria um novo cliente para Claude
func NewClaudeClient() *ClaudeClient {
	return &ClaudeClient{
		apiKey:    config.GetClaudeAPIKey(),
		baseURL:   "https://api.anthropic.com/v1/messages",
		model:     config.GetClaudeModel(),
		maxTokens: config.GetMaxTokens(),
		httpClient: &http.Client{
			Timeout: time.Duration(config.GlobalConfig.Claude.TimeoutSec) * time.Second,
		},
	}
}

// NewClaudeClientWithConfig cria um cliente Claude com configurações específicas
func NewClaudeClientWithConfig(apiKey, model string, maxTokens, timeoutSec int) *ClaudeClient {
	return &ClaudeClient{
		apiKey:    apiKey,
		baseURL:   "https://api.anthropic.com/v1/messages",
		model:     model,
		maxTokens: maxTokens,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

// AnalyzeStrategy usa Claude para analisar dados e gerar estratégia
func (c *ClaudeClient) AnalyzeStrategy(request lottery.AnalysisRequest) (*lottery.AnalysisResponse, error) {
	// Logs especializados de IA
	logs.LogAI("🔍 Iniciando AnalyzeStrategy...")
	logs.LogAI("🔍 API Key length: %d", len(c.apiKey))

	if c.apiKey != "" {
		logs.LogAI("🔍 API Key prefix: %s", c.apiKey[:min(10, len(c.apiKey))])
	} else {
		logs.LogError(logs.CategoryAI, "API Key VAZIA! ❌")
		return nil, fmt.Errorf("chave da API do Claude não configurada")
	}

	logs.LogAI("🔍 Model: %s", c.model)
	logs.LogAI("🔍 MaxTokens: %d", c.maxTokens)
	logs.LogAI("🔍 BaseURL: %s", c.baseURL)

	prompt := c.BuildAnalysisPrompt(request)

	claudeReq := ClaudeRequest{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	reqBody, err := json.Marshal(claudeReq)
	if err != nil {
		logs.LogError(logs.CategoryAI, "Erro ao serializar requisição: %v", err)
		return nil, fmt.Errorf("erro ao serializar requisição: %w", err)
	}

	logs.LogAI("🔍 Request body preparado. Size: %d bytes", len(reqBody))

	// Implementar retry logic com exponential backoff
	var resp *http.Response
	maxRetries := 3
	baseDelay := 2 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(reqBody))
		if err != nil {
			logs.LogError(logs.CategoryAI, "Erro ao criar requisição: %v", err)
			return nil, fmt.Errorf("erro ao criar requisição: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err = c.httpClient.Do(req)
		if err != nil {
			if attempt < maxRetries-1 {
				delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
				if config.IsVerbose() {
					logs.LogAI("⚠️ Tentativa %d falhou, tentando novamente em %v...", attempt+1, delay)
				}
				time.Sleep(delay)
				continue
			}
			logs.LogError(logs.CategoryAI, "Erro na requisição após %d tentativas: %v", maxRetries, err)
			return nil, fmt.Errorf("erro na requisição após %d tentativas: %w", maxRetries, err)
		}
		break
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logs.LogError(logs.CategoryAI, "API retornou status %d", resp.StatusCode)
		return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
	}

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		logs.LogError(logs.CategoryAI, "Erro ao decodificar resposta: %v", err)
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if len(claudeResp.Content) == 0 {
		logs.LogError(logs.CategoryAI, "Resposta vazia do Claude")
		return nil, fmt.Errorf("resposta vazia do Claude")
	}

	// Extrair JSON limpo da resposta
	rawResponse := claudeResp.Content[0].Text
	jsonContent := extractJSON(rawResponse)

	// Enhanced debug logging
	if config.IsVerbose() {
		logs.LogAI("🤖 Resposta COMPLETA do Claude: %s", rawResponse)
		logs.LogAI("🔍 JSON extraído: %s", jsonContent)
	} else {
		logs.LogAI("🤖 Resposta do Claude: %s", rawResponse[:min(200, len(rawResponse))]+"...")
		logs.LogAI("🔍 JSON extraído: %s", jsonContent[:min(200, len(jsonContent))]+"...")
	}

	// Parsear a resposta JSON do Claude
	var analysisResp lottery.AnalysisResponse
	if err := json.Unmarshal([]byte(jsonContent), &analysisResp); err != nil {
		logs.LogError(logs.CategoryAI, "❌ Erro ao fazer parse do JSON: %v", err)
		logs.LogAI("📄 JSON que falhou: %s", jsonContent)

		// Tentar identificar o problema específico no JSON
		if strings.Contains(err.Error(), "invalid character") {
			logs.LogAI("🔍 Problema: JSON contém caracteres inválidos")
		} else if strings.Contains(err.Error(), "unexpected end") {
			logs.LogAI("🔍 Problema: JSON incompleto ou cortado")
		} else if strings.Contains(err.Error(), "cannot unmarshal") {
			logs.LogAI("🔍 Problema: Estrutura JSON não corresponde ao esperado")
		}

		// Verificar se o JSON extraído está vazio ou muito pequeno
		if len(strings.TrimSpace(jsonContent)) < 50 {
			logs.LogAI("⚠️ JSON extraído muito pequeno: '%s'", jsonContent)
			logs.LogAI("📄 Resposta completa do Claude: %s", rawResponse)
		}

		// SEM FALLBACK! Retornar erro para o usuário tentar novamente
		return nil, fmt.Errorf("erro no parsing da resposta do Claude - JSON inválido: %v", err)
	} else {
		// CORREÇÃO AUTOMÁTICA DE TIPOS DE LOTERIA INCORRETOS
		for i := range analysisResp.Strategy.Games {
			game := &analysisResp.Strategy.Games[i]

			// Converter tipos incorretos que o Claude possa retornar
			switch string(game.Type) {
			case "mega-sena", "megasena", "Mega-Sena", "MEGASENA":
				game.Type = lottery.MegaSena
				logs.LogAI("🔧 CORRIGINDO TIPO: '%s' -> 'megasena'", string(game.Type))
			case "loto-facil", "lotofacil", "Lotofácil", "LOTOFACIL":
				game.Type = lottery.Lotofacil
				logs.LogAI("🔧 CORRIGINDO TIPO: '%s' -> 'lotofacil'", string(game.Type))
			}
		}

		// VALIDAÇÃO CRÍTICA DE CUSTOS - Corrigir custos incorretos do Claude
		totalCostRecalculated := 0.0
		for i := range analysisResp.Strategy.Games {
			game := &analysisResp.Strategy.Games[i]
			correctCost := lottery.CalculateGameCost(game.Type, len(game.Numbers))

			if game.Cost != correctCost {
				logs.LogAI("🔧 CORRIGINDO CUSTO: %s com %d números - Claude retornou R$ %.2f, correto é R$ %.2f",
					game.Type, len(game.Numbers), game.Cost, correctCost)
				game.Cost = correctCost
			}
			totalCostRecalculated += game.Cost
		}

		// Atualizar custo total se necessário
		if analysisResp.Strategy.TotalCost != totalCostRecalculated {
			logs.LogAI("🔧 CORRIGINDO CUSTO TOTAL: Claude retornou R$ %.2f, correto é R$ %.2f",
				analysisResp.Strategy.TotalCost, totalCostRecalculated)
			analysisResp.Strategy.TotalCost = totalCostRecalculated
		}

		// VALIDAÇÃO CRÍTICA DE PRIORIZAÇÃO LOTOFÁCIL
		megaCount := 0
		lotoCount := 0
		megaCost := 0.0
		lotoCost := 0.0

		for _, game := range analysisResp.Strategy.Games {
			if game.Type == lottery.MegaSena {
				megaCount++
				megaCost += game.Cost
			} else if game.Type == lottery.Lotofacil {
				lotoCount++
				lotoCost += game.Cost
			}
		}

		// Verificar se está priorizando Lotofácil corretamente
		if megaCount > lotoCount && lotoCount > 0 {
			logs.LogAI("⚠️ ESTRATÉGIA INCORRETA: %d jogos Mega-Sena vs %d Lotofácil - CORRIGINDO!", megaCount, lotoCount)

			// Remover jogos de Mega-Sena em excesso, mantendo apenas 1-2
			correctedGames := []lottery.Game{}
			megaAdded := 0
			maxMegaGames := 1
			if request.Preferences.Budget > 150 {
				maxMegaGames = 2
			}

			// Adicionar todos os jogos de Lotofácil primeiro
			for _, game := range analysisResp.Strategy.Games {
				if game.Type == lottery.Lotofacil {
					correctedGames = append(correctedGames, game)
				}
			}

			// Adicionar apenas alguns jogos de Mega-Sena
			for _, game := range analysisResp.Strategy.Games {
				if game.Type == lottery.MegaSena && megaAdded < maxMegaGames {
					correctedGames = append(correctedGames, game)
					megaAdded++
				}
			}

			// Recalcular custos
			newTotalCost := 0.0
			for _, game := range correctedGames {
				newTotalCost += game.Cost
			}

			analysisResp.Strategy.Games = correctedGames
			analysisResp.Strategy.TotalCost = newTotalCost

			logs.LogAI("✅ ESTRATÉGIA CORRIGIDA: %d Lotofácil + %d Mega-Sena = R$ %.2f",
				len(correctedGames)-megaAdded, megaAdded, newTotalCost)
		} else {
			logs.LogAI("✅ PRIORIZAÇÃO CORRETA: %d Lotofácil + %d Mega-Sena", lotoCount, megaCount)
		}

		// Validate parsed strategy
		if analysisResp.Strategy.Games == nil || len(analysisResp.Strategy.Games) == 0 {
			logs.LogAI("⚠️ JSON parseado mas sem jogos válidos")
			return nil, fmt.Errorf("estratégia inválida gerada pelo Claude - tente novamente")
		} else {
			// VALIDAÇÃO DE DIVERSIFICAÇÃO CRÍTICA
			if !validateDiversification(analysisResp.Strategy.Games) {
				logs.LogAI("🔄 Estratégia falhou na validação de diversificação, tentando novamente...")

				// Retry até 5 vezes mais para conseguir diversificação correta
				maxRetries := 5
				bestStrategy := analysisResp // Manter a melhor estratégia gerada

				for retry := 0; retry < maxRetries; retry++ {
					logs.LogAI("🔄 Tentativa %d/%d para diversificação correta...", retry+1, maxRetries)

					// Gerar nova estratégia
					newPrompt := c.BuildAnalysisPrompt(request)
					newClaudeReq := ClaudeRequest{
						Model:     c.model,
						MaxTokens: c.maxTokens,
						Messages: []Message{
							{
								Role:    "user",
								Content: newPrompt,
							},
						},
					}

					newReqBody, _ := json.Marshal(newClaudeReq)
					newReq, _ := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(newReqBody))
					newReq.Header.Set("Content-Type", "application/json")
					newReq.Header.Set("x-api-key", c.apiKey)
					newReq.Header.Set("anthropic-version", "2023-06-01")

					newResp, err := c.httpClient.Do(newReq)
					if err != nil {
						continue
					}
					defer newResp.Body.Close()

					if newResp.StatusCode != http.StatusOK {
						continue
					}

					var newClaudeResp ClaudeResponse
					if err := json.NewDecoder(newResp.Body).Decode(&newClaudeResp); err != nil {
						continue
					}

					if len(newClaudeResp.Content) == 0 {
						continue
					}

					newJsonContent := extractJSON(newClaudeResp.Content[0].Text)
					var newAnalysisResp lottery.AnalysisResponse

					if err := json.Unmarshal([]byte(newJsonContent), &newAnalysisResp); err == nil {
						// Validar custos da nova estratégia também
						newTotalCost := 0.0
						for i := range newAnalysisResp.Strategy.Games {
							game := &newAnalysisResp.Strategy.Games[i]
							correctCost := lottery.CalculateGameCost(game.Type, len(game.Numbers))
							game.Cost = correctCost
							newTotalCost += game.Cost
						}
						newAnalysisResp.Strategy.TotalCost = newTotalCost

						if validateDiversification(newAnalysisResp.Strategy.Games) {
							logs.LogAI("✅ Diversificação correta conseguida na tentativa %d!", retry+1)
							analysisResp = newAnalysisResp
							break
						} else {
							// Manter a estratégia com melhor orçamento/qualidade
							if newAnalysisResp.Strategy.TotalCost > bestStrategy.Strategy.TotalCost {
								bestStrategy = newAnalysisResp
								logs.LogAI("💡 Nova melhor estratégia encontrada: R$ %.2f", newAnalysisResp.Strategy.TotalCost)
							}
						}
					}
				}

				// Se não conseguiu diversificação perfeita, usar a MELHOR estratégia do Claude
				if !validateDiversification(analysisResp.Strategy.Games) {
					logs.LogAI("💪 Usando MELHOR estratégia Claude (sem fallback!): R$ %.2f - Qualidade superior!", bestStrategy.Strategy.TotalCost)
					analysisResp = bestStrategy
					analysisResp.Confidence = analysisResp.Confidence * 0.9 // Reduzir confiança ligeiramente
				}
			} else {
				logs.LogAI("✅ JSON parseado com sucesso: %d jogos gerados", len(analysisResp.Strategy.Games))
			}
		}
	}

	if config.IsVerbose() {
		logs.LogAI("Tokens usados: %d input + %d output = %d total",
			claudeResp.Usage.InputTokens, claudeResp.Usage.OutputTokens,
			claudeResp.Usage.InputTokens+claudeResp.Usage.OutputTokens)
	}

	return &analysisResp, nil
}

// generateFallbackStrategy generates a simple but valid strategy when AI parsing fails
func (c *ClaudeClient) generateFallbackStrategy(request lottery.AnalysisRequest) lottery.AnalysisResponse {
	budget := request.Preferences.Budget
	var games []lottery.Game
	totalCost := 0.0

	logs.LogAI("🔄 Gerando estratégia fallback INTELIGENTE para orçamento R$ %.2f", budget)

	// ESTRATÉGIA INTELIGENTE: Analisar qual opção maximiza combinações
	targetBudget := budget * 0.85 // Usar 85% do orçamento

	// Analisar opções matemáticas para Lotofácil
	options := []struct {
		numbers      int
		cost         float64
		combinations int
		description  string
	}{
		{15, 3.00, 1, "15 números"},
		{16, 48.00, 16, "16 números"},
		{17, 408.00, 136, "17 números"},
		{18, 2448.00, 816, "18 números"},
	}

	bestOption := options[0]
	maxCombinations := 0

	// Encontrar a opção que maximiza combinações dentro do orçamento
	for _, option := range options {
		if option.cost <= targetBudget {
			possibleGames := int(targetBudget / option.cost)
			totalCombinations := possibleGames * option.combinations

			logs.LogAI("📊 Opção %s: %d jogos × %d combinações = %d combinações totais (R$ %.2f)",
				option.description, possibleGames, option.combinations, totalCombinations, float64(possibleGames)*option.cost)

			if totalCombinations > maxCombinations {
				maxCombinations = totalCombinations
				bestOption = option
			}
		}
	}

	// Gerar jogos com a melhor opção encontrada
	for _, lotteryType := range request.Preferences.LotteryTypes {
		if lotteryType == lottery.Lotofacil {
			gamesCount := int(targetBudget / bestOption.cost)

			logs.LogAI("🍀 ESTRATÉGIA ÓTIMA: %d jogos de %s = %d combinações totais (R$ %.2f)",
				gamesCount, bestOption.description, gamesCount*bestOption.combinations, float64(gamesCount)*bestOption.cost)

			for i := 0; i < gamesCount && totalCost+bestOption.cost <= budget; i++ {
				// Gerar números baseado na quantidade otimizada
				numbers := generateOptimizedLotofacilNumbers(bestOption.numbers, i)
				cost := lottery.CalculateGameCost(lottery.Lotofacil, len(numbers))

				games = append(games, lottery.Game{
					Type:    lottery.Lotofacil,
					Numbers: numbers,
					Cost:    cost,
				})
				totalCost += cost
			}
		}
	}

	// Usar orçamento restante para Mega-Sena se disponível
	for _, lotteryType := range request.Preferences.LotteryTypes {
		if lotteryType == lottery.MegaSena {
			remainingBudget := budget - totalCost
			if remainingBudget >= 5.0 {
				megaGamesCount := int(remainingBudget / 5.0) // Jogos simples de 6 números

				logs.LogAI("🎰 Complementando com %d jogos de Mega-Sena (R$ %.2f)", megaGamesCount, float64(megaGamesCount)*5.0)

				for i := 0; i < megaGamesCount && totalCost+5 <= budget; i++ {
					numbers := generateMegaSenaNumbers(i)
					cost := lottery.CalculateGameCost(lottery.MegaSena, len(numbers))

					games = append(games, lottery.Game{
						Type:    lottery.MegaSena,
						Numbers: numbers,
						Cost:    cost,
					})
					totalCost += cost
				}
			}
		}
	}

	utilizationPercent := (totalCost / budget) * 100

	reasoning := fmt.Sprintf("Estratégia fallback matematicamente otimizada: %d jogos por R$ %.2f (%.1f%% do orçamento). "+
		"Análise matemática determinou que jogos de %s maximizam as combinações (%d combinações totais) para orçamento de R$ %.2f.",
		len(games), totalCost, utilizationPercent, bestOption.description, maxCombinations, budget)

	logs.LogAI("✅ Fallback INTELIGENTE: %d jogos, R$ %.2f (%.1f%% do orçamento), %d combinações totais",
		len(games), totalCost, utilizationPercent, maxCombinations)

	return lottery.AnalysisResponse{
		Strategy: lottery.Strategy{
			Budget:    budget,
			TotalCost: totalCost,
			Games:     games,
			Reasoning: reasoning,
			CreatedAt: time.Now(),
			Statistics: lottery.Stats{
				AnalyzedDraws: len(request.Draws),
				HotNumbers:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},          // Simple defaults
				ColdNumbers:   []int{51, 52, 53, 54, 55, 56, 57, 58, 59, 60}, // Simple defaults
			},
		},
		Confidence: 0.8, // Higher confidence for mathematically optimized fallback
	}
}

// generateOptimizedLotofacilNumbers gera números de Lotofácil com quantidade otimizada
func generateOptimizedLotofacilNumbers(count int, index int) []int {
	// Números base distribuídos uniformemente
	baseNumbers := []int{}
	step := 25 / count

	for i := 0; i < count; i++ {
		num := (i * step) + 1 + (index % 3) // Adicionar variação baseada no índice
		if num > 25 {
			num = num%25 + 1
		}
		baseNumbers = append(baseNumbers, num)
	}

	// Garantir que temos exatamente a quantidade correta de números únicos
	uniqueNumbers := make(map[int]bool)
	result := []int{}

	for _, num := range baseNumbers {
		if !uniqueNumbers[num] && len(result) < count {
			uniqueNumbers[num] = true
			result = append(result, num)
		}
	}

	// Completar se necessário
	for num := 1; num <= 25 && len(result) < count; num++ {
		if !uniqueNumbers[num] {
			result = append(result, num)
		}
	}

	return result
}

// generateLotofacilNumbers gera números de Lotofácil com variação baseada no índice
func generateLotofacilNumbers(index int) []int {
	baseNumbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	// Aplicar variação baseada no índice para diversificar
	offset := index % 10
	for i := range baseNumbers {
		baseNumbers[i] = ((baseNumbers[i] + offset - 1) % 25) + 1
	}

	// Garantir que temos exatamente 15 números únicos
	uniqueNumbers := make(map[int]bool)
	result := []int{}

	for _, num := range baseNumbers {
		if !uniqueNumbers[num] && len(result) < 15 {
			uniqueNumbers[num] = true
			result = append(result, num)
		}
	}

	// Completar se necessário
	for num := 1; num <= 25 && len(result) < 15; num++ {
		if !uniqueNumbers[num] {
			result = append(result, num)
		}
	}

	return result
}

// generateMegaSenaNumbers gera números de Mega-Sena com variação baseada no índice
func generateMegaSenaNumbers(index int) []int {
	baseNumbers := []int{7, 15, 23, 35, 42, 58}

	// Aplicar variação baseada no índice para diversificar
	offset := index % 10
	for i := range baseNumbers {
		baseNumbers[i] = ((baseNumbers[i] + offset - 1) % 60) + 1
	}

	// Garantir que temos exatamente 6 números únicos
	uniqueNumbers := make(map[int]bool)
	result := []int{}

	for _, num := range baseNumbers {
		if !uniqueNumbers[num] && len(result) < 6 {
			uniqueNumbers[num] = true
			result = append(result, num)
		}
	}

	// Completar se necessário
	for num := 1; num <= 60 && len(result) < 6; num++ {
		if !uniqueNumbers[num] {
			result = append(result, num)
		}
	}

	return result
}

// extractJSON extrai o primeiro JSON válido encontrado no texto
func extractJSON(text string) string {
	// Primeiro, tentar encontrar JSON entre ```json e ```
	if start := strings.Index(text, "```json"); start != -1 {
		start += len("```json")
		if end := strings.Index(text[start:], "```"); end != -1 {
			jsonContent := strings.TrimSpace(text[start : start+end])
			logs.LogAI("🔍 JSON encontrado entre markdown: %s", jsonContent[:min(100, len(jsonContent))]+"...")
			return jsonContent
		}
	}

	// Segundo, tentar encontrar JSON entre ``` e ```
	if start := strings.Index(text, "```"); start != -1 {
		start += 3
		if end := strings.Index(text[start:], "```"); end != -1 {
			jsonContent := strings.TrimSpace(text[start : start+end])
			// Verificar se parece com JSON
			if strings.HasPrefix(jsonContent, "{") && strings.HasSuffix(jsonContent, "}") {
				logs.LogAI("🔍 JSON encontrado entre markdown genérico: %s", jsonContent[:min(100, len(jsonContent))]+"...")
				return jsonContent
			}
		}
	}

	// Terceiro, procurar pelo início do JSON usando contagem de chaves
	start := -1
	braceCount := 0

	for i, char := range text {
		if char == '{' {
			if start == -1 {
				start = i
			}
			braceCount++
		} else if char == '}' {
			braceCount--
			if start != -1 && braceCount == 0 {
				jsonContent := text[start : i+1]
				logs.LogAI("🔍 JSON encontrado por contagem de chaves: %s", jsonContent[:min(100, len(jsonContent))]+"...")
				return jsonContent
			}
		}
	}

	// Se não encontrou JSON válido, tentar extrair qualquer coisa que pareça JSON
	lines := strings.Split(text, "\n")
	var jsonLines []string
	inJson := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") {
			inJson = true
			jsonLines = append(jsonLines, line)
		} else if inJson {
			jsonLines = append(jsonLines, line)
			if strings.HasSuffix(trimmed, "}") && strings.Count(strings.Join(jsonLines, "\n"), "{") == strings.Count(strings.Join(jsonLines, "\n"), "}") {
				jsonContent := strings.Join(jsonLines, "\n")
				logs.LogAI("🔍 JSON encontrado por análise de linhas: %s", jsonContent[:min(100, len(jsonContent))]+"...")
				return jsonContent
			}
		}
	}

	// Último recurso: retornar o texto original
	logs.LogAI("⚠️ Nenhum JSON válido encontrado, retornando texto original")
	return text
}

// extractReasoningFromText extrai um reasoning mais limpo do texto bruto
func extractReasoningFromText(text string) string {
	// Tentar extrair apenas o campo reasoning se possível
	if start := strings.Index(text, `"reasoning": "`); start != -1 {
		start += len(`"reasoning": "`)
		end := start
		escapeCount := 0

		for i := start; i < len(text); i++ {
			if text[i] == '\\' {
				escapeCount++
				continue
			}
			if text[i] == '"' && escapeCount%2 == 0 {
				end = i
				break
			}
			escapeCount = 0
		}

		if end > start {
			reasoning := text[start:end]
			// Remover escapes desnecessários
			reasoning = strings.ReplaceAll(reasoning, `\"`, `"`)
			reasoning = strings.ReplaceAll(reasoning, `\n`, "\n")
			return reasoning
		}
	}

	// Fallback: retornar uma versão mais limpa do texto
	cleaned := text
	// Remover JSON se estiver misturado
	if jsonStart := strings.Index(cleaned, "{"); jsonStart != -1 {
		if jsonStart > 50 { // Se tem texto antes do JSON
			cleaned = cleaned[:jsonStart]
		}
	}

	// Limitar tamanho
	if len(cleaned) > 500 {
		cleaned = cleaned[:500] + "..."
	}

	if strings.TrimSpace(cleaned) == "" {
		return "Estratégia gerada com base em análise estatística de dados históricos."
	}

	return strings.TrimSpace(cleaned)
}

// min função auxiliar para retornar o menor valor
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BuildAnalysisPrompt constrói o prompt para análise com ESTRATÉGIAS PROFISSIONAIS MUNDIAIS
func (c *ClaudeClient) BuildAnalysisPrompt(request lottery.AnalysisRequest) string {
	budget := request.Preferences.Budget

	// ANÁLISE ESTATÍSTICA RIGOROSA DOS DADOS HISTÓRICOS REAIS
	statisticalAnalysis := c.analyzeHistoricalData(request.Draws, request.Preferences.LotteryTypes)

	prompt := fmt.Sprintf(`Você é um especialista em loterias. Analise os dados históricos e gere uma estratégia otimizada.

ORÇAMENTO DISPONÍVEL: R$ %.2f
OBJETIVO: Maximizar probabilidade de ganho usando 85-95%% do orçamento.

=== DADOS HISTÓRICOS ===
%s

=== REGRAS OBRIGATÓRIAS ===

LOTOFÁCIL:
- Mínimo: 15 números, Máximo: 20 números
- Preços: 15 números = R$ 3,00 | 16 números = R$ 48,00 | 17 números = R$ 408,00 | 18 números = R$ 2.448,00 | 19 números = R$ 11.628,00 | 20 números = R$ 46.512,00

MEGA-SENA:
- Mínimo: 6 números, Máximo: 20 números  
- Preços: 6 números = R$ 5,00 | 7 números = R$ 35,00 | 8 números = R$ 140,00 | 9 números = R$ 420,00 | 10 números = R$ 1.050,00 | 11 números = R$ 2.310,00

ESTRATÉGIA:
1. Use 85-95%% do orçamento total
2. Priorize Lotofácil (mais eficiente)
3. Escolha a quantidade de números que maximiza probabilidade
4. Gere jogos com números baseados na análise histórica

FORMATO DE RESPOSTA (JSON apenas):
{
  "strategy": {
    "budget": %.2f,
    "totalCost": [CUSTO TOTAL - ENTRE 85-95%% DO ORÇAMENTO],
    "games": [
      {
        "type": "lotofacil",
        "numbers": [15 A 20 NÚMEROS ÚNICOS DE 1 A 25],
        "cost": [CUSTO EXATO]
      },
      {
        "type": "megasena", 
        "numbers": [6 A 20 NÚMEROS ÚNICOS DE 1 A 60],
        "cost": [CUSTO EXATO]
      }
    ],
    "reasoning": "Explicação da estratégia escolhida",
    "statistics": {
      "analyzedDraws": %.0f,
      "hotNumbers": [NÚMEROS MAIS FREQUENTES],
      "coldNumbers": [NÚMEROS MENOS FREQUENTES]
    }
  },
  "confidence": 0.9
}

IMPORTANTE:
- Use "lotofacil" e "megasena" (sem hífen)
- Números devem estar na faixa correta (1-25 para Lotofácil, 1-60 para Mega-Sena)
- Quantidade de números deve estar no mínimo/máximo permitido
- Custo deve usar pelo menos 85%% do orçamento
- Retorne APENAS o JSON, sem texto adicional`,
		budget, statisticalAnalysis, budget, float64(len(request.Draws)))

	return prompt
}

// max função auxiliar para retornar o maior valor
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TestConnection testa conectividade com Claude
func (c *ClaudeClient) TestConnection() error {
	testReq := ClaudeRequest{
		Model:     c.model,
		MaxTokens: 10,
		Messages: []Message{
			{
				Role:    "user",
				Content: "Teste de conectividade. Responda apenas: OK",
			},
		},
	}

	reqBody, err := json.Marshal(testReq)
	if err != nil {
		return fmt.Errorf("erro ao serializar teste: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("erro ao criar requisição teste: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro na requisição teste: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Claude API retornou status %d", resp.StatusCode)
	}

	return nil
}

// analyzeHistoricalData realiza análise estatística rigorosa dos dados históricos REAIS
func (c *ClaudeClient) analyzeHistoricalData(draws []lottery.Draw, lotteryTypes []lottery.LotteryType) string {
	if len(draws) == 0 {
		return "ERRO: Nenhum dado histórico disponível para análise."
	}

	analysis := strings.Builder{}
	analysis.WriteString(fmt.Sprintf("📊 ANÁLISE DE %d SORTEIOS REAIS:\n\n", len(draws)))

	// Separar dados por tipo de loteria
	megaDraws := []lottery.Draw{}
	lotoDraws := []lottery.Draw{}

	for _, draw := range draws {
		numbers := draw.Numbers.ToIntSlice()
		if len(numbers) == 6 { // Mega-Sena
			megaDraws = append(megaDraws, draw)
		} else if len(numbers) >= 15 { // Lotofácil
			lotoDraws = append(lotoDraws, draw)
		}
	}

	// Analisar Mega-Sena
	if len(megaDraws) > 0 {
		analysis.WriteString("🎰 MEGA-SENA - FREQUÊNCIAS REAIS:\n")
		megaFreq := calculateNumberFrequency(megaDraws, 60)
		megaHot, megaCold := getHotColdNumbers(megaFreq, 10)
		megaSums := calculateSumDistribution(megaDraws)
		megaPairs := calculatePairImparDistribution(megaDraws)
		megaSumMin, megaSumMax := getMostCommonSumRange(megaSums)

		analysis.WriteString(fmt.Sprintf("• Sorteios analisados: %d\n", len(megaDraws)))
		analysis.WriteString(fmt.Sprintf("• Números MAIS frequentes: %v\n", megaHot))
		analysis.WriteString(fmt.Sprintf("• Números MENOS frequentes: %v\n", megaCold))
		analysis.WriteString(fmt.Sprintf("• Soma mais comum: %d-%d\n", megaSumMin, megaSumMax))
		analysis.WriteString(fmt.Sprintf("• Distribuição Par/Ímpar: %.1f%% pares\n", megaPairs))
		analysis.WriteString(fmt.Sprintf("• Faixas por frequência:\n"))
		analysis.WriteString(fmt.Sprintf("  - 1-15: %v\n", getNumbersInRange(megaHot, 1, 15)))
		analysis.WriteString(fmt.Sprintf("  - 16-30: %v\n", getNumbersInRange(megaHot, 16, 30)))
		analysis.WriteString(fmt.Sprintf("  - 31-45: %v\n", getNumbersInRange(megaHot, 31, 45)))
		analysis.WriteString(fmt.Sprintf("  - 46-60: %v\n", getNumbersInRange(megaHot, 46, 60)))
		analysis.WriteString("\n")
	}

	// Analisar Lotofácil
	if len(lotoDraws) > 0 {
		analysis.WriteString("🍀 LOTOFÁCIL - FREQUÊNCIAS REAIS:\n")
		lotoFreq := calculateNumberFrequency(lotoDraws, 25)
		lotoHot, lotoCold := getHotColdNumbers(lotoFreq, 8)
		lotoSums := calculateSumDistribution(lotoDraws)
		lotoPairs := calculatePairImparDistribution(lotoDraws)
		lotoSumMin, lotoSumMax := getMostCommonSumRange(lotoSums)

		analysis.WriteString(fmt.Sprintf("• Sorteios analisados: %d\n", len(lotoDraws)))
		analysis.WriteString(fmt.Sprintf("• Números MAIS frequentes: %v\n", lotoHot))
		analysis.WriteString(fmt.Sprintf("• Números MENOS frequentes: %v\n", lotoCold))
		analysis.WriteString(fmt.Sprintf("• Soma mais comum: %d-%d\n", lotoSumMin, lotoSumMax))
		analysis.WriteString(fmt.Sprintf("• Distribuição Par/Ímpar: %.1f%% pares\n", lotoPairs))
		analysis.WriteString(fmt.Sprintf("• Quadrantes por frequência:\n"))
		analysis.WriteString(fmt.Sprintf("  - Q1 (1-6): %v\n", getNumbersInRange(lotoHot, 1, 6)))
		analysis.WriteString(fmt.Sprintf("  - Q2 (7-12): %v\n", getNumbersInRange(lotoHot, 7, 12)))
		analysis.WriteString(fmt.Sprintf("  - Q3 (13-18): %v\n", getNumbersInRange(lotoHot, 13, 18)))
		analysis.WriteString(fmt.Sprintf("  - Q4 (19-25): %v\n", getNumbersInRange(lotoHot, 19, 25)))
		analysis.WriteString("\n")
	}

	analysis.WriteString("⚡ OTIMIZAÇÃO MATEMÁTICA:\n")
	analysis.WriteString("• Lotofácil 16 números = 16 combinações por R$48 = 0.33 comb/real\n")
	analysis.WriteString("• Mega-Sena 8 números = 28 combinações por R$140 = 0.20 comb/real\n")
	analysis.WriteString("• ROI Lotofácil é 1.67x superior!\n")
	analysis.WriteString("• ESTRATÉGIA ÓTIMA: Priorizar Lotofácil para melhor custo-benefício\n\n")

	return analysis.String()
}

// calculateNumberFrequency calcula frequência de cada número nos sorteios
func calculateNumberFrequency(draws []lottery.Draw, maxNumber int) map[int]int {
	frequency := make(map[int]int)

	for _, draw := range draws {
		numbers := draw.Numbers.ToIntSlice()
		for _, num := range numbers {
			if num >= 1 && num <= maxNumber {
				frequency[num]++
			}
		}
	}

	return frequency
}

// getHotColdNumbers retorna os números mais e menos frequentes
func getHotColdNumbers(frequency map[int]int, count int) ([]int, []int) {
	type numberFreq struct {
		number int
		freq   int
	}

	var numbers []numberFreq
	for num, freq := range frequency {
		numbers = append(numbers, numberFreq{num, freq})
	}

	// Ordenar por frequência (decrescente)
	sort.Slice(numbers, func(i, j int) bool {
		return numbers[i].freq > numbers[j].freq
	})

	var hot, cold []int

	// Números mais frequentes (hot)
	for i := 0; i < count && i < len(numbers); i++ {
		hot = append(hot, numbers[i].number)
	}

	// Números menos frequentes (cold)
	for i := len(numbers) - count; i < len(numbers) && i >= 0; i++ {
		if i >= 0 {
			cold = append(cold, numbers[i].number)
		}
	}

	sort.Ints(hot)
	sort.Ints(cold)

	return hot, cold
}

// calculateSumDistribution calcula distribuição das somas dos sorteios
func calculateSumDistribution(draws []lottery.Draw) map[int]int {
	sums := make(map[int]int)

	for _, draw := range draws {
		numbers := draw.Numbers.ToIntSlice()
		sum := 0
		for _, num := range numbers {
			sum += num
		}
		sums[sum]++
	}

	return sums
}

// getMostCommonSumRange retorna a faixa de soma mais comum
func getMostCommonSumRange(sums map[int]int) (int, int) {
	if len(sums) == 0 {
		return 0, 0
	}

	// Encontrar soma mais frequente
	maxFreq := 0
	mostCommonSum := 0

	for sum, freq := range sums {
		if freq > maxFreq {
			maxFreq = freq
			mostCommonSum = sum
		}
	}

	// Retornar faixa ±10
	return mostCommonSum - 10, mostCommonSum + 10
}

// calculatePairImparDistribution calcula percentual de números pares
func calculatePairImparDistribution(draws []lottery.Draw) float64 {
	totalNumbers := 0
	pairNumbers := 0

	for _, draw := range draws {
		numbers := draw.Numbers.ToIntSlice()
		totalNumbers += len(numbers)

		for _, num := range numbers {
			if num%2 == 0 {
				pairNumbers++
			}
		}
	}

	if totalNumbers == 0 {
		return 0
	}

	return (float64(pairNumbers) / float64(totalNumbers)) * 100
}

// getNumbersInRange retorna números de uma lista que estão em uma faixa
func getNumbersInRange(numbers []int, min, max int) []int {
	var result []int

	for _, num := range numbers {
		if num >= min && num <= max {
			result = append(result, num)
		}
	}

	return result
}

// validateDiversification verifica se cada par de jogos Lotofácil tem pelo menos 8 números diferentes
func validateDiversification(games []lottery.Game) bool {
	lotofacilGames := []lottery.Game{}

	// Filtrar apenas jogos Lotofácil
	for _, game := range games {
		if game.Type == lottery.Lotofacil && len(game.Numbers) >= 15 {
			lotofacilGames = append(lotofacilGames, game)
		}
	}

	// Se menos de 2 jogos Lotofácil, não precisa validar diversificação
	if len(lotofacilGames) < 2 {
		return true
	}

	// Verificar cada par de jogos
	for i := 0; i < len(lotofacilGames); i++ {
		for j := i + 1; j < len(lotofacilGames); j++ {
			commonNumbers := getCommonNumbers(lotofacilGames[i].Numbers, lotofacilGames[j].Numbers)
			differentNumbers := len(lotofacilGames[i].Numbers) - commonNumbers

			logs.LogAI("🔍 Diversificação Jogo %d vs %d: %d números em comum, %d diferentes",
				i+1, j+1, commonNumbers, differentNumbers)

			// Regra: cada par deve ter pelo menos 8 números DIFERENTES (máximo 8 em comum)
			if commonNumbers > 8 {
				logs.LogAI("❌ FALHA na diversificação: %d números em comum (máximo permitido: 8)", commonNumbers)
				return false
			}
		}
	}

	logs.LogAI("✅ Diversificação validada com sucesso!")
	return true
}

// getCommonNumbers conta quantos números são comuns entre dois jogos
func getCommonNumbers(numbers1, numbers2 []int) int {
	numberMap := make(map[int]bool)

	// Mapear números do primeiro jogo
	for _, num := range numbers1 {
		numberMap[num] = true
	}

	// Contar números em comum
	common := 0
	for _, num := range numbers2 {
		if numberMap[num] {
			common++
		}
	}

	return common
}
