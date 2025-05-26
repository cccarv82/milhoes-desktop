import './style.css';
import './app.css';

import { GenerateStrategy, TestConnections, GetNextDraws, SaveConfig, ValidateConfig, GetDefaultConfig, GetStatistics, TestConnectionsWithConfig, DebugClaudeConfig } from '../wailsjs/go/main/App';

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
    claudeModel: 'claude-3-5-sonnet-20241022',
    timeoutSec: 60,
    maxTokens: 8000,
    verbose: false
};

// ===============================
// INICIALIZAÇÃO
// ===============================

document.addEventListener('DOMContentLoaded', async () => {
    console.log('🎰 Lottery Optimizer iniciado!');
    
    // Adicionar botão de debug
    addDebugButton();
    
    // Verificar configuração e renderizar tela apropriada
    await checkConfigAndRender();
});

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
                                <option value="claude-3-5-sonnet-20241022" ${currentConfig.claudeModel === 'claude-3-5-sonnet-20241022' ? 'selected' : ''}>Claude 3.5 Sonnet (Recomendado)</option>
                                <option value="claude-3-opus-20240229" ${currentConfig.claudeModel === 'claude-3-opus-20240229' ? 'selected' : ''}>Claude 3 Opus</option>
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
                            Teste de Conexão
                        </h3>
                        <p style="color: var(--text-secondary); margin-bottom: var(--spacing-4);">
                            Teste as conexões com as APIs antes de salvar.
                        </p>
                        
                        <div style="display: flex; gap: var(--spacing-4); margin-bottom: var(--spacing-6); flex-wrap: wrap;">
                            <button type="button" class="btn-secondary" onclick="testConnections()">
                                <span class="btn-icon">🔄</span>
                                Testar Conexões
                            </button>
                        </div>
                        
                        <div id="connectionStatus"></div>
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
                <div class="ai-badge">
                    <span class="ai-icon">🤖</span>
                    Powered by Claude AI
                </div>
            </header>
            
            <div class="main-content">
                <div class="welcome-section">
                    <h2 style="font-size: var(--font-size-4xl); font-weight: 800; margin-bottom: var(--spacing-6); background: linear-gradient(135deg, var(--accent-primary), var(--accent-secondary)); background-clip: text; -webkit-background-clip: text; -webkit-text-fill-color: transparent;">
                        Bem-vindo ao Futuro das Loterias! 🚀
                    </h2>
                    <p style="font-size: var(--font-size-xl); color: var(--text-secondary); max-width: 600px; margin: 0 auto var(--spacing-8) auto; line-height: 1.7;">
                        Utilize o poder da inteligência artificial para gerar estratégias baseadas em análise histórica, 
                        padrões estatísticos e suas preferências pessoais.
                    </p>
                </div>
                
                <div class="features-grid">
                    <div class="feature-card">
                        <span class="feature-icon">🧠</span>
                        <h3>IA Avançada</h3>
                        <p>Claude 3.5 Sonnet analisa milhares de sorteios históricos para identificar padrões e tendências únicas.</p>
                    </div>
                    
                    <div class="feature-card">
                        <span class="feature-icon">📊</span>
                        <h3>Análise Estatística</h3>
                        <p>Algoritmos sofisticados calculam probabilidades, números quentes e frios, além de padrões de frequência.</p>
                    </div>
                    
                    <div class="feature-card">
                        <span class="feature-icon">🎯</span>
                        <h3>Estratégias Personalizadas</h3>
                        <p>Configure seu orçamento, prefira números da sorte e evite padrões para estratégias totalmente customizadas.</p>
                    </div>
                    
                    <div class="feature-card">
                        <span class="feature-icon">💎</span>
                        <h3>Multi-Loteria</h3>
                        <p>Suporte completo para Mega-Sena e Lotofácil com dados sempre atualizados da CAIXA.</p>
                    </div>
                    
                    <div class="feature-card">
                        <span class="feature-icon">⚡</span>
                        <h3>Resultados Instantâneos</h3>
                        <p>Gere estratégias completas em segundos com explicações detalhadas do raciocínio da IA.</p>
                    </div>
                    
                    <div class="feature-card">
                        <span class="feature-icon">🔒</span>
                        <h3>100% Seguro</h3>
                        <p>Todos os cálculos são feitos localmente. Seus dados e preferências nunca saem do seu computador.</p>
                    </div>
                </div>
                
                <div class="cta-section">
                    <button class="btn-primary" onclick="startStrategyWizard()">
                        <span class="btn-icon">🎲</span>
                        Gerar Estratégia Inteligente
                    </button>
                    <button class="btn-secondary" onclick="renderConfigurationScreen()">
                        <span class="btn-icon">⚙️</span>
                        Configurações
                    </button>
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
                
                <!-- Estatísticas rápidas -->
                <div class="statistics-section">
                    <h3>
                        <span>📈</span>
                        Estatísticas Rápidas
                    </h3>
                    <div class="stats-grid" id="quickStats">
                        <div class="loading">Carregando estatísticas...</div>
                    </div>
                </div>
            </div>
        </div>
    `;
    
    // Carregar dados assíncronos
    loadNextDraws();
    loadConnectionStatus();
    loadQuickStats();
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

// Carregar estatísticas rápidas
async function loadQuickStats() {
    try {
        const stats = await GetStatistics();
        const container = document.getElementById('quickStats');
        
        if (!container) {
            console.warn('Elemento quickStats não encontrado');
            return;
        }
        
        let html = '';
        
        if (stats.megasena) {
            html += `
                <div class="stat-card">
                    <div class="stat-icon">🔥</div>
                    <div class="stat-content">
                        <span class="label">Mega-Sena</span>
                        <span class="value">${stats.megasena.totalDraws}</span>
                        <small>sorteios analisados</small>
                    </div>
                </div>
            `;
        }
        
        if (stats.lotofacil) {
            html += `
                <div class="stat-card">
                    <div class="stat-icon">⭐</div>
                    <div class="stat-content">
                        <span class="label">Lotofácil</span>
                        <span class="value">${stats.lotofacil.totalDraws}</span>
                        <small>sorteios analisados</small>
                    </div>
                </div>
            `;
        }
        
        if (html === '') {
            html = '<div class="no-data">Nenhuma estatística disponível</div>';
        }
        
        container.innerHTML = html;
    } catch (error) {
        const container = document.getElementById('quickStats');
        if (container) {
            container.innerHTML = '<div class="no-data">Erro ao carregar estatísticas</div>';
        }
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
                        <div class="advanced-options">
                            <div class="checkbox-option">
                                <input type="checkbox" name="avoidPatterns" id="avoidPatterns">
                                <label for="avoidPatterns">Evitar padrões óbvios (sequências, múltiplos)</label>
                            </div>
                        </div>
                        
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
        avoidPatterns: form.avoidPatterns.checked,
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
                                    <span class="game-icon">${game.type === 'megasena' ? '🔥' : '⭐'}</span>
                                    <span class="game-title">${game.type === 'megasena' ? 'Mega-Sena' : 'Lotofácil'} #${index + 1}</span>
                                    <span class="game-cost">R$ ${game.cost.toFixed(2)}</span>
                                </div>
                                <div class="game-numbers">
                                    ${game.numbers.map((num: number) => `<span class="number">${num.toString().padStart(2, '0')}</span>`).join('')}
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

                <!-- Estatísticas -->
                <div class="form-section">
                    <h3>
                        <span>📊</span>
                        Estatísticas da Análise
                    </h3>
                    <div class="stats-grid">
                        <div class="stat-card">
                            <div class="stat-icon">📈</div>
                            <div class="stat-content">
                                <span class="label">Sorteios Analisados</span>
                                <span class="value">${strategy.statistics.analyzedDraws}</span>
                            </div>
                        </div>
                        
                        <div class="stat-card">
                            <div class="stat-icon">🔥</div>
                            <div class="stat-content">
                                <span class="label">Números Quentes</span>
                                <div style="display: flex; gap: var(--spacing-1); flex-wrap: wrap; margin-top: var(--spacing-2);">
                                    ${(strategy.statistics.hotNumbers && Array.isArray(strategy.statistics.hotNumbers)) 
                                        ? strategy.statistics.hotNumbers.slice(0, 10).map(num => 
                                            `<span style="background: var(--accent-success); color: white; padding: var(--spacing-1) var(--spacing-2); border-radius: 4px; font-size: var(--font-size-sm); font-weight: 600;">${num}</span>`
                                        ).join('')
                                        : '<span style="color: var(--text-secondary);">Dados não disponíveis</span>'
                                    }
                                </div>
                            </div>
                        </div>
                        
                        <div class="stat-card">
                            <div class="stat-icon">❄️</div>
                            <div class="stat-content">
                                <span class="label">Números Frios</span>
                                <div style="display: flex; gap: var(--spacing-1); flex-wrap: wrap; margin-top: var(--spacing-2);">
                                    ${(strategy.statistics.coldNumbers && Array.isArray(strategy.statistics.coldNumbers))
                                        ? strategy.statistics.coldNumbers.slice(0, 10).map(num => 
                                            `<span style="background: var(--accent-info); color: white; padding: var(--spacing-1) var(--spacing-2); border-radius: 4px; font-size: var(--font-size-sm); font-weight: 600;">${num}</span>`
                                        ).join('')
                                        : '<span style="color: var(--text-secondary);">Dados não disponíveis</span>'
                                    }
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
                            <span class="game-title">${game.type === 'megasena' ? '🔥 Mega-Sena' : '⭐ Lotofácil'} #${index + 1}</span>
                            <span class="game-cost">R$ ${game.cost.toFixed(2)}</span>
                        </div>
                        <div class="game-numbers">
                            ${game.numbers.map((num: number) => `<span class="number">${num.toString().padStart(2, '0')}</span>`).join('')}
                        </div>
                    </div>
                `).join('')}
            </div>
            
            <div class="disclaimer">
                <p><strong>⚠️ IMPORTANTE:</strong> A loteria é um jogo de azar. Jogue com responsabilidade e apenas o que pode perder.</p>
            </div>
            
            <div class="footer">
                <p><strong>Estratégia gerada com base em análise estatística de dados históricos</strong></p>
                <p>Esta estratégia foi criada usando inteligência artificial que analisou ${strategy.statistics?.analyzedDraws || 100} sorteios históricos</p>
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
// FUNÇÕES GLOBAIS
// ===============================

// Disponibilizar funções globalmente
(window as any).renderWelcome = renderWelcome;
(window as any).renderConfigurationScreen = renderConfigurationScreen;
(window as any).renderPreferencesForm = renderPreferencesForm;
(window as any).startStrategyWizard = startStrategyWizard;
(window as any).setBudget = setBudget;
(window as any).handlePreferencesSubmit = handlePreferencesSubmit;
(window as any).handleConfigSave = handleConfigSave;
(window as any).testConnections = testConnections;
(window as any).loadDefaultConfig = loadDefaultConfig;
(window as any).checkConfigAndRender = checkConfigAndRender;
(window as any).printStrategy = printStrategy;

// Função de debug específica para Claude
async function debugClaudeConfig() {
    console.log('🔍 [DEBUG] Testando configuração do Claude...');
    
    try {
        const debugInfo = await DebugClaudeConfig();
        console.log('🔍 [DEBUG] Informações do Claude:', debugInfo);
        
        // Mostrar info detalhada no console
        console.table(debugInfo);
        
        // Mostrar alerta com informações principais
        const summary = `
🔍 DEBUG CLAUDE CONFIG:
• Has API Key: ${debugInfo.hasApiKey}
• API Key Length: ${debugInfo.apiKeyLength}
• API Key Preview: ${debugInfo.apiKeyPreview}
• API Key Valid Format: ${debugInfo.apiKeyLooksValid}
• Connection Test: ${debugInfo.connectionTest}
• Config Path: ${debugInfo.configPath}
• Config Exists: ${debugInfo.configExists}
• Claude Model: ${debugInfo.claudeModel}
• Max Tokens: ${debugInfo.maxTokens}
• Timeout: ${debugInfo.timeout}
• Verbose: ${debugInfo.verbose}
`;
        
        alert(summary);
        
        return debugInfo;
        
    } catch (error) {
        console.error('❌ [DEBUG] Erro ao testar Claude:', error);
        alert('❌ Erro ao testar configuração do Claude: ' + error);
        return null;
    }
}

// Função para adicionar botão de debug na interface
function addDebugButton() {
    const debugButton = document.createElement('button');
    debugButton.textContent = '🔍 Debug Claude';
    debugButton.style.cssText = `
        position: fixed;
        bottom: 10px;
        right: 10px;
        z-index: 9999;
        background: #007bff;
        color: white;
        border: none;
        padding: 8px 12px;
        border-radius: 4px;
        cursor: pointer;
        font-size: 12px;
    `;
    debugButton.onclick = debugClaudeConfig;
    document.body.appendChild(debugButton);
}
