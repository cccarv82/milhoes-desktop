# 📋 Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### 🚀 Features
- Initial project setup with modern Go architecture
- Claude Sonnet 4 AI integration for lottery analysis
- Interactive CLI interface with colorful output
- Support for Mega Sena and Lotofácil lotteries
- Three strategy types: Conservative, Balanced, Aggressive
- Real-time data fetching from CAIXA APIs
- Strategy validation and optimization
- Cross-platform builds with GoReleaser
- Docker containerization with multi-stage builds
- Comprehensive CI/CD pipeline with GitHub Actions

### 🔧 Technical
- Go 1.22+ with modern dependencies
- Cobra CLI framework for command structure
- Viper for configuration management
- Resty for HTTP client functionality
- PromptUI for interactive user experience
- Claude API integration with detailed prompting
- Statistical analysis and pattern detection
- Budget optimization algorithms
- Multi-platform release automation

### 📊 Quality Assurance
- GitHub Actions CI/CD pipeline
- Cross-platform testing (Linux, Windows, macOS)
- Security scanning with gosec and Trivy
- Code quality checks with golangci-lint
- Docker image optimization
- Automated releases with GoReleaser
- Comprehensive documentation

## [1.0.20] - 2025-05-26

### ✨ Nova Funcionalidade - Jogos Salvos

#### 🎯 Principais Adições:
- **Sistema de Jogos Salvos**: Salve seus jogos gerados para acompanhar resultados automaticamente
- **Verificação Automática**: Sistema verifica resultados a cada 6 horas automaticamente
- **Interface Completa**: Tela dedicada para gerenciar jogos salvos com filtros e estatísticas
- **Notificações**: Alertas visuais sobre ganhos e verificações de resultados

#### 🔧 Melhorias Técnicas:
- **Banco SQLite Puro Go**: Implementado com `modernc.org/sqlite` (sem dependência CGO)
- **API Robusta**: 6 novos endpoints para funcionalidade completa de jogos salvos
- **Armazenamento Local**: Dados salvos localmente no diretório da aplicação
- **Debug Avançado**: Ferramenta de diagnóstico para troubleshooting

#### 📊 Interface de Usuário:
- **Modal de Salvamento**: Interface intuitiva para salvar jogos com data automática
- **Filtros Inteligentes**: Filtre por loteria, status e período
- **Cards Visuais**: Design moderno com indicadores de status coloridos
- **Resultados Detalhados**: Visualização clara de acertos e prêmios

#### 🛠 Backend Robusto:
- **Verificador de Resultados**: Serviço automático integrado com API da CAIXA
- **Tratamento de Erros**: Sistema robusto de fallback e recovery
- **Performance**: Indexação otimizada no banco de dados
- **Escalabilidade**: Arquitetura preparada para futuras expansões

### 🐛 Correções:
- Corrigido problema de inicialização do banco SQLite
- Melhorado tratamento de erros de rede
- Otimizado performance geral da aplicação

### 📈 Estatísticas da Versão:
- **6 novos endpoints** de API
- **2 novas telas** no frontend
- **1 banco de dados** SQLite implementado
- **100% funcional** em ambiente de produção

## [1.0.19] - 2025-05-25

### 🔧 Melhorias de Infraestrutura:
- Preparação para sistema de jogos salvos
- Refatoração da arquitetura de dados
- Melhorias no sistema de configuração

## [1.0.18] e anteriores

### 🚀 Funcionalidades Base:
- Sistema de geração de estratégias com IA Claude
- Integração com API da CAIXA
- Interface moderna e responsiva
- Sistema de configuração avançado
- Análise estatística de dados históricos

## [1.0.0] - 2025-01-27

### 🎉 Initial Release
- First stable release of Lottery Optimizer
- Complete AI-powered lottery strategy generation
- Full CLI functionality
- Production-ready deployment options

---

## Types of Changes

- 🚀 **Features** - New features and enhancements
- 🐛 **Bug Fixes** - Bug fixes and corrections
- 🔐 **Security** - Security improvements and fixes
- 📈 **Performance** - Performance improvements
- 🔧 **Technical** - Technical improvements and refactoring
- 📊 **Quality** - Quality assurance and testing improvements
- 📝 **Documentation** - Documentation updates
- 🎨 **UI/UX** - User interface and experience improvements
- 🔄 **Dependencies** - Dependency updates
- ⚠️ **Breaking** - Breaking changes (major version bumps)

---

**Legend:**
- `[Unreleased]` - Changes not yet released
- `[X.Y.Z]` - Released version with date
- Links to compare versions available in repository 