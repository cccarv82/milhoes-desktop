package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"lottery-optimizer-gui/internal/config"
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
	// DEBUG: Logs detalhados sobre o cliente
	fmt.Printf("🔍 [CLAUDE DEBUG] Iniciando AnalyzeStrategy...\n")
	fmt.Printf("🔍 [CLAUDE DEBUG] API Key length: %d\n", len(c.apiKey))
	if c.apiKey != "" {
		fmt.Printf("🔍 [CLAUDE DEBUG] API Key prefix: %s\n", c.apiKey[:min(10, len(c.apiKey))])
	} else {
		fmt.Printf("🔍 [CLAUDE DEBUG] API Key VAZIA! ❌\n")
		// Se não tem chave, retornar erro ao invés de continuar
		return nil, fmt.Errorf("chave da API do Claude não configurada")
	}
	fmt.Printf("🔍 [CLAUDE DEBUG] Model: %s\n", c.model)
	fmt.Printf("🔍 [CLAUDE DEBUG] MaxTokens: %d\n", c.maxTokens)
	fmt.Printf("🔍 [CLAUDE DEBUG] BaseURL: %s\n", c.baseURL)
	
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
		fmt.Printf("🔍 [CLAUDE DEBUG] Erro ao serializar: %v\n", err)
		return nil, fmt.Errorf("erro ao serializar requisição: %w", err)
	}
	
	fmt.Printf("🔍 [CLAUDE DEBUG] Request body preparado. Size: %d bytes\n", len(reqBody))
	
	// Implementar retry logic com exponential backoff
	var resp *http.Response
	maxRetries := 3
	baseDelay := 2 * time.Second
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(reqBody))
		if err != nil {
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
					fmt.Printf("⚠️  Tentativa %d falhou, tentando novamente em %v...\n", attempt+1, delay)
				}
				time.Sleep(delay)
				continue
			}
			return nil, fmt.Errorf("erro na requisição após %d tentativas: %w", maxRetries, err)
		}
		break
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API retornou status %d", resp.StatusCode)
	}
	
	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}
	
	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("resposta vazia do Claude")
	}
	
	// Extrair JSON limpo da resposta
	rawResponse := claudeResp.Content[0].Text
	jsonContent := extractJSON(rawResponse)
	
	// Enhanced debug logging
	if config.IsVerbose() {
		fmt.Printf("🤖 Resposta COMPLETA do Claude:\n%s\n", rawResponse)
		fmt.Printf("🔍 JSON extraído:\n%s\n", jsonContent)
	} else {
		fmt.Printf("🤖 Resposta do Claude: %s\n", rawResponse[:min(200, len(rawResponse))]+"...")
		fmt.Printf("🔍 JSON extraído: %s\n", jsonContent[:min(200, len(jsonContent))]+"...")
	}
	
	// Parsear a resposta JSON do Claude
	var analysisResp lottery.AnalysisResponse
	if err := json.Unmarshal([]byte(jsonContent), &analysisResp); err != nil {
		fmt.Printf("❌ Erro ao fazer parse do JSON: %v\n", err)
		fmt.Printf("📄 JSON que falhou: %s\n", jsonContent)
		
		// SEM FALLBACK! Retornar erro para o usuário tentar novamente
		return nil, fmt.Errorf("erro no parsing da resposta do Claude - tente gerar novamente")
	} else {
		// Validate parsed strategy
		if analysisResp.Strategy.Games == nil || len(analysisResp.Strategy.Games) == 0 {
			fmt.Printf("⚠️ JSON parseado mas sem jogos válidos\n")
			return nil, fmt.Errorf("estratégia inválida gerada pelo Claude - tente novamente")
		} else {
			// VALIDAÇÃO DE DIVERSIFICAÇÃO CRÍTICA
			if !validateDiversification(analysisResp.Strategy.Games) {
				fmt.Printf("🔄 Estratégia falhou na validação de diversificação, tentando novamente...\n")
				
				// Retry até 5 vezes mais para conseguir diversificação correta
				maxRetries := 5
				bestStrategy := analysisResp // Manter a melhor estratégia gerada
				
				for retry := 0; retry < maxRetries; retry++ {
					fmt.Printf("🔄 Tentativa %d/%d para diversificação correta...\n", retry+1, maxRetries)
					
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
							fmt.Printf("✅ Diversificação correta conseguida na tentativa %d!\n", retry+1)
							analysisResp = newAnalysisResp
							break
						} else {
							// Manter a estratégia com melhor orçamento/qualidade
							if newAnalysisResp.Strategy.TotalCost > bestStrategy.Strategy.TotalCost {
								bestStrategy = newAnalysisResp
								fmt.Printf("💡 Nova melhor estratégia encontrada: R$ %.2f\n", newAnalysisResp.Strategy.TotalCost)
							}
						}
					}
				}
				
				// Se não conseguiu diversificação perfeita, usar a MELHOR estratégia do Claude
				if !validateDiversification(analysisResp.Strategy.Games) {
					fmt.Printf("💪 Usando MELHOR estratégia Claude (sem fallback!): R$ %.2f - Qualidade superior!\n", bestStrategy.Strategy.TotalCost)
					analysisResp = bestStrategy
					analysisResp.Confidence = analysisResp.Confidence * 0.9 // Reduzir confiança ligeiramente
				}
			} else {
				fmt.Printf("✅ JSON parseado com sucesso: %d jogos gerados\n", len(analysisResp.Strategy.Games))
			}
		}
	}
	
	if config.IsVerbose() {
		fmt.Printf("Tokens usados: %d input + %d output = %d total\n", 
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
	
	fmt.Printf("🔄 Gerando estratégia fallback para orçamento R$ %.2f\n", budget)
	
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
			Budget:     budget,
			TotalCost:  totalCost,
			Games:      games,
			Reasoning:  reasoning,
			CreatedAt:  time.Now(),
			Statistics: lottery.Stats{
				AnalyzedDraws: len(request.Draws),
				HotNumbers:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, // Simple defaults
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

// buildAnalysisPrompt constrói o prompt para análise com DADOS ESTATÍSTICOS REAIS
func (c *ClaudeClient) buildAnalysisPrompt(request lottery.AnalysisRequest) string {
	budget := request.Preferences.Budget
	
	// ANÁLISE ESTATÍSTICA RIGOROSA DOS DADOS HISTÓRICOS REAIS
	statisticalAnalysis := c.analyzeHistoricalData(request.Draws, request.Preferences.LotteryTypes)
	
	prompt := fmt.Sprintf(`Você é um MATEMÁTICO ESPECIALISTA em otimização de loterias. Use APENAS os dados estatísticos REAIS fornecidos abaixo.

🎯 OBJETIVO: MAXIMIZAR matematicamente as chances de ganho para R$ %.2f

=== DADOS ESTATÍSTICOS REAIS ===
%s

=== OTIMIZAÇÃO MATEMÁTICA OBRIGATÓRIA ===
MEGA-SENA ROI: 6num=1comb/R$5(0.2), 7num=7comb/R$35(0.2), 8num=28comb/R$140(0.2), 9num=84comb/R$420(0.2)
LOTOFÁCIL ROI: 15num=1comb/R$3(0.33), 16num=8008comb/R$48(166.8!), 17num=19448comb/R$408(47.7)

MATEMATICAMENTE: Lotofácil 16 números tem ROI 834x melhor que Mega-Sena!

=== ESTRATÉGIA OBRIGATÓRIA ===
1. PRIORIZE Lotofácil 16+ números (ROI máximo)
2. Use 90-95%% do orçamento (nunca menos)
3. Máximo 4 jogos totais
4. Use APENAS os números das estatísticas reais fornecidas
5. Escolha baseado nas FREQUÊNCIAS REAIS calculadas
6. **DIVERSIFICAÇÃO OBRIGATÓRIA**: Cada jogo deve ter pelo menos 50%% de números DIFERENTES dos outros jogos
7. **COBERTURA MÁXIMA**: Distribua os números mais frequentes entre TODOS os jogos, não concentre

=== PARA MEGA-SENA ===
- Use preferencialmente os números MAIS FREQUENTES dos dados reais
- Evite números MENOS FREQUENTES 
- Distribua pelas faixas: 1-15, 16-30, 31-45, 46-60
- Somas históricas mais comuns: 150-200

=== PARA LOTOFÁCIL ===
- Use números das MAIORES FREQUÊNCIAS dos dados reais
- Distribua pelos quadrantes: 1-6, 7-12, 13-18, 19-25
- Somas históricas mais comuns: 180-220
- PRIORIZE jogos de 16 números (ROI 166.8 comb/real)
- **DIVERSIFIQUE**: Se fizer múltiplos jogos, cada um deve cobrir DIFERENTES combinações dos números frequentes
- **ESTRATÉGIA DE COBERTURA**: 
  * Jogo 1: Primeiros 8 mais frequentes + 8 complementares
  * Jogo 2: Próximos 8 mais frequentes + 8 complementares diferentes
  * Jogo 3: Misture os mais frequentes de forma diferente

=== DIVERSIFICAÇÃO OBRIGATÓRIA (CRÍTICO PARA 10/10) ===
🚨 PRIORIDADE MÁXIMA: DIVERSIFICAÇÃO > FREQUÊNCIA PURA
Se gerar múltiplos jogos Lotofácil:
- Jogo 1 vs Jogo 2: pelo menos 8 números DIFERENTES
- Jogo 1 vs Jogo 3: pelo menos 8 números DIFERENTES  
- Jogo 2 vs Jogo 3: pelo menos 8 números DIFERENTES
- REGRA: CADA PAR de jogos deve ter no máximo 8 números em comum

ALGORITMO DE DIVERSIFICAÇÃO:
1. Selecione os 24 números mais frequentes dos dados reais
2. DISTRIBUA estes 24 números entre os 3 jogos de forma balanceada
3. Jogo 1: Use números 1-8 + complementares 17-24 (16 total)
4. Jogo 2: Use números 1-8 + complementares 9-16 (16 total) 
5. Jogo 3: Use números 9-16 + complementares 17-24 (16 total)
6. VERIFIQUE: cada par tem exatamente 8 números em comum

EXEMPLO MATEMÁTICO OBRIGATÓRIO:
- Frequentes: [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24]
- Jogo 1: [1,2,3,4,5,6,7,8,17,18,19,20,21,22,23,24] 
- Jogo 2: [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16]
- Jogo 3: [9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24]
- Verificação: J1∩J2={1,2,3,4,5,6,7,8}=8 ✅, J1∩J3={17,18,19,20,21,22,23,24}=8 ✅, J2∩J3={9,10,11,12,13,14,15,16}=8 ✅

=== VALIDAÇÃO MATEMÁTICA OBRIGATÓRIA ===
Para CADA PAR de jogos Lotofácil (A,B):
- Intersecção |A ∩ B| ≤ 8 números
- Diferença |A - B| ≥ 8 números
- Diferença |B - A| ≥ 8 números

CÁLCULO MATEMÁTICO:
- Jogo A: [a1,a2,a3,...,a16] 
- Jogo B: [b1,b2,b3,...,b16]
- Conte números comuns: quantos ai estão também em B?
- MÁXIMO PERMITIDO: 8 números comuns
- MÍNIMO EXIGIDO: 8 números diferentes

EXEMPLO DE CÁLCULO:
- A=[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16]
- B=[1,2,3,4,5,6,7,8,17,18,19,20,21,22,23,24] 
- Comuns: {1,2,3,4,5,6,7,8} = 8 ✅
- Diferentes em A: {9,10,11,12,13,14,15,16} = 8 ✅
- Diferentes em B: {17,18,19,20,21,22,23,24} = 8 ✅

VERIFIQUE MATEMÁTICA ANTES DE RETORNAR!

RETORNE APENAS JSON (sem markdown):
{
  "strategy": {
    "budget": %.2f,
    "totalCost": [CALCULE EXATO],
    "games": [
      {
        "type": "lotofacil",
        "numbers": [EXATAMENTE 15 OU 16 NÚMEROS ÚNICOS DIFERENTES - SEM REPETIÇÃO],
        "cost": [CUSTO EXATO - R$3 para 15 números, R$48 para 16 números]
      },
      {
        "type": "megasena", 
        "numbers": [EXATAMENTE 6, 7 OU 8 NÚMEROS ÚNICOS DIFERENTES - SEM REPETIÇÃO],
        "cost": [CUSTO EXATO - R$5 para 6, R$35 para 7, R$140 para 8 números]
      }
    ],
    "reasoning": "[EXPLIQUE detalhadamente: quais números das estatísticas reais escolheu, por que essas frequências específicas, como otimizou o ROI matemático, qual a distribuição por quadrantes, como chegou no percentual do orçamento. Mínimo 150 palavras com dados específicos das frequências reais.]",
    "statistics": {
      "analyzedDraws": %d,
      "hotNumbers": [ARRAY SIMPLES DOS NÚMEROS MAIS FREQUENTES - TODOS MISTURADOS],
      "coldNumbers": [ARRAY SIMPLES DOS NÚMEROS MENOS FREQUENTES - TODOS MISTURADOS]
    }
  },
  "confidence": [0.85-0.95]
}

CRÍTICO: 
- CADA JOGO deve ser um OBJETO com "type", "numbers", "cost"
- NUNCA use arrays simples de números para games
- NUNCA repita números no mesmo jogo - cada número deve aparecer APENAS UMA VEZ
- Para Lotofácil: números de 1 a 25, exatamente 15 ou 16 números únicos
- Para Mega-Sena: números de 1 a 60, exatamente 6, 7 ou 8 números únicos
- hotNumbers e coldNumbers devem ser ARRAYS SIMPLES de números: [1,2,3,4,5]
- NÃO usar objetos aninhados como {"megasena": [...], "lotofacil": [...]}
- **DIVERSIFICAÇÃO CRÍTICA**: Se gerar múltiplos jogos Lotofácil, cada um deve ter pelo menos 8 números DIFERENTES dos outros
- **EXEMPLO DE DIVERSIFICAÇÃO CORRETA**:
  * Jogo Lotofácil 1: [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16] 
  * Jogo Lotofácil 2: [1,2,3,4,17,18,19,20,21,22,23,24,25,14,15,16] (8 números diferentes)
  * Jogo Lotofácil 3: [1,2,9,10,17,18,19,20,5,6,23,24,25,13,14,15] (8+ números diferentes)
- Use SOMENTE as frequências e padrões dos dados reais fornecidos. NÃO INVENTE números!`, 
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
	analysis.WriteString("• Lotofácil 16 números = 8.008 combinações por R$48 = 166.8 comb/real\n")
	analysis.WriteString("• Mega-Sena 8 números = 28 combinações por R$140 = 0.2 comb/real\n")
	analysis.WriteString("• ROI Lotofácil é 834x superior!\n")
	analysis.WriteString("• ESTRATÉGIA ÓTIMA: Priorizar Lotofácil 16+ números\n\n")
	
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
			
			fmt.Printf("🔍 Diversificação Jogo %d vs %d: %d números em comum, %d diferentes\n", 
				i+1, j+1, commonNumbers, differentNumbers)
			
			// Regra: cada par deve ter pelo menos 8 números DIFERENTES (máximo 8 em comum)
			if commonNumbers > 8 {
				fmt.Printf("❌ FALHA na diversificação: %d números em comum (máximo permitido: 8)\n", commonNumbers)
				return false
			}
		}
	}
	
	fmt.Printf("✅ Diversificação validada com sucesso!\n")
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