# 🚀 Distribuição Automatizada - Milhões

## 📦 Sistema Completo de Distribuição

Este projeto possui um sistema completo de **build**, **release** e **atualização automática**.

---

## 🔧 Como Funciona

### 1️⃣ **Build Automático (GitHub Actions)**
- **Trigger**: Push de tag `v*` (ex: `v1.0.0`)
- **Processo**: 
  - Build multiplataforma
  - Criação do instalador Windows
  - Upload automático para GitHub Releases

### 2️⃣ **Instalador Profissional**
- **Inno Setup** para Windows
- **Instalação silenciosa** suportada
- **Registro no sistema** com desinstalador
- **Associação de arquivos** .lottery

### 3️⃣ **Auto-Update Integrado**
- **Verificação automática** de novas versões
- **Download inteligente** do instalador
- **Instalação sem interrupção** do usuário

---

## 🚀 Como Fazer um Release

### 1️⃣ **Preparar Versão**
```bash
# 1. Fazer commit das alterações
git add .
git commit -m "feat: nova funcionalidade X"

# 2. Criar tag de versão
git tag v1.0.0

# 3. Push da tag (dispara build automático)
git push origin v1.0.0
```

### 2️⃣ **Processo Automático**
O GitHub Actions automaticamente:
- ✅ Faz build do app
- ✅ Cria instalador Windows  
- ✅ Gera release com assets
- ✅ Anexa `MilhoesSetup.exe` e `.zip`

### 3️⃣ **Resultado**
- **Release público**: https://github.com/yourusername/milhoes/releases
- **Instalador**: `MilhoesSetup.exe` 
- **ZIP portável**: `milhoes-windows-amd64.zip`

---

## 📋 Versionamento Semântico

### Formato: `v{MAJOR}.{MINOR}.{PATCH}`

- **MAJOR**: Mudanças incompatíveis
- **MINOR**: Novas funcionalidades
- **PATCH**: Correções de bugs

### Exemplos:
- `v1.0.0` - Release inicial
- `v1.1.0` - Nova funcionalidade
- `v1.1.1` - Correção de bug
- `v2.0.0` - Mudança incompatível

---

## 🔄 Sistema de Auto-Update

### **Para Usuários**
1. **Instalação inicial**: Baixar `MilhoesSetup.exe`
2. **Atualizações**: Automáticas via app
3. **Notificação**: Popup quando nova versão disponível
4. **Um clique**: Download e instalação automática

### **Para Desenvolvedores**
```go
// Verificar atualizações
updateInfo, err := app.CheckForUpdates()
if err == nil && updateInfo.Available {
    fmt.Printf("Nova versão %s disponível!\n", updateInfo.Version)
    
    // Download
    err = app.DownloadUpdate(updateInfo)
    if err == nil {
        // Instalar (reinicia o app)
        app.InstallUpdate(updateInfo)
    }
}
```

---

## 🛠️ Configuração Inicial

### 1️⃣ **Configurar Repositório GitHub**
```bash
# Substitua no arquivo installer/setup.iss
AppPublisherURL=https://github.com/SEU_USER/milhoes
AppSupportURL=https://github.com/SEU_USER/milhoes/issues
AppUpdatesURL=https://github.com/SEU_USER/milhoes/releases

# Substitua no arquivo app.go
githubRepo = "SEU_USER/milhoes"
```

### 2️⃣ **Secrets do GitHub** (se necessário)
- `GITHUB_TOKEN` já incluído automaticamente
- Adicionar secrets extras se necessário

### 3️⃣ **Primeira Release**
```bash
git tag v1.0.0
git push origin v1.0.0
```

---

## 📁 Estrutura de Arquivos

```
milhoes/
├── .github/workflows/release.yml    # CI/CD automático
├── installer/setup.iss              # Script do instalador
├── config/lottery-optimizer.yaml.example  # Config exemplo
├── internal/updater/                # Sistema de update
├── DISTRIBUTION.md                  # Esta documentação
└── app.go                           # Integração do updater
```

---

## ✅ Checklist de Release

- [ ] Testar funcionalidades localmente
- [ ] Atualizar versão no código se necessário
- [ ] Fazer commit das alterações
- [ ] Criar tag com `git tag vX.Y.Z`
- [ ] Push da tag `git push origin vX.Y.Z`
- [ ] Aguardar build automático (~5-10 min)
- [ ] Verificar release no GitHub
- [ ] Testar instalador baixado
- [ ] Divulgar release para usuários

---

## 🎯 Benefícios

### **Para Usuários**
- ✅ **Instalação profissional** (não é só um .exe)
- ✅ **Atualizações automáticas** sem esforço
- ✅ **Desinstalação limpa** via Painel de Controle
- ✅ **Sempre na versão mais recente**

### **Para Desenvolvedores**  
- ✅ **Zero trabalho manual** de build/release
- ✅ **Distribuição escalável** via GitHub
- ✅ **Versionamento automático**
- ✅ **Feedback rápido** dos usuários

---

## 🔥 Resultado Final

**Sistema de distribuição de nível empresarial:**
- 🚀 **CI/CD completamente automatizado**
- 📦 **Instalador profissional Windows**  
- 🔄 **Auto-update sem fricção**
- 📊 **Versionamento semântico**
- 🎯 **Experiência premium** para usuários

**Um simples `git push` de tag = Release completo pronto para download!** 💪 