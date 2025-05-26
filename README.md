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
- 🔄 **Atualizações Automáticas** - Sempre na versão mais recente

## 🚀 Instalação

### Windows (Recomendado)

1. **Baixe o instalador** na [página de releases](https://github.com/yourusername/milhoes/releases)
2. **Execute** `MilhoesSetup.exe` como administrador
3. **Siga** o assistente de instalação
4. **Configure** sua chave da API do Claude

### Versão Portável

1. **Baixe** `milhoes-windows-amd64.zip`
2. **Extraia** para uma pasta de sua escolha
3. **Execute** `milhoes.exe`

## ⚙️ Configuração Inicial

### 1. Obter Chave da API Claude

1. Acesse [Claude Console](https://console.anthropic.com/)
2. Crie uma conta ou faça login
3. Gere uma nova API Key
4. Copie a chave (formato: `sk-ant-...`)

### 2. Configurar no App

1. Abra o **Milhões**
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
8. **Imprima** ou salve sua estratégia

### Recursos Avançados

- 📊 **Estatísticas**: Visualize padrões históricos
- 🎯 **Análise de Confiança**: Veja o nível de confiança da IA
- 💡 **Explicações Detalhadas**: Entenda o raciocínio por trás da estratégia
- 🔄 **Múltiplas Tentativas**: Gere diferentes variações

## 🛠️ Desenvolvimento

### Tecnologias Utilizadas

- **Frontend**: TypeScript + Wails
- **Backend**: Go 1.21+
- **IA**: Claude 3.5 Sonnet (Anthropic)
- **APIs**: CAIXA Loterias
- **Build**: GitHub Actions + Inno Setup

### Executar Localmente

```bash
# Clone o repositório
git clone https://github.com/yourusername/milhoes.git
cd milhoes

# Instale dependências
go mod download

# Execute em modo de desenvolvimento
wails dev

# Build para produção
wails build
```

### Estrutura do Projeto

```
milhoes/
├── app.go                     # Bridge Go ↔ Frontend
├── internal/
│   ├── ai/                    # Cliente Claude AI
│   ├── data/                  # APIs da CAIXA
│   ├── lottery/               # Lógica das loterias
│   ├── config/                # Configurações
│   └── updater/               # Sistema de atualização
├── frontend/                  # Interface TypeScript
├── .github/workflows/         # CI/CD Automático
└── installer/                 # Instalador Windows
```

## 🤝 Contribuindo

1. **Fork** o repositório
2. **Crie** uma branch: `git checkout -b feature/nova-funcionalidade`
3. **Commit** suas mudanças: `git commit -m 'feat: nova funcionalidade'`
4. **Push** para a branch: `git push origin feature/nova-funcionalidade`
5. **Abra** um Pull Request

## 📄 Licença

Este projeto está licenciado sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para detalhes.

## 🆘 Suporte

- 🐛 **Reportar Bugs**: [Issues](https://github.com/yourusername/milhoes/issues)
- 💬 **Discussões**: [Discussions](https://github.com/yourusername/milhoes/discussions)
- 📧 **Contato**: [Email](mailto:suporte@milhoes.app)

## ⚠️ Aviso Legal

Este software é para fins educacionais e de entretenimento. Jogue com responsabilidade. Apostas podem causar dependência.

---

<div align="center">
  <strong>🎯 Feito com ❤️ para otimizar suas chances nas loterias</strong>
</div>
