# 🚀 Sistema de Launcher e Auto-Update

## Visão Geral

O projeto agora utiliza um sistema de **launcher + app principal** que permite atualizações automáticas sem interromper o usuário.

## Estrutura do Sistema

```
Milhões Lottery Optimizer/
├── launcher.exe          # ← PONTO DE ENTRADA (usuário executa este)
├── milhoes.exe           # ← APP PRINCIPAL (executado pelo launcher)
├── apply_update.bat      # ← Script de atualização (criado automaticamente)
├── update_*.zip          # ← Arquivos de atualização (temporários)
└── logs/                 # ← Logs do sistema
```

## Como Funciona

### 1. **Inicialização**
```
Usuário clica → launcher.exe → verifica updates → inicia milhoes.exe
```

### 2. **Verificação de Atualizações**
- **Automática**: A cada 30 segundos após iniciar
- **Silenciosa**: Download e instalação em background
- **Não-intrusiva**: Usuário continua usando normalmente

### 3. **Processo de Atualização**
```
1. Launcher verifica GitHub releases
2. Se nova versão disponível:
   ├── Download em background
   ├── Prepara script de atualização
   └── Próxima execução usa nova versão
3. Usuário é notificado apenas quando concluído
```

## Repositórios

- **Privado**: `cccarv82/milhoes-desktop` (código-fonte)
- **Público**: `cccarv82/milhoes-releases` (releases públicos)

## Build e Instalação

### Build Local (Desenvolvimento)
```bash
# Build rápido (só launcher)
build_dev.bat

# Build completo (com instalador)
build_release.bat
```

### Build CI/CD (GitHub Actions)
```yaml
1. Compila milhoes.exe (Wails)
2. Compila launcher.exe (Go)
3. Gera instalador (Inno Setup)
4. Publica no repositório público
```

### Instalador (Inno Setup)
- **Instala AMBOS**: launcher.exe + milhoes.exe
- **Atalhos apontam para**: launcher.exe (não milhoes.exe)
- **Associações de arquivo**: launcher.exe

## Fluxo do Usuário

### Primeira Instalação
1. Baixa `MilhoesSetup.exe`
2. Executa instalador
3. Atalho criado aponta para `launcher.exe`

### Uso Diário
1. Clica no atalho (launcher.exe)
2. Launcher verifica atualizações
3. Launcher inicia milhoes.exe
4. Usuário usa normalmente

### Atualizações
1. Sistema detecta nova versão
2. Download silencioso em background
3. Notificação: "Próxima execução será atualizada"
4. Usuário continua usando versão atual
5. Na próxima abertura: versão atualizada

## Vantagens

### ✅ Para o Usuário
- **Zero interrupção**: Nunca precisa parar de usar
- **Zero ação necessária**: Atualizações automáticas
- **Sempre atualizado**: Versão mais recente automaticamente

### ✅ Para o Desenvolvedor
- **Releases públicos**: Sem expor código-fonte
- **Atualizações garantidas**: Todos usuários sempre atualizados
- **Logs detalhados**: Debug fácil de problemas
- **Rollback automático**: Se atualização falhar, continua com versão atual

## Arquivos de Configuração

### `launcher.exe`
- **Função**: Gerenciamento de atualizações e inicialização
- **Configuração**: Hardcoded (repositório GitHub)
- **Logs**: `logs/launcher-YYYY-MM-DD.log`

### `milhoes.exe`
- **Função**: Aplicação principal (Wails + React)
- **Configuração**: `lottery-optimizer.yaml`
- **Logs**: `logs/lottery-optimizer-YYYY-MM-DD.log`

## Debugging

### Logs Importantes
```bash
# Logs do launcher
logs/launcher-YYYY-MM-DD.log

# Logs do app principal  
logs/lottery-optimizer-YYYY-MM-DD.log
```

### Problemas Comuns

**1. Atualização não funciona**
- Verifique conexão com internet
- Verifique logs do launcher
- Repositório `cccarv82/milhoes-releases` deve estar público

**2. App não inicia após atualização**
- Launcher volta para versão anterior automaticamente
- Verifique logs de erro

**3. Instalador não encontra arquivos**
- Execute `build_release.bat` localmente
- Verifique se ambos executáveis foram gerados

## Desenvolvimento

### Estrutura de Código
```
cmd/launcher/main.go       # ← Código do launcher
app.go                     # ← App principal (Wails)
installer/setup.iss        # ← Script do instalador
.github/workflows/         # ← CI/CD
```

### Comandos Úteis
```bash
# Desenvolvimento rápido
wails dev

# Test launcher isolado
go run cmd/launcher/main.go

# Build completo local
build_release.bat

# Build só pra testar
build_dev.bat
```

### Versioning
- **App Principal**: `main.go` → `version = "v1.1.3"`
- **Launcher**: `cmd/launcher/main.go` → `launcherVersion = "v1.0.0"`
- **Instalador**: `installer/setup.iss` → `AppVersion=1.1.3`

## Próximas Melhorias

- [ ] Interface gráfica para launcher (opcional)
- [ ] Rollback automático se nova versão falhar
- [ ] Configuração de canal de updates (stable/beta)
- [ ] Delta updates (apenas diferenças)
- [ ] Assinatura digital dos updates 