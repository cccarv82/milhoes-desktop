# 🎰 Milhões - Otimizador Inteligente de Loterias

![Version](https://img.shields.io/github/v/release/yourusername/milhoes)
![License](https://img.shields.io/github/license/yourusername/milhoes)
![Platform](https://img.shields.io/badge/platform-Windows-blue)

## 🎯 Sobre o Projeto

**Milhões** é um otimizador inteligente de loterias que utiliza **IA Claude** para analisar padrões históricos e gerar estratégias matemáticas otimizadas para **Mega-Sena** e **Lotofácil**.

### ✨ Principais Funcionalidades

- 🧠 **IA Claude Integrada** - Análise avançada com inteligência artificial
- 📊 **Dados CAIXA em Tempo Real** - Sorteios históricos atualizados
- 💰 **Otimização de Orçamento** - Máximo retorno para seu investimento
- 📈 **Análise Estatística** - Padrões, frequências e tendências
- 🎮 **Interface Moderna** - Design intuitivo e responsivo
- 🔄 **Atualizações Automáticas** - Sistema de launcher com updates silenciosos
- 💾 **Jogos Salvos** - Sistema de verificação automática de resultados

## 🚀 Sistema de Auto-Update

O **Milhões** utiliza um sistema inovador de **launcher + auto-update** que garante que você sempre tenha a versão mais recente:

### Como Funciona
- **🚀 Launcher**: Ponto de entrada que gerencia atualizações
- **⚙️ Update Silencioso**: Downloads em background sem interromper o uso
- **🔄 Zero Interrupção**: Continue usando enquanto a atualização é preparada
- **✨ Próxima Execução**: Nova versão aplicada automaticamente na próxima abertura

### Estrutura
```
📁 Milhões Lottery Optimizer/
├── launcher.exe          # ← Execute este (criado pelo instalador)
├── milhoes.exe           # ← App principal (gerenciado automaticamente)
├── logs/                 # ← Logs detalhados
└── data/                 # ← Jogos salvos e configurações
```

> **📌 Importante**: Sempre execute o `launcher.exe` (atalhos criados pelo instalador já apontam corretamente)

## 🚀 Instalação

### Windows (Recomendado)

1. **Baixe o instalador** na [página de releases](https://github.com/cccarv82/milhoes-releases/releases)
2. **Execute** `MilhoesSetup.exe` como administrador
3. **Siga** o assistente de instalação
4. **Use o atalho criado** (aponta automaticamente para o launcher)
5. **Configure** sua chave da API do Claude

### Versão Portável

1. **Baixe** `milhoes-windows-amd64.zip`
2. **Extraia** para uma pasta de sua escolha
3. **Execute** `launcher.exe` (não milhoes.exe diretamente)

## ⚙️ Configuração Inicial

### 1. Obter Chave da API Claude

1. Acesse [Claude Console](https://console.anthropic.com/)
2. Crie uma conta ou faça login
3. Gere uma nova API Key
4. Copie a chave (formato: `sk-ant-...`)

### 2. Configurar no App

1. Abra o **Milhões** (via launcher)
2. Vá em **Menu → Configurações**
3. Cole sua **Chave da API Claude**
4. Clique em **Testar Conexão**
5. **Salve** as configurações

## 🎮 Como Usar

### Gerando uma Estratégia

1. **Selecione** os tipos de loteria (Mega-Sena, Lotofácil)
2. **Defina** seu orçamento disponível
3. **Escolha** a estratégia (Inteligente recomendada)
4. **Configure** preferências opcionais:
   - Números favoritos
   - Números a evitar
   - Evitar padrões óbvios
5. **Clique** em "Gerar Estratégia"
6. **Aguarde** a análise da IA (30-60 segundos)
7. **Revise** os jogos sugeridos
8. **Salve** jogos para verificação automática de resultados
9. **Imprima** ou exporte sua estratégia

### Recursos Avançados

- 📊 **Estatísticas**: Visualize padrões históricos
- 🎯 **Análise de Confiança**: Veja o nível de confiança da IA
- 💡 **Explicações Detalhadas**: Entenda o raciocínio por trás da estratégia
- 🔄 **Múltiplas Tentativas**: Gere diferentes variações
- 💾 **Jogos Salvos**: Verificação automática de resultados
- 📈 **Histórico de Resultados**: Acompanhe seus jogos anteriores

## 🛠️ Desenvolvimento

### Tecnologias Utilizadas

- **Frontend**: TypeScript + React + Wails v2
- **Backend**: Go 1.21+
- **IA**: Claude 3.5 Sonnet (Anthropic)
- **APIs**: CAIXA Loterias
- **Auto-Update**: Sistema próprio com launcher
- **Build**: GitHub Actions + Inno Setup
- **Database**: SQLite (jogos salvos)

### Build Local

```bash
# Clone o repositório
git clone https://github.com/cccarv82/milhoes-desktop.git
cd milhoes-desktop

# Build rápido (desenvolvimento)
build_dev.bat

# Build completo (com instalador)
build_release.bat

# Desenvolvimento interativo
wails dev
```

### Scripts de Build

- **`build_dev.bat`** - Build rápido apenas do launcher
- **`build_release.bat`** - Build completo (launcher + app + instalador)
- **`wails dev`** - Desenvolvimento com hot-reload

### Estrutura do Projeto

```
milhoes/
├── cmd/launcher/              # 🚀 Sistema de Launcher
│   └── main.go               # ├── Gerenciamento de updates
├── app.go                    # 🌉 Bridge Go ↔ Frontend  
├── internal/
│   ├── ai/                   # 🧠 Cliente Claude AI
│   ├── data/                 # 📊 APIs da CAIXA
│   ├── lottery/              # 🎰 Lógica das loterias
│   ├── config/               # ⚙️ Configurações
│   ├── updater/              # 🔄 Sistema de atualização
│   ├── database/             # 💾 SQLite (jogos salvos)
│   └── services/             # 🛠️ Serviços (verificação resultados)
├── frontend/                 # 🎨 Interface TypeScript
├── installer/setup.iss       # 📦 Instalador Windows
├── .github/workflows/        # 🤖 CI/CD Automático
└── LAUNCHER_README.md        # 📖 Documentação detalhada do launcher
```

## 🔄 Sistema de Releases

### Repositórios
- **🔒 Privado**: `cccarv82/milhoes-desktop` (código-fonte)
- **🌍 Público**: `cccarv82/milhoes-releases` (releases para usuários)

### CI/CD Automático
1. **Build**: Compila launcher + app principal
2. **Test**: Validação automática
3. **Package**: Cria instalador Windows
4. **Release**: Publica no repositório público
5. **Auto-Update**: Usuários recebem automaticamente

## 🤝 Contribuindo

1. **Fork** o repositório
2. **Crie** uma branch: `git checkout -b feature/nova-funcionalidade`
3. **Commit** suas mudanças: `git commit -m 'feat: nova funcionalidade'`
4. **Push** para a branch: `git push origin feature/nova-funcionalidade`
5. **Abra** um Pull Request

### 🐛 Debug e Logs

```bash
# Logs do launcher
logs/launcher-YYYY-MM-DD.log

# Logs do app principal
logs/lottery-optimizer-YYYY-MM-DD.log

# Banco de dados (jogos salvos)
data/saved_games.db
```

## 📄 Licença

Este projeto está licenciado sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🆘 Suporte

- 🐛 **Reportar Bugs**: [Issues](https://github.com/cccarv82/milhoes-releases/issues)
- 💬 **Discussões**: [Discussions](https://github.com/cccarv82/milhoes-releases/discussions)
- 📖 **Documentação**: [LAUNCHER_README.md](LAUNCHER_README.md)
- 📧 **Contato**: [Email](mailto:suporte@milhoes.app)

## ⚠️ Aviso Legal

Este software é para fins educacionais e de entretenimento. Jogue com responsabilidade. Apostas podem causar dependência.

---

<div align="center">
  <strong>🎯 Feito com ❤️ para otimizar suas chances nas loterias</strong>
  <br>
  <em>Sistema de Auto-Update • Zero Configuração • Sempre Atualizado</em>
</div>
 