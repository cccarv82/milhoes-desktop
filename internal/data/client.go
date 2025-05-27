package data

import (
	"encoding/json"
	"fmt"
	"lottery-optimizer-gui/internal/config"
	"lottery-optimizer-gui/internal/logs"
	"lottery-optimizer-gui/internal/lottery"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client cliente para APIs de loterias
type Client struct {
	client       *resty.Client
	baseURL      string
	cacheManager *CacheManager
}

// NewClient cria um novo cliente para APIs de loterias
func NewClient() *Client {
	client := resty.New()
	client.SetTimeout(60 * time.Second) // Aumentado para 60 segundos
	client.SetRetryCount(3)
	client.SetRetryWaitTime(2 * time.Second)

	// Headers para parecer mais com um browser real
	client.SetHeaders(map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "pt-BR,pt;q=0.9,en;q=0.8",
		"Accept-Encoding": "gzip, deflate, br",
		"Cache-Control":   "no-cache",
		"Pragma":          "no-cache",
		"Sec-Fetch-Dest":  "empty",
		"Sec-Fetch-Mode":  "cors",
		"Sec-Fetch-Site":  "cross-site",
		"Referer":         "https://loterias.caixa.gov.br/",
		"Origin":          "https://loterias.caixa.gov.br",
	})

	return &Client{
		client:       client,
		baseURL:      config.GlobalConfig.App.DataSourceURL,
		cacheManager: NewCacheManager(),
	}
}

// GetLatestDraws busca os últimos sorteios de uma loteria com sistema de cache
func (c *Client) GetLatestDraws(ltype lottery.LotteryType, count int) ([]lottery.Draw, error) {
	// Tentar buscar da API primeiro
	draws, err := c.fetchFromAPI(ltype, count)

	if err == nil && len(draws) >= count/2 {
		// API funcionou, salvar no cache
		if cacheErr := c.cacheManager.SaveToCache(ltype, draws, count); cacheErr != nil {
			if config.IsVerbose() {
				logs.LogData("⚠️ Erro ao salvar cache: %v", cacheErr)
			}
		}
		return draws, nil
	}

	// API falhou, tentar cache
	if config.IsVerbose() {
		logs.LogData("⚠️ API falhou para %s: %v", ltype, err)
		logs.LogData("🔍 Verificando cache...")
	}

	cachedDraws, hasCachedData := c.cacheManager.LoadFromCache(ltype, count)

	if hasCachedData {
		cacheTime, _, _ := c.cacheManager.GetCacheInfo(ltype)
		logs.LogData("📋 Usando dados do cache para %s (salvos em %s)",
			ltype, cacheTime.Format("02/01/2006 15:04"))
		return cachedDraws, nil
	}

	// Sem API e sem cache válido
	return nil, fmt.Errorf("API da CAIXA indisponível e cache não encontrado ou expirado")
}

// fetchFromAPI busca dados diretamente da API
func (c *Client) fetchFromAPI(ltype lottery.LotteryType, count int) ([]lottery.Draw, error) {
	endpoint := fmt.Sprintf("%s/%s/", c.baseURL, string(ltype))

	var draws []lottery.Draw

	// Delay inicial para não parecer bot
	time.Sleep(1 * time.Second)

	// Buscar o último sorteio primeiro para descobrir o número atual
	latestResp, err := c.client.R().Get(endpoint)

	if err != nil {
		logs.LogError(logs.CategoryData, "Erro de conectividade: %v", err)
		return nil, fmt.Errorf("erro de conectividade: %w", err)
	}

	if latestResp.StatusCode() == 403 {
		logs.LogError(logs.CategoryData, "API da CAIXA bloqueada (erro 403)")
		return nil, fmt.Errorf("API da CAIXA bloqueada (erro 403)")
	}

	if latestResp.StatusCode() != 200 {
		logs.LogError(logs.CategoryData, "API retornou status %d", latestResp.StatusCode())
		return nil, fmt.Errorf("API retornou status %d", latestResp.StatusCode())
	}

	// Debug: Log da resposta
	if config.IsVerbose() {
		logs.LogData("🔍 Debug %s: Status=%d, Content-Type=%s", ltype, latestResp.StatusCode(), latestResp.Header().Get("Content-Type"))
		logs.LogData("🔍 Debug %s: Primeiros 200 chars da resposta: %s", ltype, string(latestResp.Body()[:min(200, len(latestResp.Body()))]))
	}

	var latest lottery.Draw
	if err := json.Unmarshal(latestResp.Body(), &latest); err != nil {
		logs.LogError(logs.CategoryData, "Erro ao decodificar resposta da API: %v", err)
		return nil, fmt.Errorf("erro ao decodificar resposta da API: %w", err)
	}

	draws = append(draws, latest)

	// Buscar sorteios anteriores com delay entre requisições
	for i := 1; i < count && latest.Number-i > 0; i++ {
		// Delay para não sobrecarregar a API
		time.Sleep(500 * time.Millisecond)

		drawNumber := latest.Number - i
		drawResp, err := c.client.R().Get(fmt.Sprintf("%s%d", endpoint, drawNumber))

		if err != nil || drawResp.StatusCode() != 200 {
			if config.IsVerbose() {
				logs.LogData("Erro ao buscar sorteio %d, parando busca: %v", drawNumber, err)
			}
			break
		}

		var draw lottery.Draw
		if err := json.Unmarshal(drawResp.Body(), &draw); err != nil {
			if config.IsVerbose() {
				logs.LogData("Erro ao decodificar sorteio %d: %v", drawNumber, err)
			}
			continue
		}

		draws = append(draws, draw)
	}

	logs.LogData("✅ Fetched %d draws for %s from API", len(draws), ltype)
	return draws, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetDrawByNumber busca um sorteio específico pelo número
func (c *Client) GetDrawByNumber(ltype lottery.LotteryType, number int) (*lottery.Draw, error) {
	endpoint := fmt.Sprintf("%s/%s/%d", c.baseURL, string(ltype), number)

	resp, err := c.client.R().
		SetHeader("Accept", "application/json").
		Get(endpoint)

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar sorteio %d: %w", number, err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("API retornou status %d para sorteio %d", resp.StatusCode(), number)
	}

	var draw lottery.Draw
	if err := json.Unmarshal(resp.Body(), &draw); err != nil {
		return nil, fmt.Errorf("erro ao decodificar sorteio %d: %w", number, err)
	}

	return &draw, nil
}

// GetDrawsRange busca sorteios em um intervalo
func (c *Client) GetDrawsRange(ltype lottery.LotteryType, startNumber, endNumber int) ([]lottery.Draw, error) {
	var draws []lottery.Draw

	for number := startNumber; number <= endNumber; number++ {
		draw, err := c.GetDrawByNumber(ltype, number)
		if err != nil {
			if config.IsVerbose() {
				logs.LogData("Erro ao buscar sorteio %d: %v", number, err)
			}
			continue
		}
		draws = append(draws, *draw)

		// Pequeno delay para não sobrecarregar a API
		time.Sleep(100 * time.Millisecond)
	}

	logs.LogData("✅ Fetched %d draws in range %d-%d for %s", len(draws), startNumber, endNumber, ltype)
	return draws, nil
}

// GetAllHistoricalDraws busca todo o histórico disponível (usar com cuidado)
func (c *Client) GetAllHistoricalDraws(ltype lottery.LotteryType) ([]lottery.Draw, error) {
	// Primeiro, buscar o último sorteio para saber quantos existem
	latest, err := c.GetLatestDraws(ltype, 1)
	if err != nil {
		logs.LogError(logs.CategoryData, "Erro ao buscar último sorteio: %v", err)
		return nil, fmt.Errorf("erro ao buscar último sorteio: %w", err)
	}

	if len(latest) == 0 {
		logs.LogError(logs.CategoryData, "Nenhum sorteio encontrado")
		return nil, fmt.Errorf("nenhum sorteio encontrado")
	}

	latestNumber := latest[0].Number
	logs.LogData("🔍 Buscando histórico completo de %s até sorteio %d", ltype, latestNumber)

	// Buscar todos os sorteios do 1 até o último
	return c.GetDrawsRange(ltype, 1, latestNumber)
}

// TestConnection testa se a API está respondendo
func (c *Client) TestConnection() error {
	resp, err := c.client.R().Get(fmt.Sprintf("%s/megasena/", c.baseURL))

	if err != nil {
		logs.LogError(logs.CategoryData, "Erro de conectividade no teste: %v", err)
		return fmt.Errorf("erro de conectividade: %w", err)
	}

	if resp.StatusCode() == 403 {
		logs.LogError(logs.CategoryData, "API da CAIXA bloqueada no teste (403)")
		return fmt.Errorf("API da CAIXA bloqueada (403)")
	}

	if resp.StatusCode() != 200 {
		logs.LogError(logs.CategoryData, "API retornou status %d no teste", resp.StatusCode())
		return fmt.Errorf("API retornou status %d", resp.StatusCode())
	}

	logs.LogData("✅ Teste de conexão com API da CAIXA bem-sucedido")
	return nil
}

// GetNextDrawInfo busca informações sobre o próximo sorteio
func (c *Client) GetNextDrawInfo(ltype lottery.LotteryType) (time.Time, int, error) {
	draws, err := c.GetLatestDraws(ltype, 1)
	if err != nil {
		logs.LogError(logs.CategoryData, "Erro ao buscar próximo sorteio para %s: %v", ltype, err)
		return time.Time{}, 0, err
	}

	if len(draws) == 0 {
		logs.LogError(logs.CategoryData, "Nenhum sorteio encontrado para %s", ltype)
		return time.Time{}, 0, fmt.Errorf("nenhum sorteio encontrado")
	}

	latest := draws[0]
	logs.LogData("🔍 Próximo sorteio %s: %d em %s", ltype, latest.NextDrawNumber, time.Time(latest.NextDrawDate).Format("02/01/2006"))
	return time.Time(latest.NextDrawDate), latest.NextDrawNumber, nil
}

// TestDirectAPI testa diretamente a API para debug
func (c *Client) TestDirectAPI() (string, error) {
	resp, err := c.client.R().Get(fmt.Sprintf("%s/megasena/", c.baseURL))

	if err != nil {
		logs.LogError(logs.CategoryData, "Erro no teste direto da API: %v", err)
		return "", fmt.Errorf("erro de conectividade: %w", err)
	}

	result := fmt.Sprintf("Status: %d, Content-Type: %s, Body: %s",
		resp.StatusCode(),
		resp.Header().Get("Content-Type"),
		string(resp.Body()))

	logs.LogData("🔍 Teste direto da API: %s", result)
	return result, nil
}

// CleanCache limpa caches antigos
func (c *Client) CleanCache() error {
	err := c.cacheManager.CleanOldCache()
	if err != nil {
		logs.LogError(logs.CategoryData, "Erro ao limpar cache: %v", err)
	} else {
		logs.LogData("✅ Cache limpo com sucesso")
	}
	return err
}
