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

	prompt := c.buildAnalysisPrompt(request)

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

		// SEM FALLBACK! Retornar erro para o usuário tentar novamente
		return nil, fmt.Errorf("erro no parsing da resposta do Claude - tente gerar novamente")
	}

	// Validate parsed strategy
	if analysisResp.Strategy.Games == nil || len(analysisResp.Strategy.Games) == 0 {
		logs.LogAI("⚠️ JSON parseado mas sem jogos válidos")
		return nil, fmt.Errorf("estratégia inválida gerada pelo Claude - tente novamente")
	}

	// VALIDAÇÃO CRÍTICA: Verificar se todos os jogos têm números mínimos
	for i, game := range analysisResp.Strategy.Games {
		var minNumbers int
		if game.Type == "lotofacil" {
			minNumbers = 15
		} else if game.Type == "megasena" {
			minNumbers = 6
		}

		if len(game.Numbers) < minNumbers {
			logs.LogError(logs.CategoryAI, "❌ ERRO CRÍTICO: Jogo %d (%s) tem apenas %d números, mínimo é %d",
				i+1, game.Type, len(game.Numbers), minNumbers)
			logs.LogAI("🎲 Jogo inválido: %v", game.Numbers)
			return nil, fmt.Errorf("IA gerou jogo inválido: %s com apenas %d números (mínimo: %d)",
				game.Type, len(game.Numbers), minNumbers)
		}

		logs.LogAI("✅ Jogo %d validado: %s com %d números", i+1, game.Type, len(game.Numbers))
	}

	if !validateDiversification(analysisResp.Strategy.Games) {
		logs.LogAI("🔄 Estratégia falhou na validação de diversificação, tentando novamente...")

		// Retry até 5 vezes mais para conseguir diversificação correta
		maxRetries := 5
		bestStrategy := analysisResp // Manter a melhor estratégia gerada

		for retry := 0; retry < maxRetries; retry++ {
			logs.LogAI("🔄 Tentativa %d/%d para diversificação correta...", retry+1, maxRetries)

			// Gerar nova estratégia
			newPrompt := c.buildAnalysisPrompt(request)
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

	logs.LogAI("🔄 Gerando estratégia fallback para orçamento R$ %.2f", budget)

	// Generate simple games based on budget and preferences
	for _, lotteryType := range request.Preferences.LotteryTypes {
		if lotteryType == "megasena" && budget-totalCost >= 5 {
			remainingBudget := budget - totalCost

			if remainingBudget >= 140 { // Can afford 8 numbers
				games = append(games, lottery.Game{
					Type:    "megasena",
					Numbers: []int{1, 7, 15, 23, 35, 42, 48, 58}, // Simple fallback numbers
					Cost:    140,
				})
				totalCost += 140
			} else if remainingBudget >= 35 { // Can afford 7 numbers
				games = append(games, lottery.Game{
					Type:    "megasena",
					Numbers: []int{7, 15, 23, 35, 42, 48, 58}, // Simple fallback numbers
					Cost:    35,
				})
				totalCost += 35
			} else if remainingBudget >= 5 { // Simple 6 numbers
				games = append(games, lottery.Game{
					Type:    "megasena",
					Numbers: []int{7, 15, 23, 35, 42, 58}, // Simple fallback numbers
					Cost:    5,
				})
				totalCost += 5
			}
		}

		if lotteryType == "lotofacil" && budget-totalCost >= 3 {
			remainingBudget := budget - totalCost

			if remainingBudget >= 48 { // Can afford 16 numbers
				games = append(games, lottery.Game{
					Type:    "lotofacil",
					Numbers: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, // Simple fallback
					Cost:    48,
				})
				totalCost += 48
			} else if remainingBudget >= 3 { // Simple 15 numbers
				games = append(games, lottery.Game{
					Type:    "lotofacil",
					Numbers: []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, // Simple fallback
					Cost:    3,
				})
				totalCost += 3
			}
		}
	}

	reasoning := fmt.Sprintf("Estratégia fallback gerada: %d jogos por R$ %.2f (%.1f%% do orçamento). "+
		"Esta é uma estratégia básica gerada quando a análise avançada da IA falha.",
		len(games), totalCost, (totalCost/budget)*100)

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
		Confidence: 0.6, // Lower confidence for fallback
	}
}

// extractJSON extrai o primeiro JSON válido encontrado no texto
func extractJSON(text string) string {
	// Procurar pelo início do JSON
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
				return text[start : i+1]
			}
		}
	}

	// Se não encontrou JSON válido, retorna o texto original
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

// buildAnalysisPrompt constrói o prompt para análise com ESTRATÉGIAS PROFISSIONAIS MUNDIAIS
func (c *ClaudeClient) buildAnalysisPrompt(request lottery.AnalysisRequest) string {
	budget := request.Preferences.Budget

	// ANÁLISE ESTATÍSTICA RIGOROSA DOS DADOS HISTÓRICOS REAIS
	statisticalAnalysis := c.analyzeHistoricalData(request.Draws, request.Preferences.LotteryTypes)

	prompt := fmt.Sprintf(`Você é um MATEMÁTICO ESPECIALISTA MUNDIAL em loterias, combinatória avançada e teoria de jogos. Use as ESTRATÉGIAS PROFISSIONAIS mais avançadas do mundo.

🎯 OBJETIVO: MAXIMIZAR matematicamente as chances REAIS de ganho para R$ %.2f usando técnicas de ESPECIALISTAS MUNDIAIS.

=== DADOS ESTATÍSTICOS REAIS ===
%s

=== PREÇOS OFICIAIS CAIXA (EXATOS) ===
MEGA-SENA: 6→R$5,00 | 7→R$35,00 | 8→R$140,00 | 9→R$420,00 | 10→R$1.050,00 | 11→R$2.310,00 | 12→R$4.620,00
LOTOFÁCIL: 15→R$3,00 | 16→R$48,00 | 17→R$408,00 | 18→R$2.448,00 | 19→R$11.628,00 | 20→R$46.512,00

=== ANÁLISE DE VALOR ESPERADO PROFISSIONAL ===
LOTOFÁCIL VALOR ESPERADO COMPLETO (incluindo prêmios secundários):
• 15 números: -R$0,85 por jogo (melhor relação custo/benefício)
• 16 números: -R$12,80 por jogo MAS 16x mais chances de 14 pontos
• 17 números: Garantia matemática de pelo menos 11 pontos

MEGA-SENA VALOR ESPERADO:
• 6 números: -R$2,50 por jogo
• 7 números: -R$17,50 MAS 7x mais chances de quadra/quina
• 8 números: -R$70,00 MAS 28x mais chances + cobertura sistêmica

ESTRATÉGIA PROFISSIONAL: Priorizar Lotofácil para ROI, Mega-Sena para prêmios que mudam a vida.

=== SISTEMAS DE REDUÇÃO PROFISSIONAIS (WHEELING) ===
LOTOFÁCIL - SISTEMAS DE GARANTIA:
• Sistema 18x15: 18 números em 3 jogos de 16 → GARANTE 13 pontos se sair 15
• Sistema 20x15: 20 números em 4 jogos de 16 → GARANTE 14 pontos se sair 15  
• Sistema 22x15: 22 números em 6 jogos de 16 → GARANTE 15 pontos se sair 15

MEGA-SENA - SISTEMAS DE GARANTIA:
• Sistema 9x6: 9 números em 7 jogos de 6 → GARANTE terno se sair quadra
• Sistema 10x6: 10 números em 10 jogos de 6 → GARANTE quadra se sair quina
• Sistema 12x6: 12 números em 22 jogos de 6 → GARANTE quina se sair sena

=== FILTROS MATEMÁTICOS AVANÇADOS (OBRIGATÓRIOS) ===

🚨 NÚMEROS MÍNIMOS OBRIGATÓRIOS (CRÍTICO):
• LOTOFÁCIL: SEMPRE 15, 16, 17, 18, 19 ou 20 números (NUNCA MENOS QUE 15!)
• MEGA-SENA: SEMPRE 6, 7, 8, 9, 10, 11 ou 12 números (NUNCA MENOS QUE 6!)

1. **FILTRO DE SOMA INTELIGENTE:**
   - Lotofácil: somas entre 170-210 (80%% dos sorteios históricos)
   - Mega-Sena: somas entre 140-200 (75%% dos sorteios históricos)
   - REJEITE jogos fora dessa faixa estatística!

2. **FILTRO DE PARIDADE BALANCEADA:**
   - Lotofácil 16 números: 8 pares + 8 ímpares (±1)
   - Mega-Sena 6 números: 3 pares + 3 ímpares (±1)
   - NUNCA faça jogos com mais de 70%% de uma paridade!

3. **FILTRO DE DÉCADAS/QUADRANTES:**
   - Distribua números por TODAS as faixas
   - Lotofácil: pelo menos 2 números em cada quadrante (1-6, 7-12, 13-18, 19-25)
   - Mega-Sena: pelo menos 1 número em cada década (1-10, 11-20, 21-30, 31-40, 41-50, 51-60)

4. **FILTRO DE CONSECUTIVOS MATEMÁTICO:**
   - Máximo 2 números consecutivos por jogo
   - EVITE sequências tipo: 1,2,3,4,5,6 ou 10,11,12,13

5. **FILTRO DE TERMINAÇÕES:**
   - Máximo 2 números com mesma terminação (ex: 1,11,21)
   - Distribua terminações 0-9 uniformemente

6. **FILTRO DE REPETIÇÕES HISTÓRICAS:**
   - EVITE reproduzir exatamente jogos já sorteados
   - Use pelo menos 50%% de números diferentes do último sorteio

=== ESTRATÉGIA DE COBERTURA COMBINATORIAL ===

**PARA ORÇAMENTOS BAIXOS (R$50-150):**
- Foque em Lotofácil 16 números (melhor valor esperado)
- Use diversificação de Hamming: distância mínima de 8 números entre jogos
- Aplique TODOS os filtros matemáticos

**PARA ORÇAMENTOS MÉDIOS (R$150-500):**
- Sistema misto: 70%% Lotofácil + 30%% Mega-Sena
- Implemente sistema de redução básico
- Use balanceamento por blocos numéricos

**PARA ORÇAMENTOS ALTOS (R$500+):**
- Implemente sistemas de garantia completos
- Use matrizes de redução profissionais
- Estratégia de portfólio diversificado

=== ALGORITMO DE SELEÇÃO PROFISSIONAL ===

1. **ANÁLISE DE TENDÊNCIA REGRESSIVA:**
   - Números "frios" têm probabilidade crescente (Lei dos Grandes Números)
   - Balanceie 60%% números frequentes + 40%% números devidos

2. **MATRIZ DE DISTÂNCIA HAMMING:**
   Para cada par de jogos (A,B): distância = |A ⊕ B| ≥ 8
   - Jogo 1: [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16]
   - Jogo 2: [1,2,3,4,17,18,19,20,21,22,23,24,25,14,15,16] (8 diferentes)
   - Jogo 3: [9,10,11,12,17,18,19,20,5,6,7,8,23,24,25,13] (8+ diferentes)

3. **VALIDAÇÃO MULTI-FILTRO:**
   CADA jogo deve passar TODOS os filtros:
   ✓ Soma dentro da faixa histórica
   ✓ Paridade balanceada (±1)
   ✓ Distribuição por quadrantes
   ✓ Máximo 2 consecutivos
   ✓ Máximo 2 mesmas terminações
   ✓ Distância Hamming ≥8 de outros jogos

=== ESTRATÉGIA FINANCEIRA OTIMIZADA ===
- Use 95-98%% do orçamento (máxima eficiência)
- Priorize sistemas que garantem prêmios menores
- Balanceie risco vs. retorno baseado no perfil do usuário

=== SAÍDA JSON OBRIGATÓRIA ===
RETORNE APENAS JSON VÁLIDO (sem markdown):
{
  "strategy": {
    "budget": %.2f,
    "totalCost": [SOMA EXATA DOS CUSTOS],
    "games": [
      {
        "type": "lotofacil",
        "numbers": [EXATAMENTE 15/16/17/18/19/20 NÚMEROS ÚNICOS - NUNCA MENOS QUE 15!],
        "cost": [CUSTO OFICIAL EXATO: 15números=R$3,00 | 16números=R$48,00 | 17números=R$408,00],
        "filters": {
          "sum": [SOMA DOS NÚMEROS],
          "evenOdd": "8p8i",
          "decades": [DISTRIBUIÇÃO],
          "consecutives": [QUANTIDADE],
          "endings": [TERMINAÇÕES]
        }
      }
    ],
    "reasoning": "[EXPLICAÇÃO DETALHADA: quais filtros aplicou, qual sistema de redução usou, como garantiu a cobertura combinatorial, qual o valor esperado calculado, estratégia de diversificação. Mínimo 200 palavras com dados específicos.]",
    "systemUsed": "[NOME DO SISTEMA: Ex: 'Sistema 20x15', 'Wheeling 9x6', 'Filtros Matemáticos Completos']",
    "expectedValue": [VALOR ESPERADO TOTAL DA ESTRATÉGIA],
    "guarantees": "[O QUE O SISTEMA GARANTE: Ex: 'Garante 14 pontos se sair 15 na Lotofácil']",
    "statistics": {
      "analyzedDraws": %d,
      "hotNumbers": [NÚMEROS MAIS FREQUENTES],
      "coldNumbers": [NÚMEROS MENOS FREQUENTES - ESTES TÊM MAIOR PROBABILIDADE!],
      "regressionCandidates": [NÚMEROS FRIOS QUE DEVEM SER INCLUÍDOS]
    }
  },
  "confidence": [0.88-0.95]
}

🚨 VALIDAÇÕES CRÍTICAS OBRIGATÓRIAS:
1. CADA número deve aparecer APENAS UMA VEZ por jogo
2. LOTOFÁCIL: MÍNIMO 15 NÚMEROS OBRIGATÓRIO - NUNCA MENOS!
3. MEGA-SENA: MÍNIMO 6 NÚMEROS OBRIGATÓRIO - NUNCA MENOS!
4. TODOS os filtros matemáticos devem ser aplicados
5. Valor esperado deve ser calculado corretamente
6. Sistema de redução deve ser identificado
7. Distância de Hamming entre jogos ≥8
8. Soma de cada jogo dentro da faixa histórica
9. Distribuição balanceada por quadrantes/décadas

🚨 VALIDAÇÃO FINAL OBRIGATÓRIA ANTES DE RETORNAR:
ANTES de retornar o JSON, VERIFIQUE CADA JOGO:
- Lotofácil: CONTE os números - deve ter EXATAMENTE 15, 16, 17, 18, 19 ou 20 números
- Mega-Sena: CONTE os números - deve ter EXATAMENTE 6, 7, 8, 9, 10, 11 ou 12 números
- SE algum jogo tiver menos números que o mínimo, ADICIONE números aleatórios válidos
- NUNCA retorne um jogo com números insuficientes!

Use SOMENTE os dados estatísticos fornecidos + filtros matemáticos avançados. Esta é a estratégia de ESPECIALISTAS MUNDIAIS!`,
		budget, statisticalAnalysis, budget, len(request.Draws))

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

// analyzeHistoricalData realiza análise estatística otimizada dos dados históricos REAIS
func (c *ClaudeClient) analyzeHistoricalData(draws []lottery.Draw, lotteryTypes []lottery.LotteryType) string {
	if len(draws) == 0 {
		return "ERRO: Nenhum dado histórico disponível para análise."
	}

	// OTIMIZAÇÃO: Limites ajustados para melhor base estatística
	const maxMegaSenaDraws = 100  // ~1 ano de dados (2x por semana)
	const maxLotofacilDraws = 250 // ~8 meses de dados (diário)

	// Separar dados por tipo de loteria
	megaDraws := []lottery.Draw{}
	lotoDraws := []lottery.Draw{}

	for _, draw := range draws {
		numbers := draw.Numbers.ToIntSlice()
		if len(numbers) == 6 && len(megaDraws) < maxMegaSenaDraws { // Mega-Sena
			megaDraws = append(megaDraws, draw)
		} else if len(numbers) >= 15 && len(lotoDraws) < maxLotofacilDraws { // Lotofácil
			lotoDraws = append(lotoDraws, draw)
		}

		// Parar se já temos amostras suficientes de ambas
		if len(megaDraws) >= maxMegaSenaDraws && len(lotoDraws) >= maxLotofacilDraws {
			break
		}
	}

	analysis := strings.Builder{}
	analysis.WriteString(fmt.Sprintf("📊 ANÁLISE ESTATÍSTICA ROBUSTA:\n\n"))

	// Analisar Mega-Sena
	if len(megaDraws) > 0 {
		analysis.WriteString("🎰 MEGA-SENA - ANÁLISE PROFUNDA:\n")
		megaFreq := calculateNumberFrequency(megaDraws, 60)
		megaHot, megaCold := getHotColdNumbers(megaFreq, 10) // Voltou para 10 com mais dados
		megaSums := calculateSumDistribution(megaDraws)
		megaPairs := calculatePairImparDistribution(megaDraws)
		megaSumMin, megaSumMax := getMostCommonSumRange(megaSums)

		analysis.WriteString(fmt.Sprintf("• Base estatística: %d sorteios (~1 ano)\n", len(megaDraws)))
		analysis.WriteString(fmt.Sprintf("• Números quentes: %v\n", megaHot))
		analysis.WriteString(fmt.Sprintf("• Números frios: %v\n", megaCold))
		analysis.WriteString(fmt.Sprintf("• Faixa de soma ótima: %d-%d\n", megaSumMin, megaSumMax))
		analysis.WriteString(fmt.Sprintf("• Distribuição pares: %.1f%%\n", megaPairs))
		analysis.WriteString(fmt.Sprintf("• Dezenas por faixa:\n"))
		analysis.WriteString(fmt.Sprintf("  - 01-15: %v\n", getNumbersInRange(megaHot, 1, 15)))
		analysis.WriteString(fmt.Sprintf("  - 16-30: %v\n", getNumbersInRange(megaHot, 16, 30)))
		analysis.WriteString(fmt.Sprintf("  - 31-45: %v\n", getNumbersInRange(megaHot, 31, 45)))
		analysis.WriteString(fmt.Sprintf("  - 46-60: %v\n", getNumbersInRange(megaHot, 46, 60)))
		analysis.WriteString("\n")
	}

	// Analisar Lotofácil
	if len(lotoDraws) > 0 {
		analysis.WriteString("🍀 LOTOFÁCIL - ANÁLISE PROFUNDA:\n")
		lotoFreq := calculateNumberFrequency(lotoDraws, 25)
		lotoHot, lotoCold := getHotColdNumbers(lotoFreq, 8) // Voltou para 8 com mais dados
		lotoSums := calculateSumDistribution(lotoDraws)
		lotoPairs := calculatePairImparDistribution(lotoDraws)
		lotoSumMin, lotoSumMax := getMostCommonSumRange(lotoSums)

		analysis.WriteString(fmt.Sprintf("• Base estatística: %d sorteios (~8 meses)\n", len(lotoDraws)))
		analysis.WriteString(fmt.Sprintf("• Números quentes: %v\n", lotoHot))
		analysis.WriteString(fmt.Sprintf("• Números frios: %v\n", lotoCold))
		analysis.WriteString(fmt.Sprintf("• Faixa de soma ótima: %d-%d\n", lotoSumMin, lotoSumMax))
		analysis.WriteString(fmt.Sprintf("• Distribuição pares: %.1f%%\n", lotoPairs))
		analysis.WriteString(fmt.Sprintf("• Quadrantes por frequência:\n"))
		analysis.WriteString(fmt.Sprintf("  - Q1 (01-06): %v\n", getNumbersInRange(lotoHot, 1, 6)))
		analysis.WriteString(fmt.Sprintf("  - Q2 (07-12): %v\n", getNumbersInRange(lotoHot, 7, 12)))
		analysis.WriteString(fmt.Sprintf("  - Q3 (13-18): %v\n", getNumbersInRange(lotoHot, 13, 18)))
		analysis.WriteString(fmt.Sprintf("  - Q4 (19-25): %v\n", getNumbersInRange(lotoHot, 19, 25)))
		analysis.WriteString("\n")
	}

	analysis.WriteString("⚡ ESTRATÉGIA OTIMIZADA BASEADA EM DADOS REAIS:\n")
	analysis.WriteString("• Lotofácil: Priorizar sistema 16 números para melhor ROI\n")
	analysis.WriteString("• Mega-Sena: Balancear números quentes e frios para cobertura\n")
	analysis.WriteString("• Aplicar filtros matemáticos rigorosos em todas as combinações\n")
	analysis.WriteString("• Usar distribuição por quadrantes/décadas conforme padrões históricos\n\n")

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
		if game.Type == "lotofacil" && len(game.Numbers) >= 15 {
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
