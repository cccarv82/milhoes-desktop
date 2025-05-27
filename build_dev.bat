@echo off
echo.
echo 🚀 BUILD DESENVOLVIMENTO - MILHOES
echo ===================================
echo.

REM Verificar se estamos no diretório correto
if not exist "main.go" (
    echo ❌ ERRO: Execute na raiz do projeto
    pause
    exit /b 1
)

REM Compilar apenas o launcher (rápido)
echo 🔧 Compilando launcher...
cd cmd\launcher
go build -o launcher.exe .
if errorlevel 1 (
    echo ❌ ERRO: Falha ao compilar launcher
    cd ..\..
    pause
    exit /b 1
)
cd ..\..
echo ✅ Launcher compilado

REM Rodar Wails em modo dev (se quiser app principal)
echo.
echo 🤔 Opções:
echo   1. Testar apenas LAUNCHER
echo   2. Rodar WAILS DEV (app completo)
echo   3. Sair
echo.
set /p CHOICE="Escolha (1/2/3): "

if "!CHOICE!" == "1" (
    echo.
    echo 🚀 Testando launcher...
    cmd\launcher\launcher.exe
) else if "!CHOICE!" == "2" (
    echo.
    echo 🚀 Iniciando Wails Dev...
    wails dev
) else (
    echo ✅ Saindo...
)

echo.
pause 