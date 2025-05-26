# 📋 Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.21.2] - 2025-05-26

### 🔧 Corrigido
- **Windows Installer**: Corrigido caminho de saída do instalador (`installer\Output` ao invés de `build\installer`)
- **CI/CD Pipeline**: Pipeline agora gera corretamente tanto o arquivo ZIP portátil quanto o instalador MSI
- **Release Process**: Processo de release completamente funcional com ambos os formatos de distribuição

### 📋 Detalhes Técnicos
- OutputDir corrigido no `setup.iss` para alinhar com workflow GitHub Actions
- Instalador Windows (`MilhoesSetup.exe`) agora é gerado e incluído nos releases
- Ambos os formatos disponíveis: portátil (ZIP) e instalador profissional (EXE)

## [1.0.21.1] - 2025-05-26

### 🔧 Corrigido
- **CI/CD Pipeline**: Corrigido problema de formatação Go que impedia builds automáticos
- **Setup Installer**: Removido referência ao arquivo `appicon.ico` inexistente
- **Repository URL**: Corrigido `githubRepo` para `cccarv82/milhoes-desktop`
- **Formatação Código**: Aplicado `gofmt -s -w` em todos os arquivos Go
- **Versionamento**: Sincronizado versões em todos os arquivos de configuração

### 📋 Detalhes Técnicos
- Formatação automática de código Go para compliance com CI
- Configuração do instalador ajustada para remover dependências inexistentes
- Pipeline de release otimizado para builds automáticos

## [1.0.21] - 2025-05-26

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

### ✨ Adicionado
- **Sistema de Auto-Update Completo**: Verificação automática de atualizações a cada 6 horas
- **Verificação Inicial**: Check de updates 30 segundos após inicialização do app
- **Badge de Versão**: Exibição da versão atual no header da interface
- **Verificação Manual**: Botão para verificar atualizações manualmente nas configurações
- **Logs Detalhados**: Sistema de logging completo para monitoramento de updates
- **Interface de Auto-Update**: Seção dedicada nas configurações com informações do sistema

### 🔧 Melhorado
- **Função GetAppInfo**: Nova API para obter informações detalhadas do aplicativo
- **Inicialização do App**: Startup aprimorado com inicialização automática do sistema de updates
- **Interface de Configurações**: Seção expandida com informações de versão e auto-update
- **Experiência do Usuário**: Feedback visual claro sobre status de atualizações

### 🐛 Corrigido
- **Bindings TypeScript**: Regeneração correta dos bindings para novas funções
- **Campos UpdateInfo**: Correção dos nomes de campos (available/version vs Available/Version)
- **Compilação**: Resolução de erros de compilação relacionados ao auto-update

### 📋 Técnico
- **Backend**: Implementação completa do sistema de auto-update no startup
- **Frontend**: Integração das funções de verificação manual e exibição de versão
- **CSS**: Estilização do badge de versão e seções de auto-update
- **Logs**: Sistema de logging estruturado para debugging e monitoramento

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