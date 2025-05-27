@echo off
setlocal enabledelayedexpansion

echo.
echo ========================================
echo 🚀 BUILD SCRIPT - MILHOES LOTTERY OPTIMIZER
echo ========================================
echo.

REM Verificar se estamos no diretório correto
if not exist "main.go" (
    echo ❌ ERRO: Execute este script na raiz do projeto
    echo    Certifique-se que main.go existe no diretório atual
    pause
    exit /b 1
)

echo 📁 Diretório atual: %CD%
echo.

REM Limpar builds anteriores
echo 🧹 Limpando builds anteriores...
if exist "build\bin" rmdir /s /q "build\bin"
if exist "cmd\launcher\launcher.exe" del /f /q "cmd\launcher\launcher.exe"
if exist "installer\Output" rmdir /s /q "installer\Output"
echo ✅ Limpeza concluída
echo.

REM Verificar Go
echo 🔧 Verificando Go...
go version >nul 2>&1
if errorlevel 1 (
    echo ❌ ERRO: Go não encontrado
    echo    Instale Go 1.21+ e adicione ao PATH
    pause
    exit /b 1
)
go version
echo.

REM Verificar Wails
echo 🔧 Verificando Wails...
wails version >nul 2>&1
if errorlevel 1 (
    echo ❌ ERRO: Wails não encontrado
    echo    Instale Wails: go install github.com/wailsapp/wails/v2/cmd/wails@latest
    pause
    exit /b 1
)
wails version
echo.

REM Compilar Launcher
echo 🚀 Compilando Launcher...
cd cmd\launcher
go build -o launcher.exe -ldflags "-s -w" .
if errorlevel 1 (
    echo ❌ ERRO: Falha ao compilar launcher
    cd ..\..
    pause
    exit /b 1
)
cd ..\..

if exist "cmd\launcher\launcher.exe" (
    for %%i in ("cmd\launcher\launcher.exe") do (
        echo ✅ Launcher compilado: %%~zi bytes
    )
) else (
    echo ❌ ERRO: launcher.exe não foi gerado
    pause
    exit /b 1
)
echo.

REM Compilar App Principal com Wails
echo 🚀 Compilando App Principal (Wails)...
wails build -platform windows/amd64 -ldflags "-X main.version=v1.1.3-local"
if errorlevel 1 (
    echo ❌ ERRO: Falha ao compilar app principal
    pause
    exit /b 1
)

if exist "build\bin\milhoes.exe" (
    for %%i in ("build\bin\milhoes.exe") do (
        echo ✅ App principal compilado: %%~zi bytes
    )
) else (
    echo ❌ ERRO: milhoes.exe não foi gerado
    echo 🔍 Verificando estrutura de build...
    if exist "build" (
        echo Conteúdo do diretório build:
        dir /s build
    ) else (
        echo Diretório build não existe
    )
    pause
    exit /b 1
)
echo.

REM Verificar arquivos necessários para o instalador
echo 📋 Verificando arquivos necessários...
set FILES_OK=1

if not exist "build\bin\milhoes.exe" (
    echo ❌ Arquivo ausente: build\bin\milhoes.exe
    set FILES_OK=0
)

if not exist "cmd\launcher\launcher.exe" (
    echo ❌ Arquivo ausente: cmd\launcher\launcher.exe
    set FILES_OK=0
)

if not exist "installer\setup.iss" (
    echo ❌ Arquivo ausente: installer\setup.iss
    set FILES_OK=0
)

if !FILES_OK! == 0 (
    echo ❌ ERRO: Arquivos necessários não encontrados
    pause
    exit /b 1
)

echo ✅ Todos os arquivos necessários estão presentes
echo.

REM Verificar Inno Setup
echo 🔧 Verificando Inno Setup...
where iscc >nul 2>&1
if errorlevel 1 (
    echo ⚠️ AVISO: Inno Setup não encontrado no PATH
    echo   Tentando caminhos padrão...
    
    set INNO_PATH=""
    if exist "C:\Program Files (x86)\Inno Setup 6\iscc.exe" (
        set "INNO_PATH=C:\Program Files (x86)\Inno Setup 6\iscc.exe"
    ) else if exist "C:\Program Files\Inno Setup 6\iscc.exe" (
        set "INNO_PATH=C:\Program Files\Inno Setup 6\iscc.exe"
    ) else (
        echo ❌ ERRO: Inno Setup não encontrado
        echo    Instale o Inno Setup 6: https://jrsoftware.org/isinfo.php
        pause
        exit /b 1
    )
    
    echo ✅ Inno Setup encontrado: !INNO_PATH!
) else (
    set "INNO_PATH=iscc"
    echo ✅ Inno Setup encontrado no PATH
)
echo.

REM Gerar Instalador
echo 🏗️ Gerando instalador...
!INNO_PATH! installer\setup.iss
if errorlevel 1 (
    echo ❌ ERRO: Falha ao gerar instalador
    pause
    exit /b 1
)

if exist "installer\Output\MilhoesSetup.exe" (
    for %%i in ("installer\Output\MilhoesSetup.exe") do (
        echo ✅ Instalador gerado: %%~zi bytes
    )
) else (
    echo ❌ ERRO: MilhoesSetup.exe não foi gerado
    echo 🔍 Verificando diretório Output...
    if exist "installer\Output" (
        dir installer\Output
    ) else (
        echo Diretório installer\Output não existe
    )
    pause
    exit /b 1
)
echo.

REM Resumo Final
echo ========================================
echo ✅ BUILD CONCLUÍDO COM SUCESSO!
echo ========================================
echo.
echo 📦 Arquivos gerados:
echo   • cmd\launcher\launcher.exe
echo   • build\bin\milhoes.exe  
echo   • installer\Output\MilhoesSetup.exe
echo.
echo 🚀 Para testar:
echo   1. Execute: cmd\launcher\launcher.exe
echo   2. Ou instale: installer\Output\MilhoesSetup.exe
echo.

REM Perguntar se quer testar
echo 🤔 Deseja testar o launcher agora? (S/N)
set /p CHOICE="> "
if /i "!CHOICE!" == "S" (
    echo.
    echo 🚀 Iniciando launcher...
    start "" "cmd\launcher\launcher.exe"
    echo ✅ Launcher iniciado!
) else (
    echo ✅ Build finalizado. Teste quando desejar.
)

echo.
echo Pressione qualquer tecla para sair...
pause >nul 