package models

import (
	"time"
)

// ConcursoData representa dados históricos de um concurso para análise de padrões
type ConcursoData struct {
	Numero         int           `json:"numero"`
	Data           time.Time     `json:"data"`
	Premios        []PremioFaixa `json:"premios"`
	Acumulado      bool          `json:"acumulado"`
	ValorAcumulado float64       `json:"valor_acumulado"`
	IntervaloDias  int           `json:"intervalo_dias"`
	LotteryType    string        `json:"lottery_type"`
	TemporalWeight float64       `json:"temporal_weight"` // Peso temporal para análise ponderada
}

// PremioFaixa representa informações de prêmio por faixa
type PremioFaixa struct {
	Faixa      string  `json:"faixa"`
	Ganhadores int     `json:"ganhadores"`
	Valor      float64 `json:"valor"`
}

// TemperatureAnalysis representa a análise de temperatura de um concurso
type TemperatureAnalysis struct {
	LotteryType        string    `json:"lotteryType"`
	LotteryName        string    `json:"lotteryName"`
	TemperatureScore   int       `json:"temperatureScore"`  // 0-100
	TemperatureLevel   string    `json:"temperatureLevel"`  // "FRIO", "MORNO", "QUENTE", "MUITO_QUENTE", "EXPLOSIVO"
	TemperatureAdvice  string    `json:"temperatureAdvice"` // Conselho para o usuário
	CycleAnalysis      CycleInfo `json:"cycleAnalysis"`
	AccumulationInfo   AccumInfo `json:"accumulationInfo"`
	FrequencyAnalysis  FreqInfo  `json:"frequencyAnalysis"`
	LastUpdate         time.Time `json:"lastUpdate"`
	NextDrawPrediction DrawPred  `json:"nextDrawPrediction"`
}

// CycleInfo análise de ciclos temporais
type CycleInfo struct {
	DaysSinceLastBigPrize   int     `json:"daysSinceLastBigPrize"`
	AverageCycleDays        float64 `json:"averageCycleDays"`
	CycleProgressPercentage float64 `json:"cycleProgressPercentage"`
	IsInHotZone             bool    `json:"isInHotZone"`
}

// AccumInfo análise de acumulação
type AccumInfo struct {
	ConsecutiveAccumulations int     `json:"consecutiveAccumulations"`
	CurrentAccumulatedValue  float64 `json:"currentAccumulatedValue"`
	AverageBeforeExplosion   int     `json:"averageBeforeExplosion"`
	ExplosionProbability     float64 `json:"explosionProbability"`
}

// FreqInfo análise de frequência
type FreqInfo struct {
	DaysSinceLastPrize   int     `json:"daysSinceLastPrize"`
	AverageFrequencyDays float64 `json:"averageFrequencyDays"`
	FrequencyScore       float64 `json:"frequencyScore"`
	IsOverdue            bool    `json:"isOverdue"`
}

// DrawPred predição do próximo sorteio
type DrawPred struct {
	ExpectedBigPrizeProbability float64   `json:"expectedBigPrizeProbability"`
	RecommendedAction           string    `json:"recommendedAction"`
	OptimalPlayWindow           time.Time `json:"optimalPlayWindow"`
	ConfidenceLevel             float64   `json:"confidenceLevel"`
}

// PredictorSummary resumo geral de todas as loterias
type PredictorSummary struct {
	HottestLottery    string                `json:"hottestLottery"`
	ColdestLottery    string                `json:"coldestLottery"`
	Analyses          []TemperatureAnalysis `json:"analyses"`
	GeneralAdvice     string                `json:"generalAdvice"`
	LastUpdate        time.Time             `json:"lastUpdate"`
	OverallConfidence float64               `json:"overallConfidence"`
}

// PredictorMetrics métricas de performance do preditor
type PredictorMetrics struct {
	TotalPredictions    int     `json:"totalPredictions"`
	CorrectPredictions  int     `json:"correctPredictions"`
	AccuracyPercentage  float64 `json:"accuracyPercentage"`
	LastWeekAccuracy    float64 `json:"lastWeekAccuracy"`
	LastMonthAccuracy   float64 `json:"lastMonthAccuracy"`
	UserEngagementBoost float64 `json:"userEngagementBoost"`
	UserROIImprovement  float64 `json:"userROIImprovement"`
}
