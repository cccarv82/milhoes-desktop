@echo off
echo ===========================================
echo  TESTE DO SISTEMA DE AUTO-UPDATE
echo ===========================================
echo.

echo 📋 Status atual:
echo   - Versao local: %cd%
echo   - Executaveis presentes:
dir *.exe /b

echo.
echo 🔍 Verificando ultima versao disponivel...
powershell -Command "try { $latest = Invoke-RestMethod -Uri 'https://api.github.com/repos/cccarv82/milhoes-releases/releases/latest'; Write-Host '✅ Ultima versao:' $latest.tag_name; Write-Host '📦 Instalador:' $latest.assets[0].browser_download_url; } catch { Write-Host '❌ Erro ao verificar:' $_.Exception.Message }"

echo.
echo 📥 Para testar o auto-update corretamente:
echo.
echo 1. Baixe o instalador oficial:
echo    https://github.com/cccarv82/milhoes-releases/releases/latest/download/MilhoesSetup.exe
echo.
echo 2. Execute como administrador
echo.
echo 3. Use o launcher instalado (nao este local)
echo.
echo 4. O launcher oficial ira detectar atualizacoes automaticamente
echo.

echo 🚀 Abrindo pagina de releases...
start https://github.com/cccarv82/milhoes-releases/releases/latest

echo.
echo ⚠️  IMPORTANTE: 
echo    - Seu executavel local (milhoes.exe) e versao v1.1.5
echo    - A versao oficial mais recente e v1.1.6
echo    - Para testar updates, use a versao INSTALADA, nao a local
echo.
pause 