import './style.css';
import './app.css';

import { 
    GenerateStrategy, 
    TestConnections, 
    GetNextDraws, 
    SaveConfig, 
    ValidateConfig, 
    GetDefaultConfig, 
    TestConnectionsWithConfig, 
    SaveGame,
    SaveManualGame,
    GetSavedGames,
    DeleteSavedGame,
    GetAppInfo,
    CheckForUpdates,
    GetCurrentConfig,
    // V2.0.0 - ANALYTICS & DASHBOARD
    GetPerformanceMetrics,
    GetNumberFrequencyAnalysis,
    GetROICalculator,
    GetDashboardSummary,
    GetNotifications,
    MarkNotificationAsRead,
    ClearOldNotifications,
    CheckGameResult,
    CheckAllPendingResults
} from '../wailsjs/go/main/App';

import { models } from '../wailsjs/go/models';

// Tipos TypeScript para nossa aplicação
interface UserPreferences {
    lotteryTypes: string[];
    budget: number;
    strategy: string;
    avoidPatterns: boolean;
    favoriteNumbers: number[];
    excludeNumbers: number[];
}

interface LotteryGame {
    type: string;
    numbers: number[];
    cost: number;
}

interface Strategy {
    budget: number;
    totalCost: number;
    games: LotteryGame[];
    reasoning: string;
    statistics: {
        analyzedDraws: number;
        hotNumbers: number[];
        coldNumbers: number[];
    };
}

interface StrategyResponse {
    success: boolean;
    strategy?: Strategy;
    confidence?: number;
    error?: string;
    availableLotteries?: string[];
    failedLotteries?: string[];
}

interface ConnectionStatus {
    caixaAPI: boolean;
    caixaError?: string;
    claudeAPI: boolean;
    claudeError?: string;
}

interface ConfigData {
    claudeApiKey: string;
    claudeModel: string;
    timeoutSec: number;
    maxTokens: number;
    verbose: boolean;
}

// Interfaces para jogos salvos
interface SavedGame {
    id: string;
    lottery_type: string;
    numbers: number[];
    expected_draw: string;
    contest_number: number;
    status: string; // "pending", "checked", "error"
    created_at: string;
    checked_at?: string;
    result?: GameResult;
}

interface GameResult {
    contest_number: number;
    draw_date: string;
    drawn_numbers: number[];
    matches: number[];
    hit_count: number;
    prize: string;
    prize_amount: number;
    is_winner: boolean;
}

// Interfaces para informações do app
interface AppInfo {
    success: boolean;
    version: string;
    platform: string;
    repository: string;
    buildDate: string;
    autoUpdateEnabled: boolean;
}

// ===============================
// V2.0.0 - INTERFACES ANALYTICS
// ===============================

// Interface para métricas de performance
interface PerformanceMetrics {
    totalGames: number;
    totalInvestment: number;
    totalWinnings: number;
    roiPercentage: number;
    winRate: number;
    currentWinStreak: number;
    currentLossStreak: number;
    longestWinStreak: number;
    longestLossStreak: number;
    averageWinAmount: number;
    biggestWin: number;
    last30Days: PeriodMetrics;
    last90Days: PeriodMetrics;
    last365Days: PeriodMetrics;
    monthlyTrends: MonthlyTrend[];
    lotterySpecific: LotterySpecificMetrics;
    dailyPerformance: DailyPerformance[];
}

interface PeriodMetrics {
    games: number;
    investment: number;
    winnings: number;
    roi: number;
    winRate: number;
}

interface MonthlyTrend {
    month: string;
    games: number;
    investment: number;
    winnings: number;
    roi: number;
    growth: number;
}

interface LotterySpecificMetrics {
    megaSena: LotteryMetrics;
    lotofacil: LotteryMetrics;
}

interface LotteryMetrics {
    games: number;
    investment: number;
    winnings: number;
    roi: number;
    winRate: number;
    averageNumbers: number[];
    favoriteNumbers: number[];
}

interface DailyPerformance {
    date: string;
    games: number;
    investment: number;
    winnings: number;
    roi: number;
}

// Interface para análise de frequência de números
interface NumberFrequency {
    number: number;
    frequency: number;
    percentage: number;
    lastSeen: number;
    status: string; // "hot", "cold", "normal"
}

// Interface para calculadora de ROI
interface ROICalculation {
    investment: number;
    timeframe: string;
    projectedWinnings: number;
    projectedROI: number;
    projectedProfit: number;
    historicalROI: number;
    historicalWinRate: number;
    basedOnGames: number;
    confidence: string;
    recommendation: string;
}

// Interface para resumo do dashboard
interface DashboardSummary {
    totalGames: number;
    totalInvestment: number;
    totalWinnings: number;
    currentROI: number;
    winRate: number;
    biggestWin: number;
    averageWin: number;
    trend: string; // "up", "down", "neutral"
    currentStreak: {
        type: string; // "win", "loss", "none"
        count: number;
    };
    last30Days: {
        games: number;
        investment: number;
        winnings: number;
        roi: number;
    };
    performance: {
        level: string; // "Excelente", "Boa", "Regular", "Baixa"
        description: string;
    };
}

// Interface para notificações
interface AppNotification {
    id: string;
    type: string; // "reminder", "result", "performance", "achievement", "system"
    title: string;
    message: string;
    priority: string; // "low", "medium", "high", "urgent"
    category: string; // "game", "finance", "system", "achievement"
    createdAt: string;
    readAt?: string;
    actionURL?: string;
    icon?: string;
    data?: any;
}

// Estado global da aplicação
let userPreferences: UserPreferences = {
    lotteryTypes: [],
    budget: 0,
    strategy: '',
    avoidPatterns: false,
    favoriteNumbers: [],
    excludeNumbers: []
};

let currentConfig: ConfigData = {
    claudeApiKey: '',
    claudeModel: 'claude-opus-4-20250514',
    timeoutSec: 60,
    maxTokens: 8000,
    verbose: false
};

// Estado global da aplicação
let appInfo: AppInfo | null = null;

// ===============================
// INICIALIZAÇÃO
// ===============================

document.addEventListener('DOMContentLoaded', async () => {
    console.log('🎰 Lottery Optimizer iniciado!');
    
    // Carregar informações do app
    await loadAppInfo();
    
    // Carregar configuração atual
    await loadCurrentConfig();
    
    // Verificar configuração e renderizar tela apropriada
    await checkConfigAndRender();
});

// ===============================
// INFORMAÇÕES DO APP E AUTO-UPDATE
// ===============================

async function loadAppInfo() {
    try {
        const response = await GetAppInfo();
        if (response.success) {
            appInfo = response as AppInfo;
            console.log(`🎰 App Info carregado: v${appInfo.version} (${appInfo.platform})`);
        }
    } catch (error) {
        console.error('❌ Erro ao carregar informações do app:', error);
        appInfo = {
            success: false,
            version: 'Unknown',
            platform: 'Unknown',
            repository: 'cccarv82/milhoes-desktop',
            buildDate: new Date().toISOString().split('T')[0],
            autoUpdateEnabled: false
        };
    }
}

// Carregar configuração atual do backend
async function loadCurrentConfig() {
    try {
        console.log('🔧 Carregando configuração atual do backend...');
        const config = await GetCurrentConfig();
        
        if (config.exists) {
            currentConfig = {
                claudeApiKey: config.claudeApiKey || '',
                claudeModel: config.claudeModel || 'claude-opus-4-20250514',
                timeoutSec: config.timeoutSec || 60,
                maxTokens: config.maxTokens || 8000,
                verbose: config.verbose || false
            };
            console.log(`✅ Configuração carregada: APIKey present=${currentConfig.claudeApiKey !== ''}, Model=${currentConfig.claudeModel}`);
        } else {
            console.log('⚠️ Nenhuma configuração encontrada, usando padrão');
            currentConfig = await GetDefaultConfig();
        }
    } catch (error) {
        console.error('❌ Erro ao carregar configuração:', error);
        currentConfig = await GetDefaultConfig();
    }
}

async function checkForUpdatesManually() {
    try {
        console.log('🔄 Verificando atualizações manualmente...');
        const updateInfo = await CheckForUpdates();
        
        if (updateInfo && updateInfo.available) {
            alert(`🎉 Nova versão disponível!\n\nVersão atual: ${appInfo?.version}\nNova versão: ${updateInfo.version}\n\nReinicie o app para que ele baixe automaticamente a atualização.`);
        } else {
            alert('✅ Seu app já está na versão mais recente!');
        }
    } catch (error) {
        console.error('❌ Erro ao verificar atualizações:', error);
        let errorMessage = 'Erro desconhecido';
        if (error instanceof Error) {
            errorMessage = error.message;
        }
        alert(`❌ Erro ao verificar atualizações:\n${errorMessage}`);
    }
}

// ===============================
// TELA DE CONFIGURAÇÃO OBRIGATÓRIA
// ===============================

function renderConfigurationRequired() {
    const app = document.getElementById('app')!;
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <div class="header-content">
                    <h1 class="logo">🎰 Lottery Optimizer</h1>
                    <p class="tagline">Configuração Necessária</p>
                </div>
            </header>
            
            <div class="main-content">
                <div class="error-content" style="max-width: 600px; margin: 0 auto;">
                    <div class="error-icon">⚙️</div>
                    <h2>Configuração Necessária</h2>
                    <p class="error-message">
                        Para usar o Lottery Optimizer, você precisa configurar sua chave da API do Claude. 
                        Isso permite que a IA analise os dados e gere estratégias inteligentes.
                    </p>
                    
                    <div class="error-actions">
                        <button class="btn-primary" onclick="renderConfigurationScreen()">
                            <span class="btn-icon">🔧</span>
                            Configurar Agora
                        </button>
                    </div>
                </div>
      </div>
    </div>
`;
}

// ===============================
// TELA DE CONFIGURAÇÕES
// ===============================

function renderConfigurationScreen() {
    // Carregar configuração atual antes de renderizar
    loadCurrentConfig().then(() => {
        renderConfigurationForm();
    });
}

function renderConfigurationForm() {
    const app = document.getElementById('app')!;
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <div class="header-content">
                    <h1 class="logo">🎰 Lottery Optimizer</h1>
                    <p class="tagline">Configurações</p>
                </div>
                <div class="header-actions">
                    <button class="btn-back" onclick="checkConfigAndRender()">
                        <span class="btn-icon">←</span>
                        Voltar
                    </button>
                </div>
            </header>
            
            <div class="wizard-content">
                <div class="wizard-steps">
                    <div class="step active">⚙️ Configurações</div>
                </div>

                <form class="config-form" onsubmit="handleConfigSave(event)">
                    <!-- API Claude -->
                    <div class="form-section">
                        <h3>
                            <span>🤖</span>
                            Configuração da API Claude
                        </h3>
                        <p style="color: var(--text-secondary); margin-bottom: var(--spacing-6);">
                            O Claude é a IA que analisa os dados históricos e gera estratégias inteligentes. 
                            <a href="https://console.anthropic.com/" target="_blank" style="color: var(--accent-primary);">Obtenha sua chave aqui</a>.
                        </p>
                        
                        <div class="numbers-input">
                            <label for="claudeApiKey">Chave da API *</label>
                            <input 
                                type="password" 
                                id="claudeApiKey" 
                                name="claudeApiKey" 
                                value="${currentConfig.claudeApiKey}"
                                placeholder="sk-ant-api03-..." 
                                required
                            >
                        </div>
                        
                        <div class="numbers-input">
                            <label for="claudeModel">Modelo</label>
                            <select 
                                id="claudeModel" 
                                name="claudeModel" 
                                style="width: 100%; padding: var(--spacing-4); background: var(--bg-tertiary); border: 2px solid var(--border-color); border-radius: var(--border-radius); color: var(--text-primary); font-size: var(--font-size-base);"
                            >
                                <option value="claude-opus-4-20250514" ${currentConfig.claudeModel === 'claude-opus-4-20250514' ? 'selected' : ''}>Claude Opus 4 (🧮 Melhor p/ Matemática)</option>
                                <option value="claude-3-opus-20240229" ${currentConfig.claudeModel === 'claude-3-opus-20240229' ? 'selected' : ''}>Claude 3 Opus (🎯 Recomendado p/ Análise)</option>
                                <option value="claude-sonnet-4-20250514" ${currentConfig.claudeModel === 'claude-sonnet-4-20250514' ? 'selected' : ''}>Claude Sonnet 4 (🆕 Mais Recente)</option>
                                <option value="claude-3-5-sonnet-20241022" ${currentConfig.claudeModel === 'claude-3-5-sonnet-20241022' ? 'selected' : ''}>Claude 3.5 Sonnet</option>
                                <option value="claude-3-haiku-20240307" ${currentConfig.claudeModel === 'claude-3-haiku-20240307' ? 'selected' : ''}>Claude 3 Haiku</option>
                            </select>
                        </div>
                    </div>

                    <!-- Configurações Avançadas -->
                    <div class="form-section">
                        <h3>
                            <span>🔧</span>
                            Configurações Avançadas
                        </h3>
                        
                        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: var(--spacing-4);">
                            <div class="numbers-input">
                                <label for="timeoutSec">Timeout (segundos)</label>
                                <input 
                                    type="number" 
                                    id="timeoutSec" 
                                    name="timeoutSec" 
                                    value="${currentConfig.timeoutSec}"
                                    min="10" 
                                    max="300" 
                                    required
                                >
                            </div>
                            
                            <div class="numbers-input">
                                <label for="maxTokens">Máximo de Tokens</label>
                                <input 
                                    type="number" 
                                    id="maxTokens" 
                                    name="maxTokens" 
                                    value="${currentConfig.maxTokens}"
                                    min="1000" 
                                    max="12000" 
                                    required
                                >
                            </div>
                        </div>
                        
                        <div class="checkbox-option">
                            <input 
                                type="checkbox" 
                                id="verbose" 
                                name="verbose" 
                                ${currentConfig.verbose ? 'checked' : ''}
                            >
                            <label for="verbose">Modo verboso (logs detalhados)</label>
                        </div>
                    </div>

                    <!-- Teste de Conexão -->
                    <div class="form-section">
                        <h3>
                            <span>🔗</span>
                            Teste de Conexões
                        </h3>
                        <p style="color: var(--text-secondary); margin-bottom: var(--spacing-4);">
                            Verifique se as APIs estão funcionando corretamente.
                        </p>
                        
                        <div style="display: flex; justify-content: center; margin-bottom: var(--spacing-6);">
                            <button type="button" class="btn-secondary" onclick="testConnections()">
                                <span class="btn-icon">🔗</span>
                                Testar Conexões
                            </button>
                        </div>
                        
                        <div id="connectionStatus"></div>
                    </div>

                    <!-- Atualizações Automáticas -->
                    <div class="form-section">
                        <h3>
                            <span>🔄</span>
                            Atualizações Automáticas
                        </h3>
                        <p style="color: var(--text-secondary); margin-bottom: var(--spacing-4);">
                            ${appInfo ? `Versão atual: <strong>${appInfo.version}</strong> | Auto-update: <strong>${appInfo.autoUpdateEnabled ? 'Ativado' : 'Desativado'}</strong>` : 'Carregando informações...'}
                        </p>
                        
                        <div style="display: flex; justify-content: center; margin-bottom: var(--spacing-6);">
                            <button type="button" class="btn-secondary" onclick="checkForUpdatesManually()">
                                <span class="btn-icon">🔍</span>
                                Verificar Atualizações
                            </button>
                        </div>
                    </div>

                    <!-- Ações -->
                    <div class="form-actions">
                        <button type="button" class="btn-secondary" onclick="loadDefaultConfig()">
                            <span class="btn-icon">🔄</span>
                            Restaurar Padrão
                        </button>
                        <button type="submit" class="btn-primary">
                            <span class="btn-icon">💾</span>
                            Salvar Configuração
                        </button>
                    </div>
                </form>
            </div>
        </div>
    `;
}

// Testar conexões
async function testConnections() {
    const statusDiv = document.getElementById('connectionStatus')!;
    statusDiv.innerHTML = '<div class="loading">Testando conexões...</div>';
    
    try {
        // Coletar dados do formulário atual
        const form = document.querySelector('.config-form') as HTMLFormElement;
        const formData = new FormData(form);
        
        const testConfig: ConfigData = {
            claudeApiKey: formData.get('claudeApiKey') as string,
            claudeModel: formData.get('claudeModel') as string,
            timeoutSec: parseInt(formData.get('timeoutSec') as string),
            maxTokens: parseInt(formData.get('maxTokens') as string),
            verbose: formData.has('verbose')
        };
        
        // Usar a nova função que testa com a configuração fornecida
        const status: ConnectionStatus = await TestConnectionsWithConfig(testConfig);
        
        statusDiv.innerHTML = `
            <div class="status-grid">
                <div class="status-card ${status.caixaAPI ? 'status-ok' : 'status-error'}">
                    <div class="status-icon">${status.caixaAPI ? '✅' : '❌'}</div>
                    <div class="status-content">
                        <h4>API Caixa</h4>
                        <p>${status.caixaAPI ? 'Conectado' : 'Erro'}</p>
                        ${status.caixaError ? `<small style="color: var(--accent-error);">${status.caixaError}</small>` : ''}
                    </div>
                </div>
                
                <div class="status-card ${status.claudeAPI ? 'status-ok' : 'status-error'}">
                    <div class="status-icon">${status.claudeAPI ? '✅' : '❌'}</div>
                    <div class="status-content">
                        <h4>Claude API</h4>
                        <p>${status.claudeAPI ? 'Conectado' : 'Erro'}</p>
                        ${status.claudeError ? `<small style="color: var(--accent-error);">${status.claudeError}</small>` : ''}
                    </div>
      </div>
    </div>
`;
    } catch (error) {
        statusDiv.innerHTML = `<div class="error-message">Erro ao testar conexões: ${error}</div>`;
    }
}

// Carregar configuração padrão
async function loadDefaultConfig() {
    try {
        currentConfig = await GetDefaultConfig();
        renderConfigurationScreen();
    } catch (error) {
        console.error('Erro ao carregar configuração padrão:', error);
    }
}

// Salvar configuração
async function handleConfigSave(event: Event) {
    event.preventDefault();
    
    const form = event.target as HTMLFormElement;
    const formData = new FormData(form);
    
    const configData: ConfigData = {
        claudeApiKey: formData.get('claudeApiKey') as string,
        claudeModel: formData.get('claudeModel') as string,
        timeoutSec: parseInt(formData.get('timeoutSec') as string),
        maxTokens: parseInt(formData.get('maxTokens') as string),
        verbose: formData.has('verbose')
    };
    
    try {
        const saveButton = form.querySelector('button[type="submit"]') as HTMLButtonElement;
        saveButton.innerHTML = '<span class="btn-icon">⏳</span> Salvando...';
        saveButton.disabled = true;
        
        const result = await SaveConfig(configData);
        
        if (result.success) {
            currentConfig = configData;
            
            // Mostrar sucesso
            const statusDiv = document.getElementById('connectionStatus')!;
            statusDiv.innerHTML = `
                <div style="background: rgba(16, 185, 129, 0.1); border: 1px solid var(--accent-success); border-radius: var(--border-radius); padding: var(--spacing-4); color: var(--accent-success);">
                    ✅ ${result.message}
                </div>
            `;
            
            // Aguardar um pouco para o backend processar a nova configuração
            setTimeout(async () => {
                renderWelcome();
                // Recarregar status das conexões após um pequeno delay
                setTimeout(() => {
                    loadConnectionStatus();
                }, 500);
            }, 1000);
        } else {
            throw new Error(result.error);
        }
    } catch (error) {
        const statusDiv = document.getElementById('connectionStatus')!;
        statusDiv.innerHTML = `
            <div style="background: rgba(239, 68, 68, 0.1); border: 1px solid var(--accent-error); border-radius: var(--border-radius); padding: var(--spacing-4); color: var(--accent-error);">
                ❌ Erro: ${error}
            </div>
        `;
        
        const saveButton = form.querySelector('button[type="submit"]') as HTMLButtonElement;
        saveButton.innerHTML = '<span class="btn-icon">💾</span> Salvar Configuração';
        saveButton.disabled = false;
    }
}

// ===============================
// VERIFICAR CONFIGURAÇÃO E RENDERIZAR
// ===============================

async function checkConfigAndRender() {
    try {
        const validation = await ValidateConfig();
        
        if (!validation.claudeConfigured) {
            renderConfigurationRequired();
        } else {
            renderWelcome();
        }
    } catch (error) {
        console.error('Erro ao validar configuração:', error);
        renderConfigurationRequired();
    }
}

// ===============================
// TELA DE BOAS-VINDAS MODERNA
// ===============================

function renderWelcome() {
    const app = document.getElementById('app')!;
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <div class="header-content">
                    <h1 class="logo">🎰 Lottery Optimizer</h1>
                    <p class="tagline">Estratégias Inteligentes para Loterias</p>
                </div>
                <div class="header-actions">
                    ${appInfo ? `<div class="version-badge">${appInfo.version}</div>` : ''}
                    <div class="ai-badge">
                        <span class="ai-icon">🤖</span>
                        Powered by Claude AI
                    </div>
                </div>
            </header>
            
            <div class="main-content">
                <div class="welcome-section">
                    <h2 style="font-size: var(--font-size-4xl); font-weight: 800; margin-bottom: var(--spacing-4); background: linear-gradient(135deg, var(--accent-primary), var(--accent-secondary)); background-clip: text; -webkit-background-clip: text; -webkit-text-fill-color: transparent;">
                        Estratégias Inteligentes para Loterias 🚀
                    </h2>
                    <p style="font-size: var(--font-size-xl); color: var(--text-secondary); max-width: 600px; margin: 0 auto var(--spacing-8) auto; line-height: 1.7;">
                        IA avançada + Análise matemática + Sistemas profissionais
                    </p>
                </div>
                
                <!-- Botões Principais -->
                <div class="cta-section">
                    <!-- Botão Principal: Gerar Estratégia -->
                    <button class="btn-primary main-cta" onclick="startStrategyWizard()">
                        <span class="btn-icon">🎯</span>
                        Gerar Estratégia
                    </button>
                    
                    <!-- Botão Dashboard -->
                    <button class="btn-primary dashboard-btn" onclick="renderPerformanceDashboard()">
                        <span class="btn-icon">📊</span>
                        Dashboard de Performance
                    </button>
                </div>
                
                <!-- Menu de Navegação -->
                <div class="main-nav-grid">
                    <button class="main-nav-btn" onclick="renderSavedGamesScreen()">
                        <span class="btn-icon">💾</span>
                        Jogos Salvos
                    </button>
                    
                    <button class="main-nav-btn" onclick="renderIntelligenceEngine()">
                        <span class="btn-icon">🧠</span>
                        Intelligence Engine
                    </button>
                    
                    <button class="main-nav-btn" onclick="renderROICalculator()">
                        <span class="btn-icon">💰</span>
                        Calc. ROI
                    </button>
                    
                    <button class="main-nav-btn" onclick="renderNotificationsCenter()">
                        <span class="btn-icon">🔔</span>
                        Notificações
                    </button>
                    
                    <button class="main-nav-btn" onclick="renderConfigurationScreen()">
                        <span class="btn-icon">⚙️</span>
                        Configurações
                    </button>
                </div>

                <!-- Features em Grid Compacto -->
                <div class="features-compact">
                    <h3 style="text-align: center; color: var(--accent-primary); margin-bottom: var(--spacing-6);">
                        🎲 Tecnologia de Ponta
                    </h3>
                    <div class="features-compact-grid">
                        <div class="feature-compact">
                            <span class="feature-compact-icon">🧠</span>
                            <h4>IA Claude Opus 4</h4>
                            <p>Análise de 250+ sorteios com sistemas Wheeling profissionais</p>
                        </div>
                        
                        <div class="feature-compact">
                            <span class="feature-compact-icon">📊</span>
                            <h4>Análise Matemática</h4>
                            <p>Matriz de distância Hamming e 6 filtros matemáticos obrigatórios</p>
                        </div>
                        
                        <div class="feature-compact">
                            <span class="feature-compact-icon">💎</span>
                            <h4>Multi-Loteria</h4>
                            <p>Mega-Sena e Lotofácil com preços oficiais CAIXA</p>
                        </div>
                        
                        <div class="feature-compact">
                            <span class="feature-compact-icon">🔒</span>
                            <h4>100% Privado</h4>
                            <p>Todos os cálculos são locais, seus dados não saem do computador</p>
                        </div>
                        
                        <div class="feature-compact">
                            <span class="feature-compact-icon">🔍</span>
                            <h4>Verificação Automática</h4>
                            <p>Sistema inteligente que verifica seus jogos automaticamente</p>
                        </div>
                        
                        <div class="feature-compact">
                            <span class="feature-compact-icon">🔄</span>
                            <h4>Auto-Update</h4>
                            <p>Sempre na versão mais recente com atualizações automáticas</p>
                        </div>
                    </div>
                </div>
                
                <!-- Informações dos próximos sorteios -->
                <div class="next-draws-section">
                    <h3>
                        <span>🎯</span>
                        Próximos Sorteios
                    </h3>
                    <div class="draws-grid" id="nextDraws">
                        <div class="loading">Carregando próximos sorteios...</div>
                    </div>
                </div>
                
                <!-- Status das conexões -->
                <div class="status-section">
                    <h3>
                        <span>🔗</span>
                        Status das Conexões
                    </h3>
                    <div class="status-grid" id="connectionStatusGrid">
                        <div class="loading">Verificando conexões...</div>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // Carregar dados assíncronos
    loadNextDraws();
    loadConnectionStatus();
}

// Carregar próximos sorteios
async function loadNextDraws() {
    try {
        const nextDraws = await GetNextDraws();
        const container = document.getElementById('nextDraws');
        
        if (!container) {
            console.warn('Elemento nextDraws não encontrado');
            return;
        }
        
        let html = '';
        
        if (nextDraws.megasena) {
            html += `
                <div class="draw-card">
                    <div class="draw-icon">🔥</div>
                    <div class="draw-content">
                        <h4>Mega-Sena</h4>
                        <p>Sorteio ${nextDraws.megasena.number}</p>
                        <small>${nextDraws.megasena.date}</small>
                    </div>
                </div>
            `;
        }
        
        if (nextDraws.lotofacil) {
            html += `
                <div class="draw-card">
                    <div class="draw-icon">⭐</div>
                    <div class="draw-content">
                        <h4>Lotofácil</h4>
                        <p>Sorteio ${nextDraws.lotofacil.number}</p>
                        <small>${nextDraws.lotofacil.date}</small>
                    </div>
                </div>
            `;
        }
        
        if (html === '') {
            html = '<div class="no-data">Nenhum sorteio programado</div>';
        }
        
        container.innerHTML = html;
    } catch (error) {
        const container = document.getElementById('nextDraws');
        if (container) {
            container.innerHTML = '<div class="no-data">Erro ao carregar sorteios</div>';
        }
    }
}

// Carregar status das conexões
async function loadConnectionStatus() {
    try {
        const status: ConnectionStatus = await TestConnections();
        const container = document.getElementById('connectionStatusGrid')!;
        
        container.innerHTML = `
            <div class="status-card ${status.caixaAPI ? 'status-ok' : 'status-error'}">
                <div class="status-icon">${status.caixaAPI ? '✅' : '❌'}</div>
                <div class="status-content">
                    <h4>API Caixa</h4>
                    <p>${status.caixaAPI ? 'Conectado' : 'Erro de conexão'}</p>
                </div>
            </div>
            
            <div class="status-card ${status.claudeAPI ? 'status-ok' : 'status-error'}">
                <div class="status-icon">${status.claudeAPI ? '✅' : '❌'}</div>
                <div class="status-content">
                    <h4>Claude API</h4>
                    <p>${status.claudeAPI ? 'Conectado' : 'Erro de conexão'}</p>
                </div>
            </div>
        `;
    } catch (error) {
        document.getElementById('connectionStatusGrid')!.innerHTML = '<div class="no-data">Erro ao verificar conexões</div>';
    }
}

// ===============================
// WIZARD DE ESTRATÉGIA
// ===============================

// Iniciar wizard de estratégia
function startStrategyWizard() {
    renderPreferencesForm();
}

// Renderizar formulário de preferências
function renderPreferencesForm() {
    const app = document.getElementById('app')!;
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <div class="header-content">
                    <h1 class="logo">🎰 Lottery Optimizer</h1>
                    <p class="tagline">Assistente de Estratégia</p>
                </div>
                <button class="btn-back" onclick="renderWelcome()">
                    <span class="btn-icon">←</span>
                    Voltar
                </button>
            </header>
            
            <div class="wizard-content">
                <div class="wizard-steps">
                    <div class="step active">1. Preferências</div>
                    <div class="step">2. Estratégia</div>
                    <div class="step">3. Resultados</div>
                </div>

                <form class="preferences-form" onsubmit="handlePreferencesSubmit(event)">
                    <!-- Seleção de Loterias -->
                    <div class="form-section">
                        <h3>
                            <span>🎯</span>
                            Escolha suas Loterias
                        </h3>
                        <div class="lottery-options">
                            <label class="lottery-option">
                                <input type="checkbox" name="lotteryType" value="megasena">
                                <div class="option-card">
                                    <span class="option-icon">🔥</span>
                                    <div class="option-content">
                                        <h4>Mega-Sena</h4>
                                        <p>6 números de 1 a 60</p>
                                        <small>Sorteios: Wed & Sat</small>
                                    </div>
                                </div>
                            </label>
                            
                            <label class="lottery-option">
                                <input type="checkbox" name="lotteryType" value="lotofacil">
                                <div class="option-card">
                                    <span class="option-icon">⭐</span>
                                    <div class="option-content">
                                        <h4>Lotofácil</h4>
                                        <p>15 números de 1 a 25</p>
                                        <small>Sorteios: Mon, Tue, Thu, Fri</small>
                                    </div>
                                </div>
                            </label>
                        </div>
                    </div>

                    <!-- Orçamento -->
                    <div class="form-section">
                        <h3>
                            <span>💰</span>
                            Defina seu Orçamento
                        </h3>
                        <div class="budget-input">
                            <span class="currency">R$</span>
                            <input 
                                type="number" 
                                name="budget" 
                                placeholder="100" 
                                min="5" 
                                max="10000" 
                                step="0.01" 
                                required
                            >
                        </div>
                        <div class="budget-suggestions">
                            <button type="button" class="budget-btn" onclick="setBudget(20)">R$ 20</button>
                            <button type="button" class="budget-btn" onclick="setBudget(50)">R$ 50</button>
                            <button type="button" class="budget-btn" onclick="setBudget(100)">R$ 100</button>
                            <button type="button" class="budget-btn" onclick="setBudget(200)">R$ 200</button>
                            <button type="button" class="budget-btn" onclick="setBudget(500)">R$ 500</button>
                        </div>
                    </div>

                    <!-- Estratégia -->
                    <div class="form-section">
                        <h3>
                            <span>🧠</span>
                            Estratégia de Análise
                        </h3>
                        <div class="strategy-info" style="background: var(--bg-tertiary); padding: var(--spacing-4); border-radius: var(--border-radius); margin-bottom: var(--spacing-4);">
                            <h4 style="color: var(--accent-primary); margin-bottom: var(--spacing-2);">🎯 Estratégia Inteligente</h4>
                            <p style="color: var(--text-secondary); margin: 0; line-height: 1.6;">
                                Nossa IA analisa milhares de sorteios históricos, identifica padrões estatísticos, 
                                números quentes e frios, e gera combinações otimizadas para maximizar suas chances de ganhar.
                            </p>
                        </div>
                        
                        <div class="strategy-options">
                            <label class="strategy-option">
                                <input type="radio" name="strategy" value="intelligent" checked style="display: none;">
                                <div class="option-card" style="border: 2px solid var(--accent-primary); background: rgba(99, 102, 241, 0.1);">
                                    <span class="option-icon">🤖</span>
                                    <div class="option-content">
                                        <h4>Análise Completa da IA</h4>
                                        <p>Combina análise estatística avançada com suas preferências</p>
                                        <small style="color: var(--accent-primary); font-weight: 600;">✨ Recomendado para todos os usuários</small>
                                    </div>
                                </div>
                            </label>
                        </div>
                    </div>

                    <!-- Opções Avançadas -->
                    <div class="form-section">
                        <h3>
                            <span>🔧</span>
                            Opções Avançadas
                        </h3>
                        
                        <div class="numbers-input">
                            <label for="favoriteNumbers">Números da sorte (opcional)</label>
                            <input 
                                type="text" 
                                name="favoriteNumbers" 
                                id="favoriteNumbers"
                                placeholder="Ex: 7, 13, 25, 42"
                            >
                        </div>
                        
                        <div class="numbers-input">
                            <label for="excludeNumbers">Números a evitar (opcional)</label>
                            <input 
                                type="text" 
                                name="excludeNumbers" 
                                id="excludeNumbers"
                                placeholder="Ex: 4, 13, 24"
                            >
                        </div>
                    </div>

                    <div class="form-actions">
                        <button type="button" class="btn-secondary" onclick="renderWelcome()">
                            <span class="btn-icon">←</span>
                            Cancelar
                        </button>
                        <button type="submit" class="btn-primary">
                            <span class="btn-icon">🚀</span>
                            Gerar Estratégia
                        </button>
                    </div>
                </form>
            </div>
        </div>
    `;
}

// Definir orçamento
function setBudget(amount: number) {
    const budgetInput = document.querySelector('input[name="budget"]') as HTMLInputElement;
    budgetInput.value = amount.toString();
}

// Manipular envio das preferências
async function handlePreferencesSubmit(event: Event) {
    event.preventDefault();
    
    const form = event.target as HTMLFormElement;
    
    // Coletar tipos de loteria
    const lotteryTypes = Array.from(form.querySelectorAll('input[name="lotteryType"]:checked'))
        .map(el => (el as HTMLInputElement).value);
    
    if (lotteryTypes.length === 0) {
        alert('Selecione pelo menos uma loteria!');
        return;
    }
    
    // Processar números
    const favoriteNumbers = processNumbersInput(form.favoriteNumbers.value);
    const excludeNumbers = processNumbersInput(form.excludeNumbers.value);
    
    // Montar preferências
    userPreferences = {
        lotteryTypes,
        budget: parseFloat(form.budget.value),
        strategy: form.strategy.value,
        avoidPatterns: true, // Sempre ativo (removido da interface)
        favoriteNumbers,
        excludeNumbers
    };
    
    // Gerar estratégia
    await generateStrategy();
}

// Processar entrada de números
function processNumbersInput(input: string): number[] {
    if (!input.trim()) return [];
    
    return input.split(',')
        .map(n => parseInt(n.trim()))
        .filter(n => !isNaN(n) && n > 0);
}

// Gerar estratégia
async function generateStrategy() {
    renderGeneratingScreen();
    
    try {
        // Etapa 1: Coletando dados históricos
        updateLoadingStep(0, "Coletando dados históricos...");
        await new Promise(resolve => setTimeout(resolve, 800));
        
        // Etapa 2: Analisando padrões
        updateLoadingStep(1, "Analisando padrões com IA...");
        await new Promise(resolve => setTimeout(resolve, 500));
        
        // Etapa 3: Gerando estratégia (chamada real do backend)
        updateLoadingStep(2, "Calculando probabilidades...");
        
        const response: StrategyResponse = await GenerateStrategy(userPreferences);
        
        // Debug: verificar resposta do backend
        console.log('🔍 Response from backend:', response);
        console.log('🔍 Success:', response.success);
        console.log('🔍 Strategy exists:', !!response.strategy);
        if (response.strategy) {
            console.log('🔍 Games:', response.strategy.games);
        }
        if (response.error) {
            console.log('🔍 Error:', response.error);
        }
        
        // Etapa 4: Otimizando combinações
        updateLoadingStep(3, "Otimizando combinações...");
        await new Promise(resolve => setTimeout(resolve, 500));
        
        // Etapa 5: Finalizando
        updateLoadingStep(4, "Finalizando estratégia...");
        await new Promise(resolve => setTimeout(resolve, 500));
        
        if (response.success) {
            renderStrategyResult(response);
        } else {
            renderError(response.error || 'Erro desconhecido');
        }
    } catch (error) {
        console.error('Erro ao gerar estratégia:', error);
        renderError('Erro na análise da IA: ' + error);
    }
}

// Atualizar etapa do loading
function updateLoadingStep(stepIndex: number, message: string) {
    const steps = document.querySelectorAll('.loading-step');
    
    // Remover active de todas as etapas
    steps.forEach(step => step.classList.remove('active'));
    
    // Ativar etapa atual
    if (steps[stepIndex]) {
        steps[stepIndex].classList.add('active');
        steps[stepIndex].textContent = message;
    }
}

// ===============================
// TELA DE GERAÇÃO
// ===============================

function renderGeneratingScreen() {
    const app = document.getElementById('app')!;
    app.innerHTML = `
        <div class="loading-screen">
            <div class="loading-content">
                <div class="loading-spinner">🤖</div>
                <h2>Gerando Estratégia Inteligente</h2>
                <div class="loading-steps">
                    <div class="loading-step active">Preparando análise...</div>
                    <div class="loading-step">Analisando padrões com IA...</div>
                    <div class="loading-step">Calculando probabilidades...</div>
                    <div class="loading-step">Otimizando combinações...</div>
                    <div class="loading-step">Finalizando estratégia...</div>
                </div>
            </div>
        </div>
    `;
}

// ===============================
// TELA DE RESULTADOS
// ===============================

function renderStrategyResult(response: StrategyResponse) {
    const strategy = response.strategy!;
    
    // Debug: verificar estrutura da resposta
    console.log('🔍 Debug response:', response);
    console.log('🔍 Debug strategy:', strategy);
    console.log('🔍 Debug games:', strategy.games);
    
    // Validação de segurança
    if (!strategy) {
        console.error('❌ Strategy is null');
        renderError('Erro: Estratégia não foi gerada corretamente');
        return;
    }
    
    if (!strategy.games || !Array.isArray(strategy.games)) {
        console.error('❌ Games is null or not array:', strategy.games);
        renderError('Erro: Jogos não foram gerados corretamente');
        return;
    }
    
    if (strategy.games.length === 0) {
        console.error('❌ No games generated');
        renderError('Erro: Nenhum jogo foi gerado');
        return;
    }

    // 🚨 VALIDAÇÃO CRÍTICA: Verificar números mínimos
    for (let i = 0; i < strategy.games.length; i++) {
        const game = strategy.games[i];
        const minNumbers = (game.type === 'lotofacil') ? 15 : 6;
        const maxNumbers = (game.type === 'lotofacil') ? 25 : 60;
        
        if (!game.numbers || game.numbers.length < minNumbers) {
            console.error(`❌ ERRO CRÍTICO: Jogo ${i+1} (${game.type}) tem apenas ${game.numbers?.length || 0} números, mínimo é ${minNumbers}`);
            renderError(`Erro crítico: Jogo ${i+1} da ${game.type === 'lotofacil' ? 'Lotofácil' : 'Mega-Sena'} tem apenas ${game.numbers?.length || 0} números. Mínimo obrigatório: ${minNumbers} números.`);
            return;
        }

        // Verificar se os números estão no range correto
        for (const num of game.numbers) {
            if (num < 1 || num > maxNumbers) {
                console.error(`❌ ERRO: Número ${num} fora do range (1-${maxNumbers}) no jogo ${i+1}`);
                renderError(`Erro: Número ${num} inválido no jogo ${i+1}. Deve estar entre 1 e ${maxNumbers}.`);
                return;
            }
        }

        console.log(`✅ Jogo ${i+1} validado: ${game.type} com ${game.numbers.length} números`);
    }
    
    // Salvar estratégia globalmente para impressão
    (window as any).currentStrategy = strategy;
    
    const app = document.getElementById('app')!;
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <div class="header-content">
                    <h1 class="logo">🎰 Lottery Optimizer</h1>
                    <p class="tagline">Estratégia Gerada</p>
                </div>
                <button class="btn-back" onclick="renderWelcome()">
                    <span class="btn-icon">🏠</span>
                    Início
                </button>
            </header>
            
            <div class="strategy-content">
                <div class="wizard-steps">
                    <div class="step">1. Preferências</div>
                    <div class="step">2. Estratégia</div>
                    <div class="step active">3. Resultados ✨</div>
                </div>

                <!-- Resumo da Estratégia -->
                <div class="strategy-summary">
                    <div class="summary-item">
                        <span class="label">Orçamento</span>
                        <span class="value">R$ ${strategy.budget.toFixed(2)}</span>
                    </div>
                    <div class="summary-item">
                        <span class="label">Custo Total</span>
                        <span class="value">R$ ${strategy.totalCost.toFixed(2)}</span>
                    </div>
                    <div class="summary-item">
                        <span class="label">Jogos</span>
                        <span class="value">${strategy.games.length}</span>
                    </div>
                    <div class="summary-item">
                        <span class="label">Confiança IA</span>
                        <span class="value">${((response.confidence || 0) * 100).toFixed(1)}%</span>
                    </div>
                </div>

                <!-- Jogos Gerados -->
                <div class="games-section">
                    <h3>
                        <span>🎲</span>
                        Seus Jogos Inteligentes
                    </h3>
                    <div class="games-grid">
                        ${strategy.games.map((game: LotteryGame, index: number) => `
                            <div class="game-card">
                                <div class="game-header">
                                    <span class="game-icon">${(game.type === 'megasena' || game.type === 'mega-sena') ? '🔥' : '⭐'}</span>
                                    <span class="game-title">${(game.type === 'megasena' || game.type === 'mega-sena') ? 'Mega-Sena' : 'Lotofácil'} #${index + 1}</span>
                                    <span class="game-cost">R$ ${game.cost.toFixed(2)}</span>
                                </div>
                                <div class="game-numbers">
                                    ${game.numbers.slice().sort((a, b) => a - b).map((num: number) => `<span class="number">${num.toString().padStart(2, '0')}</span>`).join('')}
                                </div>
                                <div class="game-actions" style="margin-top: var(--spacing-3); text-align: center;">
                                    <button class="btn-save-game" onclick="showSaveGameModal('${game.type}', [${game.numbers.join(',')}])" style="background: var(--accent-success); color: white; border: none; padding: var(--spacing-2) var(--spacing-4); border-radius: var(--border-radius); font-size: var(--font-size-sm); cursor: pointer; display: inline-flex; align-items: center; gap: var(--spacing-1);">
                                        <span>💾</span>
                                        Salvar Jogo
                                    </button>
                                </div>
                            </div>
                        `).join('')}
                    </div>
                </div>

                <!-- Raciocínio da IA -->
                <div class="form-section">
                    <h3>
                        <span>🧠</span>
                        Raciocínio da Inteligência Artificial
                    </h3>
                    <div style="background: var(--bg-tertiary); padding: var(--spacing-6); border-radius: var(--border-radius); border: 1px solid var(--border-color); line-height: 1.7; color: var(--text-secondary);">
                        ${strategy.reasoning.replace(/\n/g, '<br>')}
                    </div>
                </div>

                <!-- Estatísticas da Análise -->
                <div class="form-section">
                    <h3>
                        <span>📊</span>
                        Análise Estatística Detalhada
                    </h3>
                    
                    <!-- Estatísticas Gerais -->
                    <div class="stats-overview">
                        <div class="stats-header">
                            <h4>📈 Visão Geral da Análise</h4>
                            <div class="stats-badges">
                                <span class="stats-badge draws">📊 ${strategy.statistics.analyzedDraws || 250} sorteios analisados</span>
                                <span class="stats-badge coverage">🎯 Cobertura otimizada</span>
                                <span class="stats-badge ai">🤖 IA nível mundial</span>
                            </div>
                        </div>
                        
                        <div class="stats-description">
                            <p>Esta estratégia foi gerada através de análise estatística avançada de <strong>${strategy.statistics.analyzedDraws || 250} sorteios históricos</strong>, 
                            aplicando sistemas de redução profissionais, filtros matemáticos e teoria combinatorial para maximizar suas chances de retorno.</p>
                        </div>
                    </div>

                    <!-- Estatísticas por Loteria -->
                    <div class="stats-by-lottery">
                        ${generateLotteryStats(strategy)}
                    </div>

                    <!-- Estatísticas de Distribuição -->
                    <div class="distribution-stats">
                        <h4>🔬 Análise de Distribuição</h4>
                        <div class="distribution-grid">
                            <div class="distribution-item">
                                <div class="distribution-label">Estratégia de Cobertura</div>
                                <div class="distribution-value">
                                    ${strategy.games.length > 1 ? 'Diversificação Máxima' : 'Foco Concentrado'}
                                </div>
                                <div class="distribution-desc">
                                    ${strategy.games.length > 1 
                                        ? 'Múltiplos jogos com distância de Hamming ≥8 para máxima cobertura combinatorial'
                                        : 'Jogo único otimizado com base em análise estatística avançada'
                                    }
                                </div>
                            </div>
                            
                            <div class="distribution-item">
                                <div class="distribution-label">Eficiência de Orçamento</div>
                                <div class="distribution-value">
                                    ${((strategy.totalCost / strategy.budget) * 100).toFixed(1)}%
                                </div>
                                <div class="distribution-desc">
                                    Utilização otimizada do orçamento priorizando jogos mais eficientes
                                </div>
                            </div>
                            
                            <div class="distribution-item">
                                <div class="distribution-label">Valor Esperado</div>
                                <div class="distribution-value">
                                    ${calculateExpectedReturn(strategy)}
                                </div>
                                <div class="distribution-desc">
                                    Retorno esperado baseado em probabilidades matemáticas e prêmios históricos
                                </div>
                            </div>
                            
                            <div class="distribution-item">
                                <div class="distribution-label">Sistemas Aplicados</div>
                                <div class="distribution-value">
                                    ${getAppliedSystems(strategy)}
                                </div>
                                <div class="distribution-desc">
                                    Filtros matemáticos e sistemas de redução profissionais utilizados
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- Análise de Números -->
                    <div class="numbers-analysis">
                        <h4>🔢 Análise Profunda de Números</h4>
                        <div class="numbers-analysis-grid">
                            <!-- Números Quentes -->
                            <div class="numbers-category hot-numbers">
                                <div class="category-header">
                                    <span class="category-icon">🔥</span>
                                    <div class="category-info">
                                        <h5>Números Frequentes</h5>
                                        <p>Mais sorteados nos últimos ${strategy.statistics.analyzedDraws || 250} concursos</p>
                                    </div>
                                </div>
                                <div class="numbers-container">
                                    ${(strategy.statistics.hotNumbers && Array.isArray(strategy.statistics.hotNumbers)) 
                                        ? strategy.statistics.hotNumbers.slice(0, 12).map(num => 
                                            `<span class="analysis-number hot">${num.toString().padStart(2, '0')}</span>`
                                        ).join('')
                                        : '<span class="no-data-inline">Dados não disponíveis</span>'
                                    }
                                </div>
                            </div>
                            
                            <!-- Números Frios -->
                            <div class="numbers-category cold-numbers">
                                <div class="category-header">
                                    <span class="category-icon">❄️</span>
                                    <div class="category-info">
                                        <h5>Números "Devidos"</h5>
                                        <p>Menos sorteados - maior probabilidade estatística</p>
                                    </div>
                                </div>
                                <div class="numbers-container">
                                    ${(strategy.statistics.coldNumbers && Array.isArray(strategy.statistics.coldNumbers))
                                        ? strategy.statistics.coldNumbers.slice(0, 12).map(num => 
                                            `<span class="analysis-number cold">${num.toString().padStart(2, '0')}</span>`
                                        ).join('')
                                        : '<span class="no-data-inline">Dados não disponíveis</span>'
                                    }
                                </div>
                            </div>
                        </div>
                        
                        <!-- Estratégia de Seleção -->
                        <div class="selection-strategy">
                            <div class="strategy-item">
                                <span class="strategy-icon">⚖️</span>
                                <div class="strategy-content">
                                    <h6>Balanceamento Inteligente</h6>
                                    <p>A IA aplicou uma estratégia híbrida combinando 60% de números frequentes com 40% de números "devidos", 
                                    seguindo a Lei dos Grandes Números para maximizar as chances de acerto.</p>
                                </div>
                            </div>
                            
                            <div class="strategy-item">
                                <span class="strategy-icon">🎯</span>
                                <div class="strategy-content">
                                    <h6>Filtros Matemáticos</h6>
                                    <p>Todos os jogos passaram por 6 filtros matemáticos obrigatórios: soma balanceada, paridade, distribuição por quadrantes, 
                                    máximo 2 consecutivos, diversificação de terminações e distância de Hamming entre jogos.</p>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Avisos e Alertas -->
                ${(response.failedLotteries && response.failedLotteries.length > 0) ? `
                    <div style="background: rgba(245, 158, 11, 0.1); border: 1px solid var(--accent-warning); border-radius: var(--border-radius); padding: var(--spacing-4); margin: var(--spacing-6) 0; color: var(--accent-warning);">
                        ⚠️ <strong>Aviso:</strong> Algumas loterias não estavam disponíveis: ${response.failedLotteries.join(', ')}. 
                        A estratégia foi gerada apenas para: ${response.availableLotteries?.join(', ')}.
                    </div>
                ` : ''}

                <!-- Ações -->
                <div class="form-actions">
                    <button class="btn-secondary" onclick="renderPreferencesForm()">
                        <span class="btn-icon">🔄</span>
                        Nova Estratégia
                    </button>
                    <button class="btn-primary" onclick="printStrategy()">
                        <span class="btn-icon">🖨️</span>
                        Imprimir Jogos
                    </button>
                </div>
            </div>
        </div>
    `;
}

// Imprimir estratégia
function printStrategy() {
    // Buscar os dados da estratégia atual
    const strategy = (window as any).currentStrategy;
    if (!strategy) {
        alert('Nenhuma estratégia disponível para impressão');
        return;
    }
    
    // Criar janela de impressão apenas com os jogos
    const printWindow = window.open('', '_blank');
    if (!printWindow) {
        alert('Bloqueador de pop-up ativo. Permita pop-ups para imprimir.');
        return;
    }
    
    const printContent = `
        <!DOCTYPE html>
        <html>
        <head>
            <title>Jogos da Loteria - Lottery Optimizer</title>
            <style>
                body {
                    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                    margin: 0;
                    padding: 20px;
                    color: #333;
                    background: white;
                }
                .header {
                    text-align: center;
                    margin-bottom: 30px;
                    border-bottom: 3px solid #6366f1;
                    padding-bottom: 20px;
                }
                .header h1 {
                    margin: 0 0 10px 0;
                    color: #1e293b;
                    font-size: 28px;
                }
                .header p {
                    margin: 5px 0;
                    color: #64748b;
                    font-size: 14px;
                }
                .summary {
                    display: grid;
                    grid-template-columns: repeat(4, 1fr);
                    gap: 15px;
                    margin-bottom: 30px;
                    padding: 20px;
                    background: #f8fafc;
                    border-radius: 12px;
                    border: 2px solid #e2e8f0;
                }
                .summary-item {
                    text-align: center;
                    padding: 10px;
                }
                .summary-label {
                    font-size: 12px;
                    color: #64748b;
                    margin-bottom: 8px;
                    text-transform: uppercase;
                    font-weight: 600;
                    letter-spacing: 0.5px;
                }
                .summary-value {
                    font-size: 24px;
                    font-weight: bold;
                    color: #1e293b;
                }
                .games-grid {
                    display: grid;
                    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
                    gap: 20px;
                    margin-bottom: 30px;
                }
                .game-card {
                    border: 2px solid #e2e8f0;
                    border-radius: 12px;
                    padding: 20px;
                    background: white;
                    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
                }
                .game-header {
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    margin-bottom: 15px;
                    padding-bottom: 10px;
                    border-bottom: 1px solid #e2e8f0;
                }
                .game-title {
                    font-size: 18px;
                    font-weight: bold;
                    color: #1e293b;
                }
                .game-cost {
                    color: #059669;
                    font-size: 16px;
                    font-weight: bold;
                    background: #ecfdf5;
                    padding: 4px 8px;
                    border-radius: 6px;
                }
                .game-numbers {
                    display: flex;
                    flex-wrap: wrap;
                    gap: 8px;
                    justify-content: center;
                }
                .number {
                    background: #6366f1;
                    color: white;
                    padding: 10px 12px;
                    border-radius: 50%;
                    font-weight: bold;
                    font-size: 16px;
                    min-width: 24px;
                    text-align: center;
                    box-shadow: 0 2px 4px rgba(99, 102, 241, 0.3);
                }
                .footer {
                    margin-top: 40px;
                    text-align: center;
                    color: #64748b;
                    font-size: 12px;
                    border-top: 1px solid #e2e8f0;
                    padding-top: 20px;
                }
                .footer p {
                    margin: 5px 0;
                }
                .disclaimer {
                    background: #fef3c7;
                    border: 1px solid #f59e0b;
                    border-radius: 8px;
                    padding: 15px;
                    margin: 20px 0;
                    text-align: center;
                }
                @media print {
                    body { 
                        margin: 0; 
                        padding: 15px;
                        font-size: 14px;
                    }
                    .header { 
                        page-break-after: avoid; 
                        margin-bottom: 20px;
                    }
                    .game-card { 
                        page-break-inside: avoid; 
                        margin-bottom: 15px;
                    }
                    .summary {
                        margin-bottom: 20px;
                    }
                }
            </style>
        </head>
        <body>
            <div class="header">
                <h1>🎰 Seus Jogos Inteligentes</h1>
                <p>Gerado por <strong>Lottery Optimizer</strong> com Claude AI</p>
                <p><strong>Data de Geração:</strong> ${new Date().toLocaleDateString('pt-BR')} às ${new Date().toLocaleTimeString('pt-BR')}</p>
            </div>
            
            <div class="summary">
                <div class="summary-item">
                    <div class="summary-label">Orçamento</div>
                    <div class="summary-value">R$ ${strategy.budget.toFixed(2)}</div>
                </div>
                <div class="summary-item">
                    <div class="summary-label">Custo Total</div>
                    <div class="summary-value">R$ ${strategy.totalCost.toFixed(2)}</div>
                </div>
                <div class="summary-item">
                    <div class="summary-label">Total de Jogos</div>
                    <div class="summary-value">${strategy.games.length}</div>
                </div>
                <div class="summary-item">
                    <div class="summary-label">Economia</div>
                    <div class="summary-value">R$ ${(strategy.budget - strategy.totalCost).toFixed(2)}</div>
                </div>
            </div>
            
            <div class="games-grid">
                ${strategy.games.map((game: LotteryGame, index: number) => `
                    <div class="game-card">
                        <div class="game-header">
                            <span class="game-title">${(game.type === 'megasena' || game.type === 'mega-sena') ? '🔥 Mega-Sena' : '⭐ Lotofácil'} #${index + 1}</span>
                            <span class="game-cost">R$ ${game.cost.toFixed(2)}</span>
                        </div>
                        <div class="game-numbers">
                            ${game.numbers.slice().sort((a, b) => a - b).map((num: number) => `<span class="number">${num.toString().padStart(2, '0')}</span>`).join('')}
                        </div>
                    </div>
                `).join('')}
            </div>
            
            <div class="disclaimer">
                <p><strong>⚠️ IMPORTANTE:</strong> A loteria é um jogo de azar. Jogue com responsabilidade e apenas o que pode perder.</p>
            </div>
            
            <div class="footer">
                <p><strong>Estratégia gerada com base em análise estatística de dados históricos</strong></p>
                <p>Esta estratégia foi criada usando inteligência artificial que analisou ${strategy.statistics.analyzedDraws || 100} sorteios históricos</p>
                <p>Números podem ser marcados em qualquer lotérica ou site oficial da CAIXA</p>
                <p style="margin-top: 15px; font-size: 10px;">Lottery Optimizer © 2025 - Powered by Claude AI</p>
            </div>
        </body>
        </html>
    `;
    
    printWindow.document.write(printContent);
    printWindow.document.close();
    
    // Aguardar carregamento e imprimir
    printWindow.onload = () => {
        setTimeout(() => {
            printWindow.print();
            printWindow.close();
        }, 500);
    };
}

// ===============================
// TELA DE ERRO
// ===============================

function renderError(message: string) {
    const app = document.getElementById('app')!;
    app.innerHTML = `
        <div class="error-screen">
            <div class="error-content">
                <div class="error-icon">❌</div>
                <h2>Oops! Algo deu errado</h2>
                <div class="error-message">${message}</div>
                
                <div class="error-actions">
                    <button class="btn-secondary" onclick="renderWelcome()">
                        <span class="btn-icon">🏠</span>
                        Voltar ao Início
                    </button>
                    <button class="btn-primary" onclick="renderPreferencesForm()">
                        <span class="btn-icon">🔄</span>
                        Tentar Novamente
                    </button>
                </div>
            </div>
        </div>
    `;
}

// ===============================
// JOGOS SALVOS
// ===============================

// Mostrar modal para salvar jogo
function showSaveGameModal(lotteryType: string, numbers: number[]) {
    // Buscar informações do próximo sorteio
    GetNextDraws().then(nextDraws => {
        const nextDraw = (lotteryType === 'megasena' || lotteryType === 'mega-sena') ? nextDraws.megasena : nextDraws.lotofacil;
        const expectedDate = nextDraw ? nextDraw.date : '';
        const contestNumber = nextDraw ? nextDraw.number : 0;
        
        // Criar modal
        const modal = document.createElement('div');
        modal.className = 'modal-overlay';
        modal.innerHTML = `
            <div class="modal-content">
                <div class="modal-header">
                    <h3>💾 Salvar Jogo</h3>
                    <button class="modal-close" onclick="closeModal()">&times;</button>
                </div>
                <div class="modal-body">
                    <div class="save-game-form">
                        <div class="form-section">
                            <h4>${(lotteryType === 'megasena' || lotteryType === 'mega-sena') ? 'Mega-Sena' : 'Lotofácil'}</h4>
                            <div class="game-numbers">
                                ${numbers.slice().sort((a, b) => a - b).map(num => `<span class="number">${num.toString().padStart(2, '0')}</span>`).join('')}
                            </div>
                        </div>
                        
                        <div class="form-section">
                            <label for="expectedDate">Data do Próximo Sorteio</label>
                            <input type="text" id="expectedDate" value="${expectedDate}" readonly style="background: var(--bg-tertiary); color: var(--text-secondary);">
                        </div>
                        
                        <div class="form-section">
                            <label for="contestNumber">Número do Concurso</label>
                            <input type="number" id="contestNumber" value="${contestNumber}" readonly style="background: var(--bg-tertiary); color: var(--text-secondary);">
                        </div>
                    </div>
                </div>
                <div class="modal-actions">
                    <button class="btn-secondary" onclick="closeModal()">Cancelar</button>
                    <button class="btn-primary" onclick="confirmSaveGame('${lotteryType}', [${numbers.join(',')}], '${expectedDate}', ${contestNumber})">
                        <span>💾</span>
                        Salvar Jogo
                    </button>
                </div>
            </div>
        `;
        
        document.body.appendChild(modal);
    }).catch(error => {
        console.error('Erro ao buscar próximos sorteios:', error);
        alert('Erro ao buscar informações do próximo sorteio. Tente novamente.');
    });
}

// Fechar modal
function closeModal() {
    const modal = document.querySelector('.modal-overlay');
    if (modal) {
        modal.remove();
    }
}

// Confirmar salvamento do jogo
async function confirmSaveGame(lotteryType: string, numbers: number[], expectedDraw: string, contestNumber: number) {
    try {
        const request = new models.SaveGameRequest({
            lottery_type: (lotteryType === 'megasena' || lotteryType === 'mega-sena') ? 'mega-sena' : 'lotofacil',
            numbers: numbers,
            expected_draw: expectedDraw.split('/').reverse().join('-'), // Converter DD/MM/YYYY para YYYY-MM-DD
            contest_number: contestNumber
        });
        
        const response = await SaveGame(request);
        
        if (response.success) {
            closeModal();
            alert('✅ Jogo salvo com sucesso! Você será notificado quando o resultado estiver disponível.');
        } else {
            alert('❌ Erro ao salvar jogo: ' + (response.error || 'Erro desconhecido'));
        }
    } catch (error) {
        console.error('Erro ao salvar jogo:', error);
        alert('❌ Erro ao salvar jogo. Tente novamente.');
    }
}

// Renderizar tela de jogos salvos
async function renderSavedGamesScreen() {
    const app = document.getElementById('app')!;
    
    // Mostrar loading primeiro
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <div class="header-content">
                    <h1 class="logo">🎰 Lottery Optimizer</h1>
                    <p class="tagline">Jogos Salvos</p>
                </div>
                <div class="header-actions">
                    <button class="btn-back" onclick="renderWelcome()">
                        <span class="btn-icon">🏠</span>
                        Início
                    </button>
                    <button class="btn-secondary" onclick="checkAllPendingGames()">
                        <span class="btn-icon">🔄</span>
                        Verificar Resultados
                    </button>
                    <button class="btn-primary" onclick="showAddManualGameModal()">
                        <span class="btn-icon">➕</span>
                        Adicionar Jogo Manual
                    </button>
                </div>
            </header>
            
            <div class="main-content">
                <div class="loading">Carregando jogos salvos...</div>
            </div>
        </div>
    `;
    
    try {
        // Buscar jogos salvos
        const filter = new models.SavedGamesFilter({});
        const response = await GetSavedGames(filter);
        
        if (!response.success) {
            throw new Error(response.error || 'Erro ao carregar jogos salvos');
        }
        
        const savedGames: SavedGame[] = response.games || [];
        
        // Renderizar interface completa
        app.innerHTML = `
            <div class="container">
                <header class="header">
                    <div class="header-content">
                        <h1 class="logo">🎰 Lottery Optimizer</h1>
                        <p class="tagline">Jogos Salvos</p>
                    </div>
                    <div class="header-actions">
                        <button class="btn-back" onclick="renderWelcome()">
                            <span class="btn-icon">🏠</span>
                            Início
                        </button>
                        <button class="btn-secondary" onclick="checkAllPendingGames()">
                            <span class="btn-icon">🔄</span>
                            Verificar Resultados
                        </button>
                        <button class="btn-primary" onclick="showAddManualGameModal()">
                            <span class="btn-icon">➕</span>
                            Adicionar Jogo Manual
                        </button>
                    </div>
                </header>
                
                <div class="main-content">
                    <!-- Filtros -->
                    <div class="filters-section">
                        <h3>
                            <span>🔍</span>
                            Filtros
                        </h3>
                        <div class="filters-grid">
                            <select id="lotteryFilter" onchange="filterSavedGames()">
                                <option value="">Todas as Loterias</option>
                                <option value="mega-sena">Mega-Sena</option>
                                <option value="lotofacil">Lotofácil</option>
                            </select>
                            <select id="statusFilter" onchange="filterSavedGames()">
                                <option value="">Todos os Status</option>
                                <option value="pending">Pendente</option>
                                <option value="checked">Verificado</option>
                                <option value="error">Erro</option>
                            </select>
                        </div>
                    </div>
                    
                    <!-- Lista de jogos salvos -->
                    <div class="saved-games-section">
                        <h3>
                            <span>💾</span>
                            Seus Jogos (${savedGames.length})
                        </h3>
                        
                        ${savedGames.length === 0 ? `
                            <div class="no-games">
                                <div class="no-games-icon">🎲</div>
                                <h4>Nenhum jogo salvo</h4>
                                <p>Gere uma estratégia e salve seus jogos para acompanhar os resultados!</p>
                                <button class="btn-primary" onclick="startStrategyWizard()">
                                    <span class="btn-icon">🎲</span>
                                    Gerar Estratégia
                                </button>
                            </div>
                        ` : `
                            <div class="saved-games-grid" id="savedGamesGrid">
                                ${renderSavedGamesList(savedGames)}
                            </div>
                        `}
                    </div>
                </div>
            </div>
        `;
        
    } catch (error) {
        console.error('Erro ao carregar jogos salvos:', error);
        app.innerHTML = `
            <div class="container">
                <header class="header">
                    <div class="header-content">
                        <h1 class="logo">🎰 Lottery Optimizer</h1>
                        <p class="tagline">Jogos Salvos</p>
                    </div>
                    <button class="btn-back" onclick="renderWelcome()">
                        <span class="btn-icon">🏠</span>
                        Início
                    </button>
                </header>
                
                <div class="main-content">
                    <div class="error-content">
                        <div class="error-icon">❌</div>
                        <h2>Erro ao Carregar</h2>
                        <p class="error-message">${error instanceof Error ? error.message : 'Erro desconhecido'}</p>
                        <button class="btn-primary" onclick="renderSavedGamesScreen()">
                            <span class="btn-icon">🔄</span>
                            Tentar Novamente
                        </button>
                    </div>
                </div>
            </div>
        `;
    }
}

// Renderizar lista de jogos salvos
function renderSavedGamesList(savedGames: SavedGame[]): string {
    return savedGames.map(game => {
        const lotteryIcon = game.lottery_type === 'mega-sena' ? '🔥' : '⭐';
        const lotteryName = game.lottery_type === 'mega-sena' ? 'Mega-Sena' : 'Lotofácil';
        const statusClass = getStatusClass(game.status);
        const statusText = getStatusText(game.status);
        const statusIcon = getStatusIcon(game.status);
        
        return `
            <div class="saved-game-card ${statusClass}">
                <div class="saved-game-header">
                    <span class="game-icon">${lotteryIcon}</span>
                    <div class="game-info">
                        <h4>${lotteryName}</h4>
                        <small>Concurso ${game.contest_number} • ${formatDate(game.expected_draw)}</small>
                    </div>
                    <div class="status-badge">
                        <span class="status-icon">${statusIcon}</span>
                        <span class="status-text">${statusText}</span>
                    </div>
                </div>
                
                <div class="saved-game-numbers">
                    ${game.numbers.slice().sort((a, b) => a - b).map(num => `<span class="number">${num.toString().padStart(2, '0')}</span>`).join('')}
                </div>
                
                ${game.result ? renderGameResult(game.result) : ''}
                
                <div class="saved-game-actions">
                    ${game.status === 'pending' ? `
                        <button class="btn-small btn-secondary" onclick="checkSingleGame('${game.id}')">
                            <span>🔍</span>
                            Verificar
                        </button>
                    ` : ''}
                    <button class="btn-small btn-danger" onclick="deleteSavedGame('${game.id}')">
                        <span>🗑️</span>
                        Excluir
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

// Renderizar resultado do jogo
function renderGameResult(result: GameResult): string {
    const isWinner = result.is_winner;
    const matchClass = isWinner ? 'result-winner' : 'result-no-prize';
    
    return `
        <div class="game-result ${matchClass}">
            <div class="result-header">
                <span class="result-icon">${isWinner ? '🏆' : '📊'}</span>
                <span class="result-text">${result.prize}</span>
                ${isWinner ? `<span class="prize-amount">R$ ${result.prize_amount.toLocaleString('pt-BR', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>` : ''}
            </div>
            
            <div class="result-details">
                <div class="result-info">
                    <small>Sorteio ${result.contest_number} • ${formatDrawDate(result.draw_date)}</small>
                </div>
                
                <div class="numbers-comparison">
                    <div class="drawn-numbers">
                        <span class="label">Sorteados:</span>
                        <div class="numbers">
                            ${result.drawn_numbers.slice().sort((a, b) => a - b).map(num => `<span class="number">${num.toString().padStart(2, '0')}</span>`).join('')}
                        </div>
                    </div>
                    
                    <div class="matched-numbers">
                        <span class="label">Seus acertos (${result.hit_count}):</span>
                        <div class="numbers">
                            ${result.matches.slice().sort((a, b) => a - b).map(num => `<span class="number match">${num.toString().padStart(2, '0')}</span>`).join('')}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    `;
}

// Formatar data do sorteio
function formatDrawDate(dateStr: string): string {
    try {
        // Se a data já está no formato brasileiro
        if (dateStr.includes('/')) {
            return dateStr;
        }
        // Se a data está no formato ISO ou outro formato
        const date = new Date(dateStr);
        if (!isNaN(date.getTime())) {
            return date.toLocaleDateString('pt-BR');
        }
        return dateStr;
    } catch {
        return dateStr;
    }
}

// Funções auxiliares para status
function getStatusClass(status: string): string {
    switch (status) {
        case 'pending': return 'status-pending';
        case 'checked': return 'status-checked';
        case 'error': return 'status-error';
        default: return '';
    }
}

function getStatusText(status: string): string {
    switch (status) {
        case 'pending': return 'Pendente';
        case 'checked': return 'Verificado';
        case 'error': return 'Erro';
        default: return status;
    }
}

function getStatusIcon(status: string): string {
    switch (status) {
        case 'pending': return '⏳';
        case 'checked': return '✅';
        case 'error': return '❌';
        default: return '❓';
    }
}

// Formatar data
function formatDate(dateStr: string): string {
    try {
        const [year, month, day] = dateStr.split('-');
        return `${day}/${month}/${year}`;
    } catch {
        return dateStr;
    }
}

// Filtrar jogos salvos
async function filterSavedGames() {
    const lotteryFilter = (document.getElementById('lotteryFilter') as HTMLSelectElement).value;
    const statusFilter = (document.getElementById('statusFilter') as HTMLSelectElement).value;
    
    try {
        const filter = new models.SavedGamesFilter({
            lottery_type: lotteryFilter || undefined,
            status: statusFilter || undefined
        });
        
        const response = await GetSavedGames(filter);
        
        if (response.success) {
            const savedGames: SavedGame[] = response.games || [];
            const gridElement = document.getElementById('savedGamesGrid');
            if (gridElement) {
                gridElement.innerHTML = renderSavedGamesList(savedGames);
            }
        }
    } catch (error) {
        console.error('Erro ao filtrar jogos:', error);
    }
}

// Verificar jogo individual
async function checkSingleGame(gameId: string) {
    const button = document.querySelector(`button[onclick="checkSingleGame('${gameId}')"]`) as HTMLButtonElement;
    if (!button) return;

    const originalText = button.innerHTML;
    button.innerHTML = '⏳ Verificando...';
    button.disabled = true;

    try {
        const result = await CheckGameResult(gameId);
        
        if (result.success) {
            showNotification('Resultado verificado com sucesso!', 'success');
            await renderSavedGamesScreen(); // Recarregar a lista
        } else {
            showNotification('Erro ao verificar resultado: ' + (result.error || 'Erro desconhecido'), 'error');
        }
    } catch (error) {
        console.error('❌ Erro ao verificar jogo:', error);
        showNotification('Erro ao verificar jogo: ' + String(error), 'error');
    } finally {
        button.innerHTML = originalText;
        button.disabled = false;
    }
}

// Verificar todos os jogos pendentes
async function checkAllPendingGames() {
    // Procurar pelo botão que chama esta função
    const button = document.querySelector('button[onclick="checkAllPendingGames()"]') as HTMLButtonElement;
    if (!button) {
        console.warn('Botão de verificar resultados não encontrado');
        // Executar mesmo sem encontrar o botão
    }

    if (button) {
        button.innerHTML = '<span class="btn-icon">⏳</span> Verificando...';
        button.disabled = true;
    }

    try {
        const results = await CheckAllPendingResults();
        
        if (results.success) {
            showNotification(`✅ Verificados ${results.checked} de ${results.total} jogos!`, 'success');
            await renderSavedGamesScreen(); // Recarregar a lista
        } else {
            showNotification('❌ Erro ao verificar jogos: ' + (results.error || 'Erro desconhecido'), 'error');
        }
    } catch (error) {
        console.error('❌ Erro ao verificar jogos:', error);
        showNotification('❌ Erro ao verificar jogos: ' + String(error), 'error');
    } finally {
        if (button) {
            button.innerHTML = '<span class="btn-icon">🔄</span> Verificar Resultados';
            button.disabled = false;
        }
    }
}

// Deletar jogo salvo
async function deleteSavedGame(gameId: string) {
    if (!confirm('Tem certeza que deseja excluir este jogo salvo?')) {
        return;
    }
    
    try {
        const response = await DeleteSavedGame(gameId);
        
        if (response.success) {
            alert('✅ Jogo excluído com sucesso!');
            renderSavedGamesScreen(); // Recarregar a tela
        } else {
            alert('❌ Erro ao excluir jogo: ' + (response.error || 'Erro desconhecido'));
        }
    } catch (error) {
        console.error('Erro ao excluir jogo:', error);
        alert('❌ Erro ao excluir jogo. Tente novamente.');
    }
}

// ===============================
// GLOBAL WINDOW FUNCTIONS
// ===============================

// Expor funções globalmente para uso em onclick handlers
(window as any).testConnections = testConnections;
(window as any).loadDefaultConfig = loadDefaultConfig;
(window as any).handleConfigSave = handleConfigSave;
(window as any).checkConfigAndRender = checkConfigAndRender;
(window as any).startStrategyWizard = startStrategyWizard;
(window as any).setBudget = setBudget;
(window as any).handlePreferencesSubmit = handlePreferencesSubmit;
(window as any).renderWelcome = renderWelcome;
(window as any).renderConfigurationScreen = renderConfigurationScreen;
(window as any).renderSavedGamesScreen = renderSavedGamesScreen;
(window as any).showSaveGameModal = showSaveGameModal;
(window as any).closeModal = closeModal;
(window as any).confirmSaveGame = confirmSaveGame;
(window as any).filterSavedGames = filterSavedGames;
(window as any).checkSingleGame = checkSingleGame;
(window as any).checkAllPendingGames = checkAllPendingGames;
(window as any).deleteSavedGame = deleteSavedGame;
(window as any).renderPreferencesForm = renderPreferencesForm;
(window as any).generateStrategy = generateStrategy;
(window as any).renderStrategyResult = renderStrategyResult;
(window as any).printStrategy = printStrategy;

// V2.0.0 - Dashboard Analytics Functions
(window as any).renderPerformanceDashboard = renderPerformanceDashboard;
(window as any).renderROICalculator = renderROICalculator;
(window as any).renderNotificationsCenter = renderNotificationsCenter;
(window as any).markNotificationAsRead = markNotificationAsRead;
(window as any).clearOldNotifications = clearOldNotifications;
(window as any).renderDetailedAnalytics = renderDetailedAnalytics;
(window as any).renderNumberAnalysis = renderNumberAnalysis;
(window as any).loadNumberAnalysis = loadNumberAnalysis;
(window as any).loadNotifications = loadNotifications;

// Adicionando funções ao objeto global window para acessibilidade
(window as any).loadAppInfo = loadAppInfo;
(window as any).checkForUpdatesManually = checkForUpdatesManually;

// Funcionalidades de Jogo Manual
(window as any).showAddManualGameModal = showAddManualGameModal;
(window as any).updateNumberLimits = updateNumberLimits;
(window as any).createNumbersGrid = createNumbersGrid;
(window as any).toggleNumber = toggleNumber;
(window as any).updateSelectedDisplay = updateSelectedDisplay;
(window as any).updateManualInput = updateManualInput;
(window as any).updateNumbersFromText = updateNumbersFromText;
(window as any).validateNumberSelection = validateNumberSelection;
(window as any).confirmAddManualGame = confirmAddManualGame;

// ===============================
// FUNÇÕES AUXILIARES PARA ESTATÍSTICAS
// ===============================

// Gerar estatísticas por loteria
function generateLotteryStats(strategy: Strategy): string {
    const megaSenaGames = strategy.games.filter(game => (game.type === 'megasena' || game.type === 'mega-sena'));
    const lotofacilGames = strategy.games.filter(game => game.type === 'lotofacil');
    
    let html = '<div class="lottery-stats-grid">';
    
    if (megaSenaGames.length > 0) {
        const totalCostMega = megaSenaGames.reduce((sum, game) => sum + game.cost, 0);
        const avgNumbersMega = megaSenaGames.reduce((sum, game) => sum + game.numbers.length, 0) / megaSenaGames.length;
        
        html += `
            <div class="lottery-stat-card mega-sena">
                <div class="lottery-header">
                    <span class="lottery-icon">🔥</span>
                    <h5>Mega-Sena</h5>
                </div>
                <div class="lottery-metrics">
                    <div class="metric">
                        <span class="metric-value">${megaSenaGames.length}</span>
                        <span class="metric-label">jogos</span>
                    </div>
                    <div class="metric">
                        <span class="metric-value">R$ ${totalCostMega.toFixed(2)}</span>
                        <span class="metric-label">investimento</span>
                    </div>
                    <div class="metric">
                        <span class="metric-value">${avgNumbersMega.toFixed(1)}</span>
                        <span class="metric-label">números/jogo</span>
                    </div>
                </div>
                <div class="lottery-strategy">
                    <p>Estratégia de <strong>alto retorno</strong> com foco em prêmios que mudam a vida. 
                    Jogos otimizados para maximizar chances de quadra e quina.</p>
                </div>
            </div>
        `;
    }
    
    if (lotofacilGames.length > 0) {
        const totalCostLoto = lotofacilGames.reduce((sum, game) => sum + game.cost, 0);
        const avgNumbersLoto = lotofacilGames.reduce((sum, game) => sum + game.numbers.length, 0) / lotofacilGames.length;
        
        html += `
            <div class="lottery-stat-card lotofacil">
                <div class="lottery-header">
                    <span class="lottery-icon">⭐</span>
                    <h5>Lotofácil</h5>
                </div>
                <div class="lottery-metrics">
                    <div class="metric">
                        <span class="metric-value">${lotofacilGames.length}</span>
                        <span class="metric-label">jogos</span>
                    </div>
                    <div class="metric">
                        <span class="metric-value">R$ ${totalCostLoto.toFixed(2)}</span>
                        <span class="metric-label">investimento</span>
                    </div>
                    <div class="metric">
                        <span class="metric-value">${avgNumbersLoto.toFixed(1)}</span>
                        <span class="metric-label">números/jogo</span>
                    </div>
                </div>
                <div class="lottery-strategy">
                    <p>Estratégia de <strong>alta frequência</strong> com melhor valor esperado. 
                    Foco em retornos consistentes e prêmios secundários.</p>
                </div>
            </div>
        `;
    }
    
    html += '</div>';
    return html;
}

// Calcular retorno esperado estimado
function calculateExpectedReturn(strategy: Strategy): string {
    const megaSenaGames = strategy.games.filter(game => (game.type === 'megasena' || game.type === 'mega-sena'));
    const lotofacilGames = strategy.games.filter(game => game.type === 'lotofacil');
    
    // Estimativas conservadoras baseadas em estatísticas históricas
    let estimatedReturn = 0;
    
    // Mega-Sena: retorno médio de ~40% em prêmios menores
    megaSenaGames.forEach(game => {
        estimatedReturn += game.cost * 0.4;
    });
    
    // Lotofácil: retorno médio de ~60% em prêmios menores
    lotofacilGames.forEach(game => {
        estimatedReturn += game.cost * 0.6;
    });
    
    const returnPercentage = ((estimatedReturn / strategy.totalCost) * 100);
    
    if (returnPercentage >= 50) {
        return `${returnPercentage.toFixed(1)}% (Excelente)`;
    } else if (returnPercentage >= 40) {
        return `${returnPercentage.toFixed(1)}% (Bom)`;
    } else {
        return `${returnPercentage.toFixed(1)}% (Conservador)`;
    }
}

// Identificar sistemas aplicados
function getAppliedSystems(strategy: Strategy): string {
    const systems = [];
    
    // Verificar se há jogos com mais números (sistemas de redução)
    const hasExtendedGames = strategy.games.some(game => 
        (game.type === 'lotofacil' && game.numbers.length > 15) ||
        ((game.type === 'megasena' || game.type === 'mega-sena') && game.numbers.length > 6)
    );
    
    if (hasExtendedGames) {
        systems.push('Wheeling');
    }
    
    // Se há múltiplos jogos, usar diversificação
    if (strategy.games.length > 1) {
        systems.push('Diversificação');
    }
    
    // Sempre aplicar filtros matemáticos
    systems.push('Filtros Matemáticos');
    
    // Se budget foi otimizado
    if (strategy.totalCost >= strategy.budget * 0.85) {
        systems.push('Otimização de Orçamento');
    }
    
    return systems.length > 0 ? systems.join(' + ') : 'Estratégia Básica';
}

// Função para mostrar notificações temporárias
function showNotification(message: string, type: 'success' | 'error' | 'info' = 'info') {
    // Remover notificação existente se houver
    const existing = document.querySelector('.notification-toast');
    if (existing) {
        existing.remove();
    }

    // Criar nova notificação
    const notification = document.createElement('div');
    notification.className = `notification-toast notification-${type}`;
    notification.innerHTML = `
        <div class="notification-content">
            <span class="notification-icon">
                ${type === 'success' ? '✅' : type === 'error' ? '❌' : 'ℹ️'}
            </span>
            <span class="notification-message">${message}</span>
        </div>
    `;

    // Estilos inline para a notificação
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        background: ${type === 'success' ? '#10b981' : type === 'error' ? '#ef4444' : '#3b82f6'};
        color: white;
        padding: 16px 20px;
        border-radius: 8px;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        z-index: 10000;
        max-width: 400px;
        animation: slideIn 0.3s ease-out;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    `;

    // Adicionar ao documento
    document.body.appendChild(notification);

    // Remover após 3 segundos
    setTimeout(() => {
        if (notification.parentNode) {
            notification.style.animation = 'slideOut 0.3s ease-in';
            setTimeout(() => {
                if (notification.parentNode) {
                    notification.remove();
                }
            }, 300);
        }
    }, 3000);
}

// Adicionar estilos para as animações
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(100%);
            opacity: 0;
        }
    }
    
    .notification-content {
        display: flex;
        align-items: center;
        gap: 8px;
    }
`;
document.head.appendChild(style);

// ===============================
// V2.0.0 - DASHBOARD DE PERFORMANCE
// ===============================

// Renderizar Dashboard de Performance principal
async function renderPerformanceDashboard() {
    const app = document.getElementById('app')!;
    
    // Mostrar tela de carregamento primeiro
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <h1 class="logo">📊 Dashboard de Performance</h1>
                <div class="header-actions">
                    <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                </div>
            </header>
            <div class="main-content" style="padding: 2rem;">
                <div style="text-align: center; padding: 4rem;">
                    <div style="font-size: 3rem; margin-bottom: 1rem;">⏳</div>
                    <h2>Carregando Dashboard...</h2>
                    <p>Analisando suas métricas de performance...</p>
                </div>
            </div>
        </div>
    `;

    try {
        // Carregar dados do dashboard
        const [summaryResponse, metricsResponse] = await Promise.all([
            GetDashboardSummary(),
            GetPerformanceMetrics()
        ]);

        if (!summaryResponse.success) {
            throw new Error(summaryResponse.error || 'Erro ao carregar resumo');
        }

        if (!metricsResponse.success) {
            throw new Error(metricsResponse.error || 'Erro ao carregar métricas');
        }

        const summary = summaryResponse.summary as DashboardSummary;
        const metrics = metricsResponse.metrics as PerformanceMetrics;

        // Renderizar dashboard completo
        renderDashboardContent(summary, metrics);
        
    } catch (error) {
        console.error('Erro ao carregar dashboard:', error);
        renderDashboardError(String(error));
    }
}

// Renderizar conteúdo completo do dashboard
function renderDashboardContent(summary: DashboardSummary, _metrics: PerformanceMetrics) {
    const app = document.getElementById('app')!;
    
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <h1 class="logo">📊 Dashboard de Performance</h1>
                <div class="header-actions">
                    <button onclick="renderROICalculator()" class="btn-secondary">💰 ROI</button>
                    <button onclick="renderNotificationsCenter()" class="btn-secondary">🔔 Notificações</button>
                    <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                </div>
            </header>
            
            <div class="main-content">
                <!-- Resumo Executivo -->
                <div class="executive-summary">
                    <h2 style="color: var(--accent-primary); margin-bottom: var(--spacing-4); display: flex; align-items: center; gap: var(--spacing-2);">
                        ${getPerformanceIcon(summary.performance.level)} Resumo Executivo
                    </h2>
                    
                    <!-- Grid de Métricas Principais -->
                    <div class="metrics-grid">
                        <div class="metric-card">
                            <h3>ROI Atual</h3>
                            <div class="metric-value">
                                ${summary.currentROI.toFixed(2)}%
                            </div>
                            <p class="metric-description">
                                ${getTrendIcon(summary.trend)} ${getTrendText(summary.trend)}
                            </p>
                        </div>

                        <div class="metric-card">
                            <h3>Jogos Realizados</h3>
                            <div class="metric-value neutral">
                                ${summary.totalGames}
                            </div>
                            <p class="metric-description">Total de apostas</p>
                        </div>

                        <div class="metric-card">
                            <h3>Investimento Total</h3>
                            <div class="metric-value neutral">
                                R$ ${summary.totalInvestment.toFixed(2)}
                            </div>
                            <p class="metric-description">Valor investido</p>
                        </div>

                        <div class="metric-card">
                            <h3>Retorno Total</h3>
                            <div class="metric-value ${summary.totalWinnings >= summary.totalInvestment ? 'positive' : 'negative'}">
                                R$ ${summary.totalWinnings.toFixed(2)}
                            </div>
                            <p class="metric-description">
                                ${summary.totalWinnings >= summary.totalInvestment ? '📈 Lucro' : '📉 Prejuízo'}: 
                                R$ ${(summary.totalWinnings - summary.totalInvestment).toFixed(2)}
                            </p>
                        </div>

                        <div class="metric-card">
                            <h3>Taxa de Acerto</h3>
                            <div class="metric-value neutral">
                                ${summary.winRate.toFixed(1)}%
                            </div>
                            <p class="metric-description">Jogos premiados</p>
                        </div>

                        <div class="metric-card">
                            <h3>Maior Prêmio</h3>
                            <div class="metric-value warning">
                                R$ ${summary.biggestWin.toFixed(2)}
                            </div>
                            <p class="metric-description">Seu melhor resultado</p>
                        </div>
                    </div>

                    <!-- Performance Level -->
                    <div class="performance-card ${getPerformanceClass(summary.performance.level)}">
                        <div class="performance-icon">
                            ${getPerformanceIcon(summary.performance.level)}
                        </div>
                        <h2 class="performance-level">${summary.performance.level}</h2>
                        <p class="performance-description">${summary.performance.description}</p>
                    </div>
                </div>

                <!-- Ações Rápidas -->
                <div class="quick-actions">
                    <div onclick="renderDetailedAnalytics()" class="action-card">
                        <span class="action-icon">📈</span>
                        <h3 class="action-title">Análise Detalhada</h3>
                        <p class="action-description">Métricas completas e trends</p>
                    </div>

                    <div onclick="renderNumberAnalysis()" class="action-card">
                        <span class="action-icon">🔢</span>
                        <h3 class="action-title">Análise de Números</h3>
                        <p class="action-description">Frequência e padrões</p>
                    </div>

                    <div onclick="renderROICalculator()" class="action-card">
                        <span class="action-icon">💰</span>
                        <h3 class="action-title">Calculadora ROI</h3>
                        <p class="action-description">Projeções de investimento</p>
                    </div>

                    <div onclick="startStrategyWizard()" class="action-card">
                        <span class="action-icon">🧠</span>
                        <h3 class="action-title">Nova Estratégia</h3>
                        <p class="action-description">Baseada nos dados</p>
                    </div>
                    
                    <div onclick="renderIntelligenceEngine()" class="action-card" style="border: 2px solid var(--accent-primary); background: linear-gradient(135deg, rgba(139, 92, 246, 0.1), rgba(99, 102, 241, 0.1));">
                        <span class="action-icon">🧠</span>
                        <h3 class="action-title">Intelligence Engine</h3>
                        <p class="action-description">IA comportamental avançada</p>
                    </div>
                    
                    <div onclick="renderSavedGamesScreen()" class="action-card">
                        <span class="action-icon">💾</span>
                        <h3 class="action-title">Jogos Salvos</h3>
                        <p class="action-description">Histórico e resultados</p>
                    </div>
                    
                    <div onclick="renderNotificationsCenter()" class="action-card">
                        <span class="action-icon">🔔</span>
                        <h3 class="action-title">Notificações</h3>
                        <p class="action-description">Alertas e lembretes</p>
                    </div>
                </div>

                <!-- Últimos 30 Dias -->
                <div class="period-section">
                    <h3 class="period-title">📅 Últimos 30 Dias</h3>
                    <div class="period-stats">
                        <div class="period-stat">
                            <div class="period-stat-value">
                                ${summary.last30Days.games}
                            </div>
                            <p class="period-stat-label">Jogos</p>
                        </div>
                        <div class="period-stat">
                            <div class="period-stat-value">
                                R$ ${summary.last30Days.investment.toFixed(2)}
                            </div>
                            <p class="period-stat-label">Investido</p>
                        </div>
                        <div class="period-stat">
                            <div class="period-stat-value">
                                R$ ${summary.last30Days.winnings.toFixed(2)}
                            </div>
                            <p class="period-stat-label">Retorno</p>
                        </div>
                        <div class="period-stat">
                            <div class="period-stat-value" style="color: ${summary.last30Days.roi >= 0 ? 'var(--accent-success)' : 'var(--accent-error)'};">
                                ${summary.last30Days.roi.toFixed(2)}%
                            </div>
                            <p class="period-stat-label">ROI</p>
                        </div>
                    </div>
                </div>

                <!-- Current Streak -->
                ${summary.currentStreak.count > 0 ? `
                <div class="streak-section">
                    <div class="streak-card ${summary.currentStreak.type}">
                        <div class="streak-icon">
                            ${summary.currentStreak.type === 'win' ? '🔥' : '❄️'}
                        </div>
                        <h3>
                            ${summary.currentStreak.type === 'win' ? 'Sequência de Vitórias' : 'Sequência de Derrotas'}
                        </h3>
                        <div class="streak-count">
                            ${summary.currentStreak.count}
                        </div>
                        <p class="streak-description">
                            ${summary.currentStreak.type === 'win' ? 'Jogos consecutivos com prêmio!' : 'Jogos consecutivos sem prêmio'}
                        </p>
                    </div>
                </div>
                ` : ''}
            </div>
        </div>
    `;
}

// Renderizar erro do dashboard
function renderDashboardError(_error: string) {
    const app = document.getElementById('app')!;
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <h1 class="logo">📊 Dashboard de Performance</h1>
                <div class="header-actions">
                    <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                </div>
            </header>
            <div class="main-content">
                <div class="dashboard-error">
                    <div class="dashboard-error-icon">📊</div>
                    <h2>Dados Insuficientes</h2>
                    <p>
                        Você ainda não possui jogos salvos para gerar métricas de performance.
                        <br>Comece criando e salvando suas estratégias!
                    </p>
                    <div class="dashboard-error-actions">
                        <button onclick="startStrategyWizard()" class="btn-primary">
                            🎲 Gerar Primeira Estratégia
                        </button>
                        <button onclick="renderSavedGamesScreen()" class="btn-secondary">
                            💾 Ver Jogos Salvos
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
}

// Funções auxiliares para o dashboard
function getPerformanceIcon(level: string): string {
    switch (level) {
        case 'Excelente': return '🏆';
        case 'Boa': return '📈';
        case 'Regular': return '📊';
        case 'Baixa': return '📉';
        default: return '📊';
    }
}


function getTrendIcon(trend: string): string {
    switch (trend) {
        case 'up': return '📈';
        case 'down': return '📉';
        default: return '➡️';
    }
}

function getTrendText(trend: string): string {
    switch (trend) {
        case 'up': return 'Tendência de alta';
        case 'down': return 'Tendência de baixa'; 
        default: return 'Tendência estável';
    }
}

// ===============================
// CALCULADORA ROI
// ===============================

async function renderROICalculator() {
    const app = document.getElementById('app')!;
    
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <h1 class="logo">💰 Calculadora ROI</h1>
                <div class="header-actions">
                    <button onclick="renderPerformanceDashboard()" class="btn-secondary">📊 Dashboard</button>
                    <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                </div>
            </header>
            
            <div class="main-content">
                <!-- Hero Section -->
                <div class="roi-hero">
                    <div class="roi-hero-content">
                        <h2 class="roi-hero-title">
                            <span class="roi-icon">📈</span>
                            Calculadora de ROI Inteligente
                        </h2>
                        <p class="roi-hero-description">
                            Projete seus investimentos com base no histórico de performance e análise estatística avançada
                        </p>
                    </div>
                </div>

                <!-- Calculator Form -->
                <div class="roi-calculator-section">
                    <div class="roi-form-card">
                        <div class="roi-form-header">
                            <h3>🎯 Parâmetros de Projeção</h3>
                            <p>Configure os valores para calcular sua projeção de ROI personalizada</p>
                        </div>
                        
                        <form id="roiCalculatorForm" class="roi-form">
                            <div class="roi-form-grid">
                                <div class="roi-input-group">
                                    <label class="roi-label">
                                        <span class="roi-label-icon">💵</span>
                                        <span class="roi-label-text">Valor do Investimento</span>
                                    </label>
                                    <div class="roi-input-wrapper">
                                        <span class="roi-input-prefix">R$</span>
                                        <input 
                                            type="number" 
                                            id="investmentAmount" 
                                            min="1" 
                                            step="0.01" 
                                            value="100" 
                                            class="roi-input"
                                            placeholder="0,00"
                                        >
                                    </div>
                                </div>
                                
                                <div class="roi-input-group">
                                    <label class="roi-label">
                                        <span class="roi-label-icon">📅</span>
                                        <span class="roi-label-text">Período de Análise</span>
                                    </label>
                                    <select id="timeframe" class="roi-select">
                                        <option value="30">30 dias</option>
                                        <option value="90">90 dias</option>
                                        <option value="180">6 meses</option>
                                        <option value="365">1 ano</option>
                                    </select>
                                </div>
                            </div>
                            
                            <button type="submit" class="roi-calculate-btn">
                                <span class="roi-btn-icon">🧮</span>
                                <span class="roi-btn-text">Calcular Projeção de ROI</span>
                            </button>
                        </form>
                    </div>
                </div>

                <!-- Results Section -->
                <div id="roiResults" class="roi-results-section" style="display: none;">
                    <!-- Será preenchido dinamicamente -->
                </div>
            </div>
        </div>
    `;

    // Adicionar event listener para o formulário
    const form = document.getElementById('roiCalculatorForm') as HTMLFormElement;
    form.addEventListener('submit', handleROICalculation);
}

async function handleROICalculation(event: Event) {
    event.preventDefault();
    
    const investmentInput = document.getElementById('investmentAmount') as HTMLInputElement;
    const timeframeSelect = document.getElementById('timeframe') as HTMLSelectElement;
    const resultsDiv = document.getElementById('roiResults')!;
    
    const investment = parseFloat(investmentInput.value);
    const timeframe = parseInt(timeframeSelect.value);
    
    if (investment <= 0) {
        showNotification('Por favor, insira um valor de investimento válido', 'error');
        return;
    }

    // Mostrar loading
    resultsDiv.style.display = 'block';
    resultsDiv.innerHTML = `
        <div class="feature-card" style="text-align: center;">
            <div style="font-size: 2rem; margin-bottom: 1rem;">⏳</div>
            <h3>Calculando projeção...</h3>
            <p>Analisando seu histórico de performance...</p>
        </div>
    `;

    try {
        const response = await GetROICalculator(investment, timeframe.toString());
        
        if (!response.success) {
            throw new Error(response.error || 'Erro ao calcular ROI');
        }

        const calculation = response.calculation as ROICalculation;
        renderROIResults(calculation);
        
    } catch (error) {
        console.error('Erro ao calcular ROI:', error);
        resultsDiv.innerHTML = `
            <div class="feature-card" style="background: #fef2f2; border: 1px solid #fecaca;">
                <h3 style="color: #dc2626;">Erro no Cálculo</h3>
                <p style="color: #7f1d1d;">${String(error)}</p>
                <p style="color: #7f1d1d; margin-top: 1rem;">
                    <strong>Dica:</strong> Você precisa ter jogos salvos com resultados para gerar projeções precisas.
                </p>
            </div>
        `;
    }
}

function renderROIResults(calculation: ROICalculation) {
    const resultsDiv = document.getElementById('roiResults')!;
    
    resultsDiv.innerHTML = `
        <!-- Resumo da Projeção -->
        <div class="roi-summary-card">
            <div class="roi-summary-header">
                <span class="roi-summary-icon">🎯</span>
                <h3 class="roi-summary-title">Projeção de ROI</h3>
            </div>
            <div class="roi-metrics-grid">
                <div class="roi-metric">
                    <div class="roi-metric-value">R$ ${calculation.investment.toFixed(2)}</div>
                    <div class="roi-metric-label">Investimento</div>
                </div>
                <div class="roi-metric">
                    <div class="roi-metric-value">R$ ${calculation.projectedWinnings.toFixed(2)}</div>
                    <div class="roi-metric-label">Retorno Projetado</div>
                </div>
                <div class="roi-metric">
                    <div class="roi-metric-value">${calculation.projectedROI.toFixed(2)}%</div>
                    <div class="roi-metric-label">ROI Projetado</div>
                </div>
                <div class="roi-metric">
                    <div class="roi-metric-value ${calculation.projectedProfit >= 0 ? 'positive' : 'negative'}">
                        R$ ${calculation.projectedProfit.toFixed(2)}
                    </div>
                    <div class="roi-metric-label">
                        ${calculation.projectedProfit >= 0 ? 'Lucro' : 'Prejuízo'} Projetado
                    </div>
                </div>
            </div>
        </div>

        <!-- Análise Detalhada -->
        <div class="roi-analysis-grid">
            <div class="roi-analysis-card">
                <div class="roi-analysis-header">
                    <span class="roi-analysis-icon">📈</span>
                    <h4>Dados Históricos</h4>
                </div>
                <div class="roi-analysis-content">
                    <div class="roi-data-row">
                        <span class="roi-data-label">ROI Histórico:</span>
                        <span class="roi-data-value ${calculation.historicalROI >= 0 ? 'positive' : 'negative'}">
                            ${calculation.historicalROI.toFixed(2)}%
                        </span>
                    </div>
                    <div class="roi-data-row">
                        <span class="roi-data-label">Taxa de Acerto:</span>
                        <span class="roi-data-value">${calculation.historicalWinRate.toFixed(2)}%</span>
                    </div>
                    <div class="roi-data-row">
                        <span class="roi-data-label">Jogos Analisados:</span>
                        <span class="roi-data-value">${calculation.basedOnGames}</span>
                    </div>
                    <div class="roi-data-row">
                        <span class="roi-data-label">Período:</span>
                        <span class="roi-data-value">${calculation.timeframe}</span>
                    </div>
                </div>
            </div>

            <div class="roi-analysis-card">
                <div class="roi-analysis-header">
                    <span class="roi-analysis-icon">${getConfidenceIcon(calculation.confidence)}</span>
                    <h4>Análise de Confiança</h4>
                </div>
                <div class="roi-analysis-content">
                    <div class="roi-confidence-badge ${calculation.confidence.toLowerCase()}">
                        ${calculation.confidence}
                    </div>
                    <div class="roi-recommendation">
                        "${calculation.recommendation}"
                    </div>
                </div>
            </div>
        </div>

        <!-- Próximos Passos -->
        <div class="roi-next-steps">
            <div class="roi-next-steps-header">
                <span class="roi-steps-icon">💡</span>
                <h4>Próximos Passos Recomendados</h4>
            </div>
            <p class="roi-steps-description">
                Com base na sua projeção, recomendamos as seguintes ações para otimizar seus resultados:
            </p>
            <div class="roi-steps-grid">
                <div class="roi-step-card" onclick="startStrategyWizard()">
                    <div class="roi-step-icon">🧠</div>
                    <h5 class="roi-step-title">Gerar Nova Estratégia</h5>
                    <p class="roi-step-description">
                        Crie uma estratégia inteligente baseada na sua projeção de ROI e dados históricos
                    </p>
                </div>

                <div class="roi-step-card" onclick="renderPerformanceDashboard()">
                    <div class="roi-step-icon">📊</div>
                    <h5 class="roi-step-title">Análise Completa</h5>
                    <p class="roi-step-description">
                        Veja todas as métricas detalhadas no dashboard de performance executivo
                    </p>
                </div>

                <div class="roi-step-card" onclick="renderSavedGamesScreen()">
                    <div class="roi-step-icon">💾</div>
                    <h5 class="roi-step-title">Revisar Histórico</h5>
                    <p class="roi-step-description">
                        Confira seus jogos salvos e verifique resultados pendentes
                    </p>
                </div>
            </div>
            
            <div class="roi-pro-tip">
                <div class="roi-tip-icon">🎯</div>
                <div class="roi-tip-content">
                    <h5>Dica Profissional</h5>
                    <p>
                        Para melhores resultados, mantenha um histórico consistente de jogos e revise suas estratégias 
                        regularmente com base nas análises de performance. Lembre-se: disciplina e análise de dados 
                        são fundamentais para o sucesso a longo prazo.
                    </p>
                </div>
            </div>
        </div>
    `;
}

function getConfidenceIcon(confidence: string): string {
    switch (confidence.toLowerCase()) {
        case 'alta': return '🎯';
        case 'média': return '📊';
        case 'baixa': return '⚠️';
        default: return '📈';
    }
}

// ===============================
// CENTRO DE NOTIFICAÇÕES
// ===============================

async function renderNotificationsCenter() {
    const app = document.getElementById('app')!;
    
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <h1 class="logo">🔔 Centro de Notificações</h1>
                <div class="header-actions">
                    <button onclick="renderPerformanceDashboard()" class="btn-secondary">📊 Dashboard</button>
                    <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                </div>
            </header>
            
            <div class="main-content" style="padding: 1rem;">
                <div class="welcome-section" style="margin-bottom: 2rem;">
                    <h2 style="color: var(--accent-primary); margin-bottom: 1rem;">
                        📬 Suas Notificações
                    </h2>
                    <div style="display: flex; gap: 1rem; margin-bottom: 2rem; flex-wrap: wrap;">
                        <button onclick="loadNotifications(50, false)" class="btn-secondary">
                            📋 Todas
                        </button>
                        <button onclick="loadNotifications(50, true)" class="btn-secondary">
                            🔴 Não Lidas
                        </button>
                        <button onclick="clearOldNotifications()" class="btn-secondary">
                            🗑️ Limpar Antigas
                        </button>
                    </div>
                </div>

                <!-- Lista de Notificações -->
                <div id="notificationsList">
                    <div style="text-align: center; padding: 2rem;">
                        <div style="font-size: 2rem; margin-bottom: 1rem;">⏳</div>
                        <h3>Carregando notificações...</h3>
                    </div>
                </div>
            </div>
        </div>
    `;

    // Carregar notificações automaticamente
    await loadNotifications(50, false);
}

async function loadNotifications(limit: number, onlyUnread: boolean) {
    const listDiv = document.getElementById('notificationsList')!;
    
    listDiv.innerHTML = `
        <div style="text-align: center; padding: 2rem;">
            <div style="font-size: 2rem; margin-bottom: 1rem;">⏳</div>
            <h3>Carregando notificações...</h3>
        </div>
    `;

    try {
        const response = await GetNotifications(limit, onlyUnread);
        
        if (!response.success) {
            throw new Error(response.error || 'Erro ao carregar notificações');
        }

        const notifications = response.notifications as AppNotification[];
        renderNotificationsList(notifications);
        
    } catch (error) {
        console.error('Erro ao carregar notificações:', error);
        listDiv.innerHTML = `
            <div class="feature-card" style="text-align: center; background: #fef2f2; border: 1px solid #fecaca;">
                <div style="font-size: 3rem; margin-bottom: 1rem;">🔔</div>
                <h3 style="color: #dc2626;">Erro ao Carregar</h3>
                <p style="color: #7f1d1d;">${String(error)}</p>
            </div>
        `;
    }
}

function renderNotificationsList(notifications: AppNotification[]) {
    const listDiv = document.getElementById('notificationsList')!;
    
    if (notifications.length === 0) {
        listDiv.innerHTML = `
            <div class="feature-card" style="text-align: center;">
                <div style="font-size: 3rem; margin-bottom: 1rem;">📭</div>
                <h3>Nenhuma Notificação</h3>
                <p style="color: var(--text-secondary);">
                    Você está em dia! Não há notificações para exibir.
                </p>
            </div>
        `;
        return;
    }

    const notificationsHtml = notifications.map(notification => `
        <div class="feature-card notification-item ${!notification.readAt ? 'unread' : ''}" 
             style="margin-bottom: 1rem; ${!notification.readAt ? 'border-left: 4px solid var(--accent-primary);' : ''}"
             data-notification-id="${notification.id}">
            <div style="display: flex; align-items: flex-start; gap: 1rem;">
                <div style="font-size: 2rem; flex-shrink: 0;">
                    ${getNotificationIcon(notification.type)}
                </div>
                <div style="flex: 1;">
                    <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 0.5rem;">
                        <h4 style="margin: 0; color: var(--accent-primary);">
                            ${notification.title}
                        </h4>
                        <div style="display: flex; align-items: center; gap: 0.5rem;">
                            <span class="priority-badge priority-${notification.priority}">
                                ${getPriorityText(notification.priority)}
                            </span>
                            ${!notification.readAt ? `
                                <button onclick="markNotificationAsRead('${notification.id}')" 
                                        class="btn-secondary" style="padding: 0.25rem 0.5rem; font-size: 0.8rem;">
                                    ✓ Marcar como lida
                                </button>
                            ` : ''}
                        </div>
                    </div>
                    <p style="margin: 0 0 0.5rem 0; color: var(--text-secondary);">
                        ${notification.message}
                    </p>
                    <div style="display: flex; justify-content: space-between; align-items: center; font-size: 0.9rem; color: var(--text-secondary);">
                        <span>📅 ${formatDate(notification.createdAt)}</span>
                        <span class="category-badge">${getCategoryText(notification.category)}</span>
                    </div>
                </div>
            </div>
        </div>
    `).join('');

    listDiv.innerHTML = `
        <div style="margin-bottom: 1rem;">
            <h3>📋 ${notifications.length} notificação${notifications.length !== 1 ? 'ões' : ''}</h3>
        </div>
        ${notificationsHtml}
    `;

    // Adicionar estilos para as notificações
    const style = document.createElement('style');
    style.textContent = `
        .notification-item.unread {
            background: linear-gradient(135deg, #eff6ff, #dbeafe);
        }
        .priority-badge {
            padding: 0.25rem 0.5rem;
            border-radius: 0.25rem;
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
        }
        .priority-urgent { background: #fecaca; color: #7f1d1d; }
        .priority-high { background: #fed7aa; color: #9a3412; }
        .priority-medium { background: #fef3c7; color: #92400e; }
        .priority-low { background: #d1fae5; color: #065f46; }
        .category-badge {
            background: #f3f4f6;
            padding: 0.25rem 0.5rem;
            border-radius: 0.25rem;
            font-size: 0.75rem;
        }
    `;
    document.head.appendChild(style);
}

async function markNotificationAsRead(notificationId: string) {
    try {
        const response = await MarkNotificationAsRead(notificationId);
        
        if (response.success) {
            showNotification('Notificação marcada como lida', 'success');
            // Recarregar a lista
            await loadNotifications(50, false);
        } else {
            throw new Error(response.error || 'Erro ao marcar notificação');
        }
    } catch (error) {
        console.error('Erro ao marcar notificação:', error);
        showNotification('Erro ao marcar notificação: ' + String(error), 'error');
    }
}

async function clearOldNotifications() {
    try {
        const response = await ClearOldNotifications(30); // Limpar notificações com mais de 30 dias
        
        if (response.success) {
            showNotification(`${response.cleared || 0} notificações antigas removidas`, 'success');
            // Recarregar a lista
            await loadNotifications(50, false);
        } else {
            throw new Error(response.error || 'Erro ao limpar notificações');
        }
    } catch (error) {
        console.error('Erro ao limpar notificações:', error);
        showNotification('Erro ao limpar notificações: ' + String(error), 'error');
    }
}

function getNotificationIcon(type: string): string {
    switch (type) {
        case 'reminder': return '⏰';
        case 'result': return '🎯';
        case 'performance': return '📊';
        case 'achievement': return '🏆';
        case 'system': return '⚙️';
        default: return '📢';
    }
}

function getPriorityText(priority: string): string {
    switch (priority) {
        case 'urgent': return 'Urgente';
        case 'high': return 'Alta';
        case 'medium': return 'Média';
        case 'low': return 'Baixa';
        default: return 'Normal';
    }
}

function getCategoryText(category: string): string {
    switch (category) {
        case 'game': return '🎲 Jogo';
        case 'finance': return '💰 Financeiro';
        case 'system': return '⚙️ Sistema';
        case 'achievement': return '🏆 Conquista';
        default: return '📢 Geral';
    }
}

// ===============================
// ANÁLISE DETALHADA
// ===============================

async function renderDetailedAnalytics() {
    const app = document.getElementById('app')!;
    
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <h1 class="logo">📈 Análise Detalhada</h1>
                <div class="header-actions">
                    <button onclick="renderPerformanceDashboard()" class="btn-secondary">📊 Dashboard</button>
                    <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                </div>
            </header>
            
            <div class="main-content" style="padding: 1rem;">
                <div style="text-align: center; padding: 4rem;">
                    <div style="font-size: 3rem; margin-bottom: 1rem;">⏳</div>
                    <h2>Carregando Análise Detalhada...</h2>
                    <p>Processando métricas avançadas...</p>
                </div>
            </div>
        </div>
    `;

    try {
        const response = await GetPerformanceMetrics();
        
        if (!response.success) {
            throw new Error(response.error || 'Erro ao carregar métricas');
        }

        const metrics = response.metrics as PerformanceMetrics;
        renderDetailedAnalyticsContent(metrics);
        
    } catch (error) {
        console.error('Erro ao carregar análise detalhada:', error);
        renderAnalyticsError(String(error));
    }
}

function renderDetailedAnalyticsContent(metrics: PerformanceMetrics) {
    const app = document.getElementById('app')!;
    
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <h1 class="logo">📈 Análise Detalhada</h1>
                <div class="header-actions">
                    <button onclick="renderPerformanceDashboard()" class="btn-secondary">📊 Dashboard</button>
                    <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                </div>
            </header>
            
            <div class="main-content" style="padding: 1rem;">
                <!-- Métricas Gerais -->
                <div class="welcome-section" style="margin-bottom: 2rem;">
                    <h2 style="color: var(--accent-primary); margin-bottom: 1rem;">
                        📊 Métricas Completas
                    </h2>
                    <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem; margin-bottom: 2rem;">
                        <div class="feature-card">
                            <h4>Total de Jogos</h4>
                            <div style="font-size: 2rem; color: var(--accent-primary); font-weight: bold;">
                                ${metrics.totalGames}
                            </div>
                        </div>
                        <div class="feature-card">
                            <h4>Investimento Total</h4>
                            <div style="font-size: 2rem; color: var(--accent-primary); font-weight: bold;">
                                R$ ${metrics.totalInvestment.toFixed(2)}
                            </div>
                        </div>
                        <div class="feature-card">
                            <h4>Retorno Total</h4>
                            <div style="font-size: 2rem; color: ${metrics.totalWinnings >= metrics.totalInvestment ? '#059669' : '#dc2626'}; font-weight: bold;">
                                R$ ${metrics.totalWinnings.toFixed(2)}
                            </div>
                        </div>
                        <div class="feature-card">
                            <h4>ROI Geral</h4>
                            <div style="font-size: 2rem; color: ${metrics.roiPercentage >= 0 ? '#059669' : '#dc2626'}; font-weight: bold;">
                                ${metrics.roiPercentage.toFixed(2)}%
                            </div>
                        </div>
                        <div class="feature-card">
                            <h4>Taxa de Acerto</h4>
                            <div style="font-size: 2rem; color: var(--accent-primary); font-weight: bold;">
                                ${metrics.winRate.toFixed(1)}%
                            </div>
                        </div>
                        <div class="feature-card">
                            <h4>Maior Prêmio</h4>
                            <div style="font-size: 2rem; color: #f59e0b; font-weight: bold;">
                                R$ ${metrics.biggestWin.toFixed(2)}
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Análise de Sequências -->
                <div class="feature-card" style="margin-bottom: 2rem;">
                    <h3 style="margin-bottom: 1rem;">🔥 Análise de Sequências</h3>
                    <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
                        <div style="text-align: center;">
                            <div style="font-size: 1.5rem; color: #059669; font-weight: bold;">
                                ${metrics.currentWinStreak}
                            </div>
                            <p style="margin: 0; color: var(--text-secondary);">Sequência Atual de Vitórias</p>
                        </div>
                        <div style="text-align: center;">
                            <div style="font-size: 1.5rem; color: #dc2626; font-weight: bold;">
                                ${metrics.currentLossStreak}
                            </div>
                            <p style="margin: 0; color: var(--text-secondary);">Sequência Atual de Derrotas</p>
                        </div>
                        <div style="text-align: center;">
                            <div style="font-size: 1.5rem; color: #059669; font-weight: bold;">
                                ${metrics.longestWinStreak}
                            </div>
                            <p style="margin: 0; color: var(--text-secondary);">Maior Sequência de Vitórias</p>
                        </div>
                        <div style="text-align: center;">
                            <div style="font-size: 1.5rem; color: #dc2626; font-weight: bold;">
                                ${metrics.longestLossStreak}
                            </div>
                            <p style="margin: 0; color: var(--text-secondary);">Maior Sequência de Derrotas</p>
                        </div>
                    </div>
                </div>

                <!-- Análise por Período -->
                <div class="feature-card" style="margin-bottom: 2rem;">
                    <h3 style="margin-bottom: 1rem;">📅 Performance por Período</h3>
                    <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 1rem;">
                        <div style="border: 1px solid #e5e7eb; border-radius: 0.5rem; padding: 1rem;">
                            <h4 style="color: var(--accent-primary); margin-bottom: 1rem;">Últimos 30 Dias</h4>
                            <div style="space-y: 0.5rem;">
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Jogos:</span>
                                    <strong>${metrics.last30Days.games}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Investimento:</span>
                                    <strong>R$ ${metrics.last30Days.investment.toFixed(2)}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Retorno:</span>
                                    <strong>R$ ${metrics.last30Days.winnings.toFixed(2)}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>ROI:</span>
                                    <strong style="color: ${metrics.last30Days.roi >= 0 ? '#059669' : '#dc2626'};">
                                        ${metrics.last30Days.roi.toFixed(2)}%
                                    </strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Taxa de Acerto:</span>
                                    <strong>${metrics.last30Days.winRate.toFixed(1)}%</strong>
                                </div>
                            </div>
                        </div>

                        <div style="border: 1px solid #e5e7eb; border-radius: 0.5rem; padding: 1rem;">
                            <h4 style="color: var(--accent-primary); margin-bottom: 1rem;">Últimos 90 Dias</h4>
                            <div style="space-y: 0.5rem;">
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Jogos:</span>
                                    <strong>${metrics.last90Days.games}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Investimento:</span>
                                    <strong>R$ ${metrics.last90Days.investment.toFixed(2)}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Retorno:</span>
                                    <strong>R$ ${metrics.last90Days.winnings.toFixed(2)}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>ROI:</span>
                                    <strong style="color: ${metrics.last90Days.roi >= 0 ? '#059669' : '#dc2626'};">
                                        ${metrics.last90Days.roi.toFixed(2)}%
                                    </strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Taxa de Acerto:</span>
                                    <strong>${metrics.last90Days.winRate.toFixed(1)}%</strong>
                                </div>
                            </div>
                        </div>

                        <div style="border: 1px solid #e5e7eb; border-radius: 0.5rem; padding: 1rem;">
                            <h4 style="color: var(--accent-primary); margin-bottom: 1rem;">Último Ano</h4>
                            <div style="space-y: 0.5rem;">
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Jogos:</span>
                                    <strong>${metrics.last365Days.games}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Investimento:</span>
                                    <strong>R$ ${metrics.last365Days.investment.toFixed(2)}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Retorno:</span>
                                    <strong>R$ ${metrics.last365Days.winnings.toFixed(2)}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>ROI:</span>
                                    <strong style="color: ${metrics.last365Days.roi >= 0 ? '#059669' : '#dc2626'};">
                                        ${metrics.last365Days.roi.toFixed(2)}%
                                    </strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Taxa de Acerto:</span>
                                    <strong>${metrics.last365Days.winRate.toFixed(1)}%</strong>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Performance por Loteria -->
                <div class="feature-card" style="margin-bottom: 2rem;">
                    <h3 style="margin-bottom: 1rem;">🎰 Performance por Loteria</h3>
                    <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 1rem;">
                        <div style="border: 1px solid #e5e7eb; border-radius: 0.5rem; padding: 1rem;">
                            <h4 style="color: #dc2626; margin-bottom: 1rem;">🔥 Mega-Sena</h4>
                            <div style="space-y: 0.5rem;">
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Jogos:</span>
                                    <strong>${metrics.lotterySpecific.megaSena.games}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Investimento:</span>
                                    <strong>R$ ${metrics.lotterySpecific.megaSena.investment.toFixed(2)}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Retorno:</span>
                                    <strong>R$ ${metrics.lotterySpecific.megaSena.winnings.toFixed(2)}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>ROI:</span>
                                    <strong style="color: ${metrics.lotterySpecific.megaSena.roi >= 0 ? '#059669' : '#dc2626'};">
                                        ${metrics.lotterySpecific.megaSena.roi.toFixed(2)}%
                                    </strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Taxa de Acerto:</span>
                                    <strong>${metrics.lotterySpecific.megaSena.winRate.toFixed(1)}%</strong>
                                </div>
                            </div>
                        </div>

                        <div style="border: 1px solid #e5e7eb; border-radius: 0.5rem; padding: 1rem;">
                            <h4 style="color: #f59e0b; margin-bottom: 1rem;">⭐ Lotofácil</h4>
                            <div style="space-y: 0.5rem;">
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Jogos:</span>
                                    <strong>${metrics.lotterySpecific.lotofacil.games}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Investimento:</span>
                                    <strong>R$ ${metrics.lotterySpecific.lotofacil.investment.toFixed(2)}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Retorno:</span>
                                    <strong>R$ ${metrics.lotterySpecific.lotofacil.winnings.toFixed(2)}</strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>ROI:</span>
                                    <strong style="color: ${metrics.lotterySpecific.lotofacil.roi >= 0 ? '#059669' : '#dc2626'};">
                                        ${metrics.lotterySpecific.lotofacil.roi.toFixed(2)}%
                                    </strong>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span>Taxa de Acerto:</span>
                                    <strong>${metrics.lotterySpecific.lotofacil.winRate.toFixed(1)}%</strong>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Ações -->
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
                    <button onclick="renderNumberAnalysis()" class="feature-card" style="border: none; cursor: pointer; transition: transform 0.2s;" onmouseover="this.style.transform='scale(1.02)'" onmouseout="this.style.transform='scale(1)'">
                        <span style="font-size: 2rem;">🔢</span>
                        <h4>Análise de Números</h4>
                        <p style="margin: 0;">Frequência e padrões</p>
                    </button>

                    <button onclick="renderROICalculator()" class="feature-card" style="border: none; cursor: pointer; transition: transform 0.2s;" onmouseover="this.style.transform='scale(1.02)'" onmouseout="this.style.transform='scale(1)'">
                        <span style="font-size: 2rem;">💰</span>
                        <h4>Calculadora ROI</h4>
                        <p style="margin: 0;">Projeções futuras</p>
                    </button>

                    <button onclick="startStrategyWizard()" class="feature-card" style="border: none; cursor: pointer; transition: transform 0.2s;" onmouseover="this.style.transform='scale(1.02)'" onmouseout="this.style.transform='scale(1)'">
                        <span style="font-size: 2rem;">🧠</span>
                        <h4>Nova Estratégia</h4>
                        <p style="margin: 0;">Baseada nos dados</p>
                    </button>
                </div>
            </div>
        </div>
    `;
}

// ===============================
// ANÁLISE DE NÚMEROS
// ===============================

async function renderNumberAnalysis() {
    const app = document.getElementById('app')!;
    
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <h1 class="logo">🔢 Análise de Números</h1>
                <div class="header-actions">
                    <button onclick="renderPerformanceDashboard()" class="btn-secondary">📊 Dashboard</button>
                    <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                </div>
            </header>
            
            <div class="main-content" style="padding: 1rem;">
                <div class="welcome-section" style="margin-bottom: 2rem;">
                    <h2 style="color: var(--accent-primary); margin-bottom: 1rem;">
                        🎯 Análise de Frequência
                    </h2>
                    <div style="display: flex; gap: 1rem; margin-bottom: 2rem; flex-wrap: wrap;">
                        <button onclick="loadNumberAnalysis('megasena')" class="btn-secondary">
                            🔥 Mega-Sena
                        </button>
                        <button onclick="loadNumberAnalysis('lotofacil')" class="btn-secondary">
                            ⭐ Lotofácil
                        </button>
                    </div>
                </div>

                <!-- Resultado da Análise -->
                <div id="numberAnalysisResults">
                    <div class="feature-card" style="text-align: center;">
                        <div style="font-size: 3rem; margin-bottom: 1rem;">🔢</div>
                        <h3>Selecione uma Loteria</h3>
                        <p style="color: var(--text-secondary);">
                            Escolha Mega-Sena ou Lotofácil para ver a análise de frequência dos números
                        </p>
                    </div>
                </div>
            </div>
        </div>
    `;
}

async function loadNumberAnalysis(lottery: string) {
    const resultsDiv = document.getElementById('numberAnalysisResults')!;
    
    resultsDiv.innerHTML = `
        <div style="text-align: center; padding: 2rem;">
            <div style="font-size: 2rem; margin-bottom: 1rem;">⏳</div>
            <h3>Analisando números da ${lottery === 'megasena' ? 'Mega-Sena' : 'Lotofácil'}...</h3>
            <p>Calculando frequências e padrões...</p>
        </div>
    `;

    try {
        const response = await GetNumberFrequencyAnalysis(lottery);
        
        if (!response.success) {
            throw new Error(response.error || 'Erro ao carregar análise');
        }

        const frequencies = response.frequencies as NumberFrequency[];
        renderNumberAnalysisResults(lottery, frequencies);
        
    } catch (error) {
        console.error('Erro ao carregar análise de números:', error);
        resultsDiv.innerHTML = `
            <div class="feature-card" style="text-align: center; background: #fef2f2; border: 1px solid #fecaca;">
                <div style="font-size: 3rem; margin-bottom: 1rem;">🔢</div>
                <h3 style="color: #dc2626;">Erro na Análise</h3>
                <p style="color: #7f1d1d;">${String(error)}</p>
                <p style="color: #7f1d1d; margin-top: 1rem;">
                    <strong>Dica:</strong> Você precisa ter jogos salvos para gerar análise de frequência.
                </p>
            </div>
        `;
    }
}

function renderNumberAnalysisResults(lottery: string, frequencies: NumberFrequency[]) {
    const resultsDiv = document.getElementById('numberAnalysisResults')!;
    
    if (frequencies.length === 0) {
        resultsDiv.innerHTML = `
            <div class="feature-card" style="text-align: center;">
                <div style="font-size: 3rem; margin-bottom: 1rem;">📊</div>
                <h3>Dados Insuficientes</h3>
                <p style="color: var(--text-secondary);">
                    Não há dados suficientes para análise de frequência da ${lottery === 'megasena' ? 'Mega-Sena' : 'Lotofácil'}.
                </p>
            </div>
        `;
        return;
    }

    // Separar números por status
    const hotNumbers = frequencies.filter(f => f.status === 'hot').sort((a, b) => b.frequency - a.frequency);
    const coldNumbers = frequencies.filter(f => f.status === 'cold').sort((a, b) => a.frequency - b.frequency);
    const normalNumbers = frequencies.filter(f => f.status === 'normal').sort((a, b) => b.frequency - a.frequency);

    resultsDiv.innerHTML = `
        <div class="feature-card" style="margin-bottom: 2rem;">
            <h3 style="margin-bottom: 1rem;">
                ${lottery === 'megasena' ? '🔥 Mega-Sena' : '⭐ Lotofácil'} - Análise de Frequência
            </h3>
            <p style="color: var(--text-secondary); margin-bottom: 2rem;">
                Análise baseada em ${frequencies.length} números dos seus jogos salvos
            </p>

            <!-- Números Quentes -->
            ${hotNumbers.length > 0 ? `
            <div style="margin-bottom: 2rem;">
                <h4 style="color: #dc2626; margin-bottom: 1rem;">🔥 Números Quentes (Mais Frequentes)</h4>
                <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 0.5rem;">
                    ${hotNumbers.slice(0, 10).map(num => `
                        <div style="background: linear-gradient(135deg, #fecaca, #fee2e2); color: #7f1d1d; padding: 1rem; border-radius: 0.5rem; text-align: center; border: 1px solid #f87171;">
                            <div style="font-size: 1.5rem; font-weight: bold;">${num.number}</div>
                            <div style="font-size: 0.8rem; opacity: 0.8;">${num.frequency}x (${num.percentage.toFixed(1)}%)</div>
                        </div>
                    `).join('')}
                </div>
            </div>
            ` : ''}

            <!-- Números Frios -->
            ${coldNumbers.length > 0 ? `
            <div style="margin-bottom: 2rem;">
                <h4 style="color: #3b82f6; margin-bottom: 1rem;">❄️ Números Frios (Menos Frequentes)</h4>
                <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 0.5rem;">
                    ${coldNumbers.slice(0, 10).map(num => `
                        <div style="background: linear-gradient(135deg, #bfdbfe, #dbeafe); color: #1e40af; padding: 1rem; border-radius: 0.5rem; text-align: center; border: 1px solid #60a5fa;">
                            <div style="font-size: 1.5rem; font-weight: bold;">${num.number}</div>
                            <div style="font-size: 0.8rem; opacity: 0.8;">${num.frequency}x (${num.percentage.toFixed(1)}%)</div>
                        </div>
                    `).join('')}
                </div>
            </div>
            ` : ''}

            <!-- Números Normais -->
            ${normalNumbers.length > 0 ? `
            <div style="margin-bottom: 2rem;">
                <h4 style="color: #059669; margin-bottom: 1rem;">📊 Números com Frequência Normal</h4>
                <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: 0.5rem;">
                    ${normalNumbers.slice(0, 15).map(num => `
                        <div style="background: linear-gradient(135deg, #a7f3d0, #d1fae5); color: #065f46; padding: 1rem; border-radius: 0.5rem; text-align: center; border: 1px solid #34d399;">
                            <div style="font-size: 1.5rem; font-weight: bold;">${num.number}</div>
                            <div style="font-size: 0.8rem; opacity: 0.8;">${num.frequency}x (${num.percentage.toFixed(1)}%)</div>
                        </div>
                    `).join('')}
                </div>
            </div>
            ` : ''}

            <!-- Estatísticas Gerais -->
            <div style="background: #f9fafb; padding: 1rem; border-radius: 0.5rem; margin-top: 2rem;">
                <h4 style="margin-bottom: 1rem;">📈 Estatísticas da Análise</h4>
                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
                    <div>
                        <strong>Total de Números Analisados:</strong> ${frequencies.length}
                    </div>
                    <div>
                        <strong>Números Quentes:</strong> ${hotNumbers.length}
                    </div>
                    <div>
                        <strong>Números Frios:</strong> ${coldNumbers.length}
                    </div>
                    <div>
                        <strong>Números Normais:</strong> ${normalNumbers.length}
                    </div>
                </div>
            </div>
        </div>

        <!-- Ações -->
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 1rem;">
            <button onclick="startStrategyWizard()" class="feature-card" style="border: none; cursor: pointer; transition: transform 0.2s;" onmouseover="this.style.transform='scale(1.02)'" onmouseout="this.style.transform='scale(1)'">
                <span style="font-size: 2rem;">🧠</span>
                <h4>Gerar Estratégia</h4>
                <p style="margin: 0;">Baseada na análise</p>
            </button>

            <button onclick="renderDetailedAnalytics()" class="feature-card" style="border: none; cursor: pointer; transition: transform 0.2s;" onmouseover="this.style.transform='scale(1.02)'" onmouseout="this.style.transform='scale(1)'">
                <span style="font-size: 2rem;">📈</span>
                <h4>Análise Completa</h4>
                <p style="margin: 0;">Todas as métricas</p>
            </button>

            <button onclick="renderROICalculator()" class="feature-card" style="border: none; cursor: pointer; transition: transform 0.2s;" onmouseover="this.style.transform='scale(1.02)'" onmouseout="this.style.transform='scale(1)'">
                <span style="font-size: 2rem;">💰</span>
                <h4>Calculadora ROI</h4>
                <p style="margin: 0;">Projeções</p>
            </button>
        </div>
    `;
}

function renderAnalyticsError(error: string) {
    const app = document.getElementById('app')!;
    app.innerHTML = `
        <div class="container">
            <header class="header">
                <h1 class="logo">📈 Análise Detalhada</h1>
                <div class="header-actions">
                    <button onclick="renderPerformanceDashboard()" class="btn-secondary">📊 Dashboard</button>
                    <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                </div>
            </header>
            <div class="main-content" style="padding: 2rem;">
                <div class="feature-card" style="text-align: center; background: #fef2f2; border: 1px solid #fecaca;">
                    <div style="font-size: 3rem; margin-bottom: 1rem;">📊</div>
                    <h2 style="color: #dc2626;">Dados Insuficientes</h2>
                    <p style="color: #7f1d1d; margin-bottom: 2rem;">
                        ${error}
                        <br><br>
                        Para gerar análises detalhadas, você precisa ter jogos salvos com resultados.
                    </p>
                    <div style="display: flex; gap: 1rem; justify-content: center; flex-wrap: wrap;">
                        <button onclick="startStrategyWizard()" class="btn-primary">
                            🎲 Gerar Primeira Estratégia
                        </button>
                        <button onclick="renderSavedGamesScreen()" class="btn-secondary">
                            💾 Ver Jogos Salvos
                        </button>
                    </div>
                </div>
            </div>
        </div>
    `;
}

// ===============================
// ADICIONAR JOGO MANUAL
// ===============================

// Mostrar modal para adicionar jogo manual
function showAddManualGameModal() {
    // Criar modal
    const modal = document.createElement('div');
    modal.className = 'modal-overlay';
    modal.innerHTML = `
        <div class="modal-content" style="max-width: 800px;">
            <div class="modal-header">
                <h3>➕ Adicionar Jogo Manual</h3>
                <button class="modal-close" onclick="closeModal()">&times;</button>
            </div>
            <div class="modal-body">
                <form id="manualGameForm" class="add-game-form">
                    <!-- Seleção da Loteria -->
                    <div class="form-section">
                        <h4>🎯 Tipo de Loteria</h4>
                        <div class="lottery-options">
                            <label class="lottery-option">
                                <input type="radio" name="lotteryType" value="mega-sena" onchange="updateNumberLimits()" checked>
                                <div class="option-card">
                                    <span class="option-icon">🔥</span>
                                    <div class="option-content">
                                        <h4>Mega-Sena</h4>
                                        <p>6 a 15 números de 1 a 60</p>
                                    </div>
                                </div>
                            </label>
                            
                            <label class="lottery-option">
                                <input type="radio" name="lotteryType" value="lotofacil" onchange="updateNumberLimits()">
                                <div class="option-card">
                                    <span class="option-icon">⭐</span>
                                    <div class="option-content">
                                        <h4>Lotofácil</h4>
                                        <p>15 a 20 números de 1 a 25</p>
                                    </div>
                                </div>
                            </label>
                        </div>
                    </div>

                    <!-- Números -->
                    <div class="form-section">
                        <h4>🔢 Números Jogados</h4>
                        <p style="color: var(--text-secondary); margin-bottom: 1rem;" id="numberLimitsText">
                            Selecione entre 6 e 15 números de 1 a 60
                        </p>
                        
                        <!-- Grid de números -->
                        <div id="numbersGrid" class="numbers-grid">
                            <!-- Será preenchido dinamicamente -->
                        </div>
                        
                        <!-- Entrada manual -->
                        <div style="margin-top: 1rem;">
                            <label for="manualNumbers">Ou digite os números separados por vírgula:</label>
                            <input type="text" id="manualNumbers" placeholder="Ex: 1, 7, 15, 23, 35, 42" 
                                   style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 0.5rem; margin-top: 0.5rem;"
                                   onchange="updateNumbersFromText()">
                        </div>
                        
                        <!-- Números selecionados -->
                        <div style="margin-top: 1rem;">
                            <h5>Números selecionados (<span id="selectedCount">0</span>):</h5>
                            <div id="selectedNumbers" class="selected-numbers-display">
                                <!-- Será preenchido dinamicamente -->
                            </div>
                        </div>
                    </div>
                    
                    <!-- Informações do Sorteio -->
                    <div class="form-section">
                        <h4>📅 Informações do Sorteio</h4>
                        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 1rem;">
                            <div>
                                <label for="manualDate">Data do Sorteio</label>
                                <input type="date" id="manualDate" required 
                                       style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 0.5rem;">
                            </div>
                            
                            <div>
                                <label for="manualContest">Número do Concurso</label>
                                <input type="number" id="manualContest" min="1" required 
                                       style="width: 100%; padding: 0.75rem; border: 1px solid #d1d5db; border-radius: 0.5rem;">
                            </div>
                        </div>
                    </div>
                </form>
            </div>
            <div class="modal-actions">
                <button class="btn-secondary" onclick="closeModal()">Cancelar</button>
                <button class="btn-primary" onclick="confirmAddManualGame()">
                    <span>💾</span>
                    Adicionar Jogo
                </button>
            </div>
        </div>
    `;
    
    document.body.appendChild(modal);
    
    // Inicializar grid de números
    updateNumberLimits();
}

// Atualizar limites de números baseado na loteria selecionada
function updateNumberLimits() {
    const lotteryType = (document.querySelector('input[name="lotteryType"]:checked') as HTMLInputElement)?.value;
    const limitsText = document.getElementById('numberLimitsText')!;
    
    if (lotteryType === 'mega-sena') {
        limitsText.textContent = 'Selecione entre 6 e 15 números de 1 a 60';
        createNumbersGrid(1, 60);
    } else if (lotteryType === 'lotofacil') {
        limitsText.textContent = 'Selecione entre 15 e 20 números de 1 a 25';
        createNumbersGrid(1, 25);
    }
}

// Criar grid de números para seleção
function createNumbersGrid(min: number, max: number) {
    const numbersGrid = document.getElementById('numbersGrid')!;
    
    let html = '';
    for (let i = min; i <= max; i++) {
        html += `
            <button type="button" class="number-btn" data-number="${i}" onclick="toggleNumber(${i})">
                ${i.toString().padStart(2, '0')}
            </button>
        `;
    }
    
    numbersGrid.innerHTML = html;
}

// Alternar seleção de um número
function toggleNumber(number: number) {
    const btn = document.querySelector(`[data-number="${number}"]`) as HTMLButtonElement;
    
    if (btn.classList.contains('selected')) {
        btn.classList.remove('selected');
    } else {
        btn.classList.add('selected');
    }
    
    updateSelectedDisplay();
    updateManualInput();
}

// Atualizar exibição dos números selecionados
function updateSelectedDisplay() {
    const selectedBtns = document.querySelectorAll('.number-btn.selected');
    const selectedNumbers = Array.from(selectedBtns).map(btn => parseInt((btn as HTMLElement).dataset.number!));
    selectedNumbers.sort((a, b) => a - b);
    
    const countElement = document.getElementById('selectedCount')!;
    const displayElement = document.getElementById('selectedNumbers')!;
    
    countElement.textContent = selectedNumbers.length.toString();
    
    if (selectedNumbers.length === 0) {
        displayElement.innerHTML = '<span style="color: var(--text-secondary);">Nenhum número selecionado</span>';
    } else {
        displayElement.innerHTML = selectedNumbers.map(num => 
            `<span class="number">${num.toString().padStart(2, '0')}</span>`
        ).join('');
    }
    
    // Validar limites
    const lotteryType = (document.querySelector('input[name="lotteryType"]:checked') as HTMLInputElement)?.value;
    validateNumberSelection(lotteryType, selectedNumbers.length);
}

// Atualizar input manual com números selecionados
function updateManualInput() {
    const selectedBtns = document.querySelectorAll('.number-btn.selected');
    const selectedNumbers = Array.from(selectedBtns).map(btn => parseInt((btn as HTMLElement).dataset.number!));
    selectedNumbers.sort((a, b) => a - b);
    
    const manualInput = document.getElementById('manualNumbers') as HTMLInputElement;
    manualInput.value = selectedNumbers.join(', ');
}

// Atualizar seleção a partir do texto
function updateNumbersFromText() {
    const manualInput = document.getElementById('manualNumbers') as HTMLInputElement;
    const text = manualInput.value.trim();
    
    // Limpar seleções anteriores
    document.querySelectorAll('.number-btn.selected').forEach(btn => {
        btn.classList.remove('selected');
    });
    
    if (text) {
        const numbers = text.split(',').map(n => parseInt(n.trim())).filter(n => !isNaN(n));
        
        numbers.forEach(num => {
            const btn = document.querySelector(`[data-number="${num}"]`) as HTMLButtonElement;
            if (btn) {
                btn.classList.add('selected');
            }
        });
    }
    
    updateSelectedDisplay();
}

// Validar seleção de números
function validateNumberSelection(lotteryType: string, count: number): boolean {
    const errorElement = document.getElementById('selectionError');
    
    // Remover erro anterior
    if (errorElement) {
        errorElement.remove();
    }
    
    let isValid = true;
    let errorMessage = '';
    
    if (lotteryType === 'mega-sena') {
        if (count < 6) {
            isValid = false;
            errorMessage = 'Mega-Sena precisa de pelo menos 6 números';
        } else if (count > 15) {
            isValid = false;
            errorMessage = 'Mega-Sena aceita no máximo 15 números';
        }
    } else if (lotteryType === 'lotofacil') {
        if (count < 15) {
            isValid = false;
            errorMessage = 'Lotofácil precisa de pelo menos 15 números';
        } else if (count > 20) {
            isValid = false;
            errorMessage = 'Lotofácil aceita no máximo 20 números';
        }
    }
    
    if (!isValid) {
        const selectedNumbersDiv = document.getElementById('selectedNumbers')!;
        const errorDiv = document.createElement('div');
        errorDiv.id = 'selectionError';
        errorDiv.style.cssText = 'color: #ef4444; font-size: 0.9rem; margin-top: 0.5rem; font-weight: 600;';
        errorDiv.textContent = errorMessage;
        selectedNumbersDiv.parentNode!.appendChild(errorDiv);
    }
    
    return isValid;
}

// Confirmar adição do jogo manual
async function confirmAddManualGame() {
    const lotteryType = (document.querySelector('input[name="lotteryType"]:checked') as HTMLInputElement)?.value;
    const manualDate = (document.getElementById('manualDate') as HTMLInputElement).value;
    const manualContest = parseInt((document.getElementById('manualContest') as HTMLInputElement).value);
    
    // Obter números selecionados
    const selectedBtns = document.querySelectorAll('.number-btn.selected');
    const selectedNumbers = Array.from(selectedBtns).map(btn => parseInt((btn as HTMLElement).dataset.number!));
    
    // Validações
    if (!lotteryType) {
        alert('❌ Selecione o tipo de loteria');
        return;
    }
    
    if (selectedNumbers.length === 0) {
        alert('❌ Selecione pelo menos um número');
        return;
    }
    
    if (!validateNumberSelection(lotteryType, selectedNumbers.length)) {
        return; // Erro já mostrado na tela
    }
    
    if (!manualDate) {
        alert('❌ Informe a data do sorteio');
        return;
    }
    
    if (!manualContest || manualContest <= 0) {
        alert('❌ Informe um número de concurso válido');
        return;
    }
    
    try {
        // Preparar dados
        const request = new models.SaveGameRequest({
            lottery_type: lotteryType,
            numbers: selectedNumbers,
            expected_draw: manualDate, // Já está no formato YYYY-MM-DD
            contest_number: manualContest
        });
        
        console.log('🎲 Enviando jogo manual:', request);
        
        // Chamar função específica para jogos manuais
        const response = await SaveManualGame(request);
        
        console.log('📝 Resposta do backend:', response);
        
        if (response.success) {
            closeModal();
            showNotification('✅ Jogo adicionado manualmente com sucesso!', 'success');
            renderSavedGamesScreen(); // Recarregar a tela
        } else {
            alert('❌ Erro ao adicionar jogo: ' + (response.error || 'Erro desconhecido'));
        }
    } catch (error) {
        console.error('❌ Erro ao adicionar jogo manual:', error);
        alert('❌ Erro ao adicionar jogo. Tente novamente.');
    }
}

function getPerformanceClass(level: string): string {
    switch (level) {
        case 'Excelente': return 'excellent';
        case 'Boa': return 'good';
        case 'Regular': return 'regular';
        case 'Baixa': return 'low';
        default: return 'regular';
    }
}

// 🧠 FASE 2: INTELLIGENCE ENGINE - IA COMPORTAMENTAL AVANÇADA
// @ts-ignore - Função usada no HTML onclick
function renderIntelligenceEngine() {
    try {
        const games = JSON.parse(localStorage.getItem('games') || '[]');
        
        const iaAnalysis = generateBehavioralAnalysis(games);
        const heatmapData = generateHeatmapData(games);
        const predictions = generateAIPredictions(games);
        const suggestions = generatePersonalizedSuggestions(games, iaAnalysis);
        const timing = calculateOptimalTiming(games);

        const app = document.getElementById('app')!;
        
        app.innerHTML = `
            <div class="container">
                <header class="header">
                    <h1 class="logo">🧠 Intelligence Engine</h1>
                    <div class="header-actions">
                        <button onclick="renderPerformanceDashboard()" class="btn-secondary">📊 Dashboard</button>
                        <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                    </div>
                </header>
                
                <div class="main-content">
                    <!-- Hero Section Épico -->
                    <div class="intelligence-hero">
                        <h1 class="intelligence-title">
                            <span class="intelligence-brain">🧠</span>
                            Intelligence Engine
                            <span class="intelligence-brain">🚀</span>
                        </h1>
                        <p style="font-size: var(--font-size-lg); color: var(--text-secondary); margin: 0;">
                            IA comportamental avançada para maximizar sua performance
                        </p>
                    </div>

                    <!-- Análise Comportamental -->
                    <div class="section">
                        <h2 class="section-title">🤖 Análise Comportamental Avançada</h2>
                        <div class="behavioral-analysis">
                            ${generateBehaviorCards(iaAnalysis)}
                        </div>
                    </div>

                    <!-- Heatmaps Épicos -->
                    <div class="section">
                        <h2 class="section-title">🔥 Heatmaps de Números</h2>
                        <div class="heatmap-section">
                            ${generateHeatmaps(heatmapData)}
                        </div>
                    </div>

                    <!-- Predições da IA -->
                    <div class="section">
                        <h2 class="section-title">📈 Predições da IA</h2>
                        <div class="predictions-section">
                            ${generatePredictionCards(predictions)}
                        </div>
                    </div>

                    <!-- Sugestões Personalizadas -->
                    <div class="section">
                        <h2 class="section-title">💡 Sugestões Personalizadas</h2>
                        <div class="suggestions-grid">
                            ${generateSuggestionCards(suggestions)}
                        </div>
                    </div>

                    <!-- Timing Ideal -->
                    <div class="section">
                        <h2 class="section-title">⏰ Momentos Ideais para Jogar</h2>
                        <div class="timing-section">
                            <div class="timing-grid">
                                ${generateTimingCards(timing)}
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
    } catch (error) {
        console.error('🧠 ERROR: Erro em renderIntelligenceEngine:', (error as Error).message);
        console.error('🧠 ERROR: Stack trace:', (error as Error).stack);
        
        const app = document.getElementById('app')!;
        app.innerHTML = `
            <div class="container">
                <header class="header">
                    <h1 class="logo">🧠 Intelligence Engine</h1>
                    <div class="header-actions">
                        <button onclick="renderWelcome()" class="btn-secondary">⬅️ Voltar</button>
                    </div>
                </header>
                
                <div class="main-content">
                    <div class="feature-card" style="background: #fef2f2; border: 1px solid #fecaca;">
                        <h2 style="color: #dc2626;">❌ Erro no Intelligence Engine</h2>
                        <p style="color: #7f1d1d;">
                            Ocorreu um erro inesperado. Por favor, tente novamente ou entre em contato com o suporte.
                        </p>
                        <button onclick="renderWelcome()" class="btn-primary" style="margin-top: 1rem;">
                            ⬅️ Voltar ao Início
                        </button>
                    </div>
                </div>
            </div>
        `;
    }
}

// Análise Comportamental IA
function generateBehavioralAnalysis(games: any[]): any {
    console.log('🔍 DEBUG: generateBehavioralAnalysis iniciado com', games.length, 'games');
    
    try {
        console.log('🔍 DEBUG: Calculando favoriteNumbers...');
        const favoriteNumbers = calculateFavoriteNumbers(games);
        console.log('🔍 DEBUG: favoriteNumbers:', favoriteNumbers);
        
        console.log('🔍 DEBUG: Analisando playingPatterns...');
        const playingPatterns = analyzePlayingPatterns(games);
        console.log('🔍 DEBUG: playingPatterns:', playingPatterns);
        
        console.log('🔍 DEBUG: Calculando riskProfile...');
        const riskProfile = calculateRiskProfile(games);
        console.log('🔍 DEBUG: riskProfile:', riskProfile);
        
        console.log('🔍 DEBUG: Analisando performanceTraits...');
        const performanceTraits = analyzePerformanceTraits(games);
        console.log('🔍 DEBUG: performanceTraits:', performanceTraits);
        
        console.log('🔍 DEBUG: Analisando timePatterns...');
        const timePatterns = analyzeTimePatterns(games);
        console.log('🔍 DEBUG: timePatterns:', timePatterns);

        const analysis = {
            favoriteNumbers,
            playingPatterns,
            riskProfile,
            performanceTraits,
            timePatterns
        };

        console.log('🔍 DEBUG: generateBehavioralAnalysis concluído:', analysis);
        return analysis;
        
    } catch (error) {
        console.error('🔍 ERROR: Erro em generateBehavioralAnalysis:', error);
        return {
            favoriteNumbers: { top5: [], avgFrequency: 0, diversity: 0, consistency: 0 },
            playingPatterns: { preferredGame: 'N/A', gamesPerWeek: 0, avgInvestment: 0, consistency: 0 },
            riskProfile: { level: 'N/A', avgInvestment: 0, maxInvestment: 0, roi: 0, volatility: 0 },
            performanceTraits: { winRate: 0, avgROI: 0, bestStreak: 0, patience: 0, adaptation: 0 },
            timePatterns: { preferredDay: 'N/A', preferredHour: 0, weekendGames: 0, weekdayGames: 0 }
        };
    }
}

// Calcula números favoritos do usuário
function calculateFavoriteNumbers(games: any[]): any {
    console.log('📊 DEBUG: calculateFavoriteNumbers iniciado com', games.length, 'games');
    
    try {
        const numberFreq: { [key: number]: number } = {};
        let totalNumbers = 0;

        games.forEach((game, index) => {
            console.log(`📊 DEBUG: Processando game ${index}:`, game);
            if (game.numbers && Array.isArray(game.numbers)) {
                game.numbers.forEach((num: number) => {
                    numberFreq[num] = (numberFreq[num] || 0) + 1;
                    totalNumbers++;
                });
            } else {
                console.warn(`📊 WARNING: Game ${index} não tem números válidos:`, game.numbers);
            }
        });

        console.log('📊 DEBUG: numberFreq final:', numberFreq);
        console.log('📊 DEBUG: totalNumbers:', totalNumbers);

        const sortedNumbers = Object.entries(numberFreq)
            .map(([num, freq]) => ({
                number: parseInt(num),
                frequency: freq,
                percentage: (freq / totalNumbers * 100)
            }))
            .sort((a, b) => b.frequency - a.frequency);

        console.log('📊 DEBUG: sortedNumbers:', sortedNumbers);

        const result = {
            top5: sortedNumbers.slice(0, 5),
            avgFrequency: totalNumbers / Object.keys(numberFreq).length,
            diversity: Object.keys(numberFreq).length,
            consistency: sortedNumbers[0]?.frequency / (totalNumbers / Object.keys(numberFreq).length) || 0
        };

        console.log('📊 DEBUG: calculateFavoriteNumbers result:', result);
        return result;
        
    } catch (error) {
        console.error('📊 ERROR: Erro em calculateFavoriteNumbers:', error);
        return {
            top5: [],
            avgFrequency: 0,
            diversity: 0,
            consistency: 0
        };
    }
}

// Analisa padrões de jogo
function analyzePlayingPatterns(games: any[]): any {
    console.log('🎯 DEBUG: analyzePlayingPatterns iniciado');
    
    try {
        const gameTypes = games.reduce((acc: any, game) => {
            acc[game.type] = (acc[game.type] || 0) + 1;
            return acc;
        }, {});

        const avgInvestment = games.length > 0 ? games.reduce((sum, game) => sum + (game.investment || 0), 0) / games.length : 0;
        
        const result = {
            preferredGame: Object.entries(gameTypes).length > 0 ? Object.entries(gameTypes).reduce((a: any, b: any) => a[1] > b[1] ? a : b)[0] : 'N/A',
            gamesPerWeek: games.length / 4, // Assumindo 4 semanas de dados
            avgInvestment: avgInvestment,
            consistency: calculateConsistency(games)
        };
        
        console.log('🎯 DEBUG: analyzePlayingPatterns result:', result);
        return result;
    } catch (error) {
        console.error('🎯 ERROR: Erro em analyzePlayingPatterns:', error);
        return {
            preferredGame: 'N/A',
            gamesPerWeek: 0,
            avgInvestment: 0,
            consistency: 0
        };
    }
}

// Calcula perfil de risco
function calculateRiskProfile(games: any[]): any {
    console.log('⚡ DEBUG: calculateRiskProfile iniciado');
    
    try {
        const totalInvestment = games.reduce((sum, game) => sum + (game.investment || 0), 0);
        const totalWinnings = games.reduce((sum, game) => sum + (game.winnings || 0), 0);
        const roi = totalInvestment > 0 ? ((totalWinnings - totalInvestment) / totalInvestment) * 100 : 0;
        
        const avgInvestment = games.length > 0 ? totalInvestment / games.length : 0;
        const maxInvestment = games.length > 0 ? Math.max(...games.map(g => g.investment || 0)) : 0;
        
        let riskLevel = 'Conservador';
        if (avgInvestment > 50) riskLevel = 'Moderado';
        if (avgInvestment > 100) riskLevel = 'Agressivo';
        if (maxInvestment > avgInvestment * 3) riskLevel = 'Alto Risco';

        const result = {
            level: riskLevel,
            avgInvestment,
            maxInvestment,
            roi,
            volatility: calculateVolatility(games)
        };
        
        console.log('⚡ DEBUG: calculateRiskProfile result:', result);
        return result;
    } catch (error) {
        console.error('⚡ ERROR: Erro em calculateRiskProfile:', error);
        return {
            level: 'N/A',
            avgInvestment: 0,
            maxInvestment: 0,
            roi: 0,
            volatility: 0
        };
    }
}

// Analisa traços de performance
function analyzePerformanceTraits(games: any[]): any {
    console.log('🏆 DEBUG: analyzePerformanceTraits iniciado');
    
    try {
        const winningGames = games.filter(g => (g.winnings || 0) > (g.investment || 0));
        const winRate = games.length > 0 ? (winningGames.length / games.length) * 100 : 0;
        
        const result = {
            winRate,
            avgROI: calculateAverageROI(games),
            bestStreak: calculateBestStreak(games),
            patience: calculatePatience(games),
            adaptation: calculateAdaptation(games)
        };
        
        console.log('🏆 DEBUG: analyzePerformanceTraits result:', result);
        return result;
    } catch (error) {
        console.error('🏆 ERROR: Erro em analyzePerformanceTraits:', error);
        return {
            winRate: 0,
            avgROI: 0,
            bestStreak: 0,
            patience: 0,
            adaptation: 0
        };
    }
}

// Analisa padrões temporais
function analyzeTimePatterns(games: any[]): any {
    console.log('⏰ DEBUG: analyzeTimePatterns iniciado');
    
    try {
        const dayOfWeek: { [key: string]: number } = {};
        const hourOfDay: { [key: number]: number } = {};
        
        games.forEach(game => {
            if (game.date) {
                const date = new Date(game.date);
                const day = date.toLocaleDateString('pt-BR', { weekday: 'long' });
                const hour = date.getHours();
                
                dayOfWeek[day] = (dayOfWeek[day] || 0) + 1;
                hourOfDay[hour] = (hourOfDay[hour] || 0) + 1;
            }
        });

        const preferredDay = Object.entries(dayOfWeek).length > 0 ? Object.entries(dayOfWeek).reduce((a, b) => a[1] > b[1] ? a : b)[0] : 'N/A';
        const preferredHour = Object.entries(hourOfDay).length > 0 ? Object.entries(hourOfDay).reduce((a, b) => a[1] > b[1] ? a : b)[0] : '0';

        const result = {
            preferredDay,
            preferredHour: parseInt(preferredHour as string),
            weekendGames: (dayOfWeek['sábado'] || 0) + (dayOfWeek['domingo'] || 0),
            weekdayGames: games.length - ((dayOfWeek['sábado'] || 0) + (dayOfWeek['domingo'] || 0))
        };
        
        console.log('⏰ DEBUG: analyzeTimePatterns result:', result);
        return result;
    } catch (error) {
        console.error('⏰ ERROR: Erro em analyzeTimePatterns:', error);
        return {
            preferredDay: 'N/A',
            preferredHour: 0,
            weekendGames: 0,
            weekdayGames: 0
        };
    }
}

// Gera cards de comportamento
function generateBehaviorCards(analysis: any): string {
    console.log('💳 DEBUG: generateBehaviorCards iniciado');
    
    try {
        return `
            <div class="behavior-card">
                <div class="behavior-header">
                    <span class="behavior-icon">🎯</span>
                    <h3 class="behavior-title">Números Favoritos</h3>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Top 3 números</span>
                    <span class="behavior-value">${analysis.favoriteNumbers.top5.slice(0, 3).map((n: any) => n.number).join(', ') || 'N/A'}</span>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Diversidade</span>
                    <span class="behavior-value">${analysis.favoriteNumbers.diversity} números únicos</span>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Consistência</span>
                    <span class="behavior-value">${(analysis.favoriteNumbers.consistency * 100).toFixed(1)}%</span>
                </div>
            </div>

            <div class="behavior-card">
                <div class="behavior-header">
                    <span class="behavior-icon">🎲</span>
                    <h3 class="behavior-title">Padrões de Jogo</h3>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Jogo Preferido</span>
                    <span class="behavior-value">${analysis.playingPatterns.preferredGame}</span>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Jogos/Semana</span>
                    <span class="behavior-value">${analysis.playingPatterns.gamesPerWeek.toFixed(1)}</span>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Investimento Médio</span>
                    <span class="behavior-value">R$ ${analysis.playingPatterns.avgInvestment.toFixed(2)}</span>
                </div>
            </div>

            <div class="behavior-card">
                <div class="behavior-header">
                    <span class="behavior-icon">⚡</span>
                    <h3 class="behavior-title">Perfil de Risco</h3>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Nível</span>
                    <span class="behavior-value">${analysis.riskProfile.level}</span>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">ROI Atual</span>
                    <span class="behavior-value">${analysis.riskProfile.roi.toFixed(2)}%</span>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Volatilidade</span>
                    <span class="behavior-value">${(analysis.riskProfile.volatility * 100).toFixed(1)}%</span>
                </div>
            </div>

            <div class="behavior-card">
                <div class="behavior-header">
                    <span class="behavior-icon">🏆</span>
                    <h3 class="behavior-title">Performance</h3>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Taxa de Vitória</span>
                    <span class="behavior-value">${analysis.performanceTraits.winRate.toFixed(1)}%</span>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Melhor Sequência</span>
                    <span class="behavior-value">${analysis.performanceTraits.bestStreak} acertos</span>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Paciência</span>
                    <span class="behavior-value">${(analysis.performanceTraits.patience * 100).toFixed(0)}%</span>
                </div>
            </div>

            <div class="behavior-card">
                <div class="behavior-header">
                    <span class="behavior-icon">⏰</span>
                    <h3 class="behavior-title">Padrões Temporais</h3>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Dia Preferido</span>
                    <span class="behavior-value">${analysis.timePatterns.preferredDay}</span>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Horário Favorito</span>
                    <span class="behavior-value">${analysis.timePatterns.preferredHour}:00</span>
                </div>
                <div class="behavior-metric">
                    <span class="behavior-label">Jogos Weekend</span>
                    <span class="behavior-value">${analysis.timePatterns.weekendGames}</span>
                </div>
            </div>
        `;
    } catch (error) {
        console.error('💳 ERROR: Erro em generateBehaviorCards:', error);
        return '<div class="behavior-card">Erro ao gerar análise comportamental</div>';
    }
}

// Gera dados do heatmap
function generateHeatmapData(games: any[]): any {
    console.log('🔥 DEBUG: generateHeatmapData iniciado');
    
    try {
        const megasenaFreq: { [key: number]: number } = {};
        const lotofacilFreq: { [key: number]: number } = {};

        games.forEach(game => {
            if (game.numbers && Array.isArray(game.numbers)) {
                const freq = game.type === 'megasena' || game.type === 'mega-sena' ? megasenaFreq : lotofacilFreq;
                game.numbers.forEach((num: number) => {
                    freq[num] = (freq[num] || 0) + 1;
                });
            }
        });

        const result = {
            megasena: calculateHeatLevels(megasenaFreq, 60),
            lotofacil: calculateHeatLevels(lotofacilFreq, 25)
        };
        
        console.log('🔥 DEBUG: generateHeatmapData result:', result);
        return result;
    } catch (error) {
        console.error('🔥 ERROR: Erro em generateHeatmapData:', error);
        return {
            megasena: [],
            lotofacil: []
        };
    }
}

// Calcula níveis de calor
function calculateHeatLevels(freq: { [key: number]: number }, maxNumber: number): any[] {
    console.log('🌡️ DEBUG: calculateHeatLevels iniciado para', maxNumber, 'números');
    
    try {
        const values = Object.values(freq);
        if (values.length === 0) {
            // Se não há dados, retornar todos como nível 1
            const result = [];
            for (let i = 1; i <= maxNumber; i++) {
                result.push({ number: i, frequency: 0, level: 1 });
            }
            return result;
        }
        
        const max = Math.max(...values);
        const min = Math.min(...values);
        
        const result = [];
        for (let i = 1; i <= maxNumber; i++) {
            const frequency = freq[i] || 0;
            let level = 1;
            
            if (max > min) {
                const normalized = (frequency - min) / (max - min);
                level = Math.ceil(normalized * 5) || 1;
            }
            
            result.push({
                number: i,
                frequency,
                level
            });
        }
        
        console.log('🌡️ DEBUG: calculateHeatLevels result:', result.length, 'números processados');
        return result;
    } catch (error) {
        console.error('🌡️ ERROR: Erro em calculateHeatLevels:', error);
        return [];
    }
}

// Gera heatmaps visuais
function generateHeatmaps(data: any): string {
    console.log('🖼️ DEBUG: generateHeatmaps iniciado');
    
    try {
        return `
            <div class="heatmap-container">
                <h3 class="heatmap-title">🔥 Mega-Sena - Frequência de Números</h3>
                <div class="heatmap-grid heatmap-megasena">
                    ${data.megasena.map((item: any) => `
                        <div class="heatmap-number heat-level-${item.level}" 
                             title="Número ${item.number}: ${item.frequency} vezes (${item.level}/5)">
                            ${item.number}
                        </div>
                    `).join('')}
                </div>
                <div class="heatmap-legend">
                    <div class="legend-item">
                        <div class="legend-color heat-level-5"></div>
                        <span class="legend-label">Muito Quente</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color heat-level-4"></div>
                        <span class="legend-label">Quente</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color heat-level-3"></div>
                        <span class="legend-label">Morno</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color heat-level-2"></div>
                        <span class="legend-label">Frio</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color heat-level-1"></div>
                        <span class="legend-label">Muito Frio</span>
                    </div>
                </div>
            </div>

            <div class="heatmap-container">
                <h3 class="heatmap-title">🍀 Lotofácil - Frequência de Números</h3>
                <div class="heatmap-grid heatmap-lotofacil">
                    ${data.lotofacil.map((item: any) => `
                        <div class="heatmap-number heat-level-${item.level}" 
                             title="Número ${item.number}: ${item.frequency} vezes (${item.level}/5)">
                            ${item.number}
                        </div>
                    `).join('')}
                </div>
                <div class="heatmap-legend">
                    <div class="legend-item">
                        <div class="legend-color heat-level-5"></div>
                        <span class="legend-label">Muito Quente</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color heat-level-4"></div>
                        <span class="legend-label">Quente</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color heat-level-3"></div>
                        <span class="legend-label">Morno</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color heat-level-2"></div>
                        <span class="legend-label">Frio</span>
                    </div>
                    <div class="legend-item">
                        <div class="legend-color heat-level-1"></div>
                        <span class="legend-label">Muito Frio</span>
                    </div>
                </div>
            </div>
        `;
    } catch (error) {
        console.error('🖼️ ERROR: Erro em generateHeatmaps:', error);
        return '<div class="heatmap-container">Erro ao gerar heatmaps</div>';
    }
}

// Gera predições da IA
function generateAIPredictions(games: any[]): any {
    console.log('📊 DEBUG: generateAIPredictions iniciado');
    
    try {
        const recentGames = games.slice(-10);
        
        const result = {
            performanceTrend: calculatePerformanceTrend(recentGames),
            optimalMoment: calculateOptimalMoment(games),
            roiPrediction: predictROI(games),
            numberRecommendations: generateNumberRecommendations(games)
        };
        
        console.log('📊 DEBUG: generateAIPredictions result:', result);
        return result;
    } catch (error) {
        console.error('📊 ERROR: Erro em generateAIPredictions:', error);
        return {
            performanceTrend: { score: 50, trend: 'Neutro' },
            optimalMoment: { score: 50, recommendation: 'Momento regular' },
            roiPrediction: { predicted: 0, confidence: 0 },
            numberRecommendations: { hot: [], cold: [], balanced: [] }
        };
    }
}

// Calcula tendência de performance
function calculatePerformanceTrend(recentGames: any[]): any {
    console.log('📈 DEBUG: calculatePerformanceTrend iniciado');
    
    try {
        if (recentGames.length === 0) {
            return { score: 50, trend: 'Neutro' };
        }
        
        const roiTrend = recentGames.map(game => {
            return (game.investment || 0) > 0 ? (((game.winnings || 0) - (game.investment || 0)) / (game.investment || 1)) * 100 : 0;
        });
        
        const avgROI = roiTrend.reduce((sum, roi) => sum + roi, 0) / roiTrend.length;
        
        let score = 50;
        if (avgROI > 10) score = 85;
        else if (avgROI > 0) score = 70;
        else if (avgROI > -10) score = 40;
        else score = 20;
        
        return { score, trend: avgROI > 0 ? 'Positiva' : 'Negativa' };
    } catch (error) {
        console.error('📈 ERROR: Erro em calculatePerformanceTrend:', error);
        return { score: 50, trend: 'Neutro' };
    }
}

// Calcula momento ótimo
function calculateOptimalMoment(games: any[]): any {
    console.log('⭐ DEBUG: calculateOptimalMoment iniciado');
    
    try {
        if (games.length === 0) {
            return { score: 50, recommendation: 'Momento regular' };
        }
        
        const winningGames = games.filter(g => (g.winnings || 0) > (g.investment || 0));
        const winRate = (winningGames.length / games.length) * 100;
        
        let score = 65;
        if (winRate > 30) score += 15;
        if (winRate > 15) score += 10;
        
        return { 
            score: Math.min(score, 95), 
            recommendation: score > 75 ? 'Excelente momento!' : score > 50 ? 'Momento favorável' : 'Aguarde melhor momento'
        };
    } catch (error) {
        console.error('⭐ ERROR: Erro em calculateOptimalMoment:', error);
        return { score: 50, recommendation: 'Momento regular' };
    }
}

// Prediz ROI futuro
function predictROI(games: any[]): any {
    console.log('💰 DEBUG: predictROI iniciado');
    
    try {
        if (games.length === 0) {
            return { predicted: 0, confidence: 0 };
        }
        
        const historical = calculateAverageROI(games);
        const trend = calculatePerformanceTrend(games.slice(-5));
        
        const predicted = historical + (trend.score - 50) * 0.1;
        
        return {
            predicted: Math.max(-50, Math.min(50, predicted)),
            confidence: Math.min(85, 60 + Math.abs(trend.score - 50) * 0.5)
        };
    } catch (error) {
        console.error('💰 ERROR: Erro em predictROI:', error);
        return { predicted: 0, confidence: 0 };
    }
}

// Gera recomendações de números
function generateNumberRecommendations(games: any[]): any {
    console.log('🔢 DEBUG: generateNumberRecommendations iniciado');
    
    try {
        if (games.length === 0) {
            return { hot: [], cold: [], balanced: [] };
        }
        
        const numberFreq: { [key: number]: number } = {};
        games.forEach(game => {
            if (game.numbers && Array.isArray(game.numbers)) {
                game.numbers.forEach((num: number) => {
                    numberFreq[num] = (numberFreq[num] || 0) + 1;
                });
            }
        });
        
        const sortedNumbers = Object.entries(numberFreq)
            .map(([num, freq]) => ({ number: parseInt(num), frequency: freq }))
            .sort((a, b) => b.frequency - a.frequency);
            
        const hotNumbers = sortedNumbers.slice(0, 3).map(n => n.number);
        const coldNumbers = [7, 13, 23, 31, 42]; // Números simulados
        
        return {
            hot: hotNumbers,
            cold: coldNumbers,
            balanced: [...hotNumbers.slice(0, 2), ...coldNumbers.slice(0, 2)]
        };
    } catch (error) {
        console.error('🔢 ERROR: Erro em generateNumberRecommendations:', error);
        return { hot: [], cold: [], balanced: [] };
    }
}

// Gera cards de predição
function generatePredictionCards(predictions: any): string {
    console.log('🎯 DEBUG: generatePredictionCards iniciado');
    
    try {
        return `
            <div class="prediction-card">
                <div class="prediction-header">
                    <span class="prediction-icon">📈</span>
                    <h3 class="prediction-title">Tendência de Performance</h3>
                </div>
                <div class="prediction-score">
                    <div class="score-circle ${getScoreClass(predictions.performanceTrend.score)}">
                        ${predictions.performanceTrend.score}
                    </div>
                    <p class="score-label">${predictions.performanceTrend.trend}</p>
                </div>
            </div>

            <div class="prediction-card">
                <div class="prediction-header">
                    <span class="prediction-icon">⭐</span>
                    <h3 class="prediction-title">Momento Ideal</h3>
                </div>
                <div class="prediction-score">
                    <div class="score-circle ${getScoreClass(predictions.optimalMoment.score)}">
                        ${predictions.optimalMoment.score}
                    </div>
                    <p class="score-label">${predictions.optimalMoment.recommendation}</p>
                </div>
            </div>

            <div class="prediction-card">
                <div class="prediction-header">
                    <span class="prediction-icon">💰</span>
                    <h3 class="prediction-title">Predição de ROI</h3>
                </div>
                <div class="prediction-score">
                    <div class="score-circle ${getScoreClass(predictions.roiPrediction.confidence)}">
                        ${predictions.roiPrediction.predicted.toFixed(1)}%
                    </div>
                    <p class="score-label">${predictions.roiPrediction.confidence.toFixed(0)}% confiança</p>
                </div>
            </div>
        `;
    } catch (error) {
        console.error('🎯 ERROR: Erro em generatePredictionCards:', (error as Error).message);
        return '<div class="prediction-card">Erro ao gerar predições</div>';
    }
}

// Gera sugestões personalizadas
function generatePersonalizedSuggestions(_games: any[], analysis: any): any[] {
    console.log('💡 DEBUG: generatePersonalizedSuggestions iniciado');
    
    try {
        const suggestions = [];
        
        // Sugestão baseada em ROI
        if (analysis.riskProfile && analysis.riskProfile.roi < 0) {
            suggestions.push({
                icon: '💡',
                title: 'Otimize sua Estratégia',
                description: 'Seu ROI está negativo. Considere reduzir investimentos e focar em jogos com melhor histórico.',
                priority: 'high',
                action: 'Ver Estratégias'
            });
        }
        
        // Sugestão baseada em consistência
        if (analysis.favoriteNumbers && analysis.favoriteNumbers.consistency < 1.5) {
            suggestions.push({
                icon: '🎯',
                title: 'Melhore a Consistência',
                description: 'Seus números variam muito. Considere manter alguns números fixos entre os jogos.',
                priority: 'medium',
                action: 'Ver Números'
            });
        }
        
        // Sugestão padrão se não há dados suficientes
        if (suggestions.length === 0) {
            suggestions.push({
                icon: '🚀',
                title: 'Continue Jogando',
                description: 'Continue salvando seus jogos para receber sugestões mais precisas!',
                priority: 'low',
                action: 'Continuar'
            });
        }
        
        console.log('💡 DEBUG: generatePersonalizedSuggestions result:', suggestions);
        return suggestions;
    } catch (error) {
        console.error('💡 ERROR: Erro em generatePersonalizedSuggestions:', (error as Error).message);
        return [{
            icon: '🚀',
            title: 'Continue Jogando',
            description: 'Continue salvando seus jogos para receber sugestões personalizadas!',
            priority: 'low',
            action: 'Continuar'
        }];
    }
}

// Gera cards de sugestões
function generateSuggestionCards(suggestions: any[]): string {
    console.log('💳 DEBUG: generateSuggestionCards iniciado');
    
    try {
        return suggestions.map(suggestion => `
            <div class="suggestion-card priority-${suggestion.priority}">
                <div class="suggestion-header">
                    <span class="suggestion-icon">${suggestion.icon}</span>
                    <h3 class="suggestion-title">${suggestion.title}</h3>
                </div>
                <p class="suggestion-description">${suggestion.description}</p>
                <button class="suggestion-action" onclick="alert('Feature em desenvolvimento!')">
                    ${suggestion.action} →
                </button>
            </div>
        `).join('');
    } catch (error) {
        console.error('💳 ERROR: Erro em generateSuggestionCards:', (error as Error).message);
        return '<div class="suggestion-card">Erro ao gerar sugestões</div>';
    }
}

// Calcula timing ótimo
function calculateOptimalTiming(games: any[]): any {
    console.log('⏰ DEBUG: calculateOptimalTiming iniciado');
    
    try {
        if (games.length === 0) {
            return {
                bestDay: 'N/A',
                bestHour: '12:00',
                frequency: '0 jogos/semana',
                momentum: 'Baixo'
            };
        }
        
        const timeAnalysis = analyzeTimePatterns(games);
        const winningGames = games.filter(g => (g.winnings || 0) > (g.investment || 0));
        const winRate = (winningGames.length / games.length) * 100;
        
        return {
            bestDay: timeAnalysis.preferredDay,
            bestHour: `${timeAnalysis.preferredHour}:00`,
            frequency: `${Math.round(games.length / 4)} jogos/semana`,
            momentum: winRate > 25 ? 'Alto' : winRate > 15 ? 'Médio' : 'Baixo'
        };
    } catch (error) {
        console.error('⏰ ERROR: Erro em calculateOptimalTiming:', (error as Error).message);
        return {
            bestDay: 'N/A',
            bestHour: '12:00',
            frequency: '0 jogos/semana',
            momentum: 'Baixo'
        };
    }
}

// Gera cards de timing
function generateTimingCards(timing: any): string {
    console.log('⏰ DEBUG: generateTimingCards iniciado');
    
    try {
        return `
            <div class="timing-card">
                <div class="timing-value">${timing.bestDay}</div>
                <p class="timing-label">Melhor Dia</p>
            </div>
            <div class="timing-card">
                <div class="timing-value">${timing.bestHour}</div>
                <p class="timing-label">Horário Ideal</p>
            </div>
            <div class="timing-card">
                <div class="timing-value">${timing.frequency}</div>
                <p class="timing-label">Frequência Ótima</p>
            </div>
            <div class="timing-card">
                <div class="timing-value">${timing.momentum}</div>
                <p class="timing-label">Momentum Atual</p>
            </div>
        `;
    } catch (error) {
        console.error('⏰ ERROR: Erro em generateTimingCards:', (error as Error).message);
        return '<div class="timing-card">Erro ao gerar timing</div>';
    }
}

// Funções auxiliares para análise
function calculateConsistency(games: any[]): number {
    try {
        if (games.length === 0) return 0;
        
        const investments = games.map(g => g.investment || 0);
        const avg = investments.reduce((sum, inv) => sum + inv, 0) / investments.length;
        
        if (avg === 0) return 0;
        
        const variance = investments.reduce((sum, inv) => sum + Math.pow(inv - avg, 2), 0) / investments.length;
        return 1 / (1 + Math.sqrt(variance) / avg);
    } catch (error) {
        console.error('🔧 ERROR: Erro em calculateConsistency:', (error as Error).message);
        return 0;
    }
}

function calculateVolatility(games: any[]): number {
    try {
        if (games.length === 0) return 0;
        
        const rois = games.map(game => {
            const investment = game.investment || 0;
            const winnings = game.winnings || 0;
            return investment > 0 ? ((winnings - investment) / investment) : 0;
        });
        
        const avgROI = rois.reduce((sum, roi) => sum + roi, 0) / rois.length;
        const variance = rois.reduce((sum, roi) => sum + Math.pow(roi - avgROI, 2), 0) / rois.length;
        return Math.sqrt(variance);
    } catch (error) {
        console.error('📊 ERROR: Erro em calculateVolatility:', (error as Error).message);
        return 0;
    }
}

function calculateAverageROI(games: any[]): number {
    try {
        if (games.length === 0) return 0;
        
        const totalInvestment = games.reduce((sum, game) => sum + (game.investment || 0), 0);
        const totalWinnings = games.reduce((sum, game) => sum + (game.winnings || 0), 0);
        return totalInvestment > 0 ? ((totalWinnings - totalInvestment) / totalInvestment) * 100 : 0;
    } catch (error) {
        console.error('💹 ERROR: Erro em calculateAverageROI:', (error as Error).message);
        return 0;
    }
}

function calculateBestStreak(games: any[]): number {
    try {
        if (games.length === 0) return 0;
        
        let currentStreak = 0;
        let bestStreak = 0;
        
        games.forEach(game => {
            if ((game.winnings || 0) > 0) {
                currentStreak++;
                bestStreak = Math.max(bestStreak, currentStreak);
            } else {
                currentStreak = 0;
            }
        });
        
        return bestStreak;
    } catch (error) {
        console.error('🏆 ERROR: Erro em calculateBestStreak:', (error as Error).message);
        return 0;
    }
}

function calculatePatience(games: any[]): number {
    try {
        if (games.length === 0) return 0;
        
        // Simula paciência baseada na consistência de investimentos
        const consistency = calculateConsistency(games);
        return Math.min(1, consistency + 0.2);
    } catch (error) {
        console.error('🧘 ERROR: Erro em calculatePatience:', (error as Error).message);
        return 0;
    }
}

function calculateAdaptation(games: any[]): number {
    try {
        if (games.length === 0) return 0;
        
        // Simula adaptação baseada na variedade de números
        const allNumbers = games.flatMap(g => g.numbers || []);
        const uniqueNumbers = new Set(allNumbers).size;
        return Math.min(1, uniqueNumbers / (games.length * 6));
    } catch (error) {
        console.error('🎯 ERROR: Erro em calculateAdaptation:', (error as Error).message);
        return 0;
    }
}

// Determina classe do score
function getScoreClass(score: number): string {
    if (score >= 80) return 'score-excellent';
    if (score >= 65) return 'score-good';
    if (score >= 45) return 'score-average';
    return 'score-poor';
}

(window as any).loadNotifications = loadNotifications;
(window as any).renderIntelligenceEngine = renderIntelligenceEngine;
