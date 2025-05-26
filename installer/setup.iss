[Setup]
; Informações básicas do app
AppId={{B8E8F8A0-1234-5678-9ABC-DEF012345678}
AppName=Milhões - Lottery Optimizer
AppVersion=1.0.21
AppVerName=Milhões - Lottery Optimizer 1.0.21
AppPublisher=Lottery Optimizer Team
AppPublisherURL=https://github.com/cccarv82/milhoes-desktop
AppSupportURL=https://github.com/cccarv82/milhoes-desktop/issues
AppUpdatesURL=https://github.com/cccarv82/milhoes-desktop/releases
DefaultDirName={autopf}\Milhoes
DefaultGroupName=Milhões - Lottery Optimizer
AllowNoIcons=yes
LicenseFile=
InfoBeforeFile=
InfoAfterFile=
OutputDir=build\installer
OutputBaseFilename=MilhoesSetup
; SetupIconFile=build\appicon.ico  ; Comentado até criarmos o ícone
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
PrivilegesRequiredOverridesAllowed=dialog
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64

; Configurações visuais (desabilitadas até criarmos as imagens)
; WizardImageFile=installer-banner.bmp
; WizardSmallImageFile=installer-icon.bmp

; Configurações de versionamento
VersionInfoVersion=1.0.21.0
VersionInfoCompany=Lottery Optimizer Team
VersionInfoDescription=Milhões - Lottery Optimizer
VersionInfoCopyright=Copyright (C) 2025 Lottery Optimizer Team
VersionInfoProductName=Milhões - Lottery Optimizer
VersionInfoProductVersion=1.0.21

[Languages]
Name: "brazilianportuguese"; MessagesFile: "compiler:Languages\BrazilianPortuguese.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "quicklaunchicon"; Description: "{cm:CreateQuickLaunchIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked; OnlyBelowVersion: 6.1
Name: "associatefiles"; Description: "Associar arquivos .lottery com Milhões"; GroupDescription: "Associações de arquivo:"; Flags: unchecked
Name: "addtopath"; Description: "Adicionar ao PATH do sistema (permite executar 'milhoes' no terminal)"; GroupDescription: "Opções avançadas:"; Flags: unchecked

[Files]
; Executável principal
Source: "..\build\bin\milhoes.exe"; DestDir: "{app}"; Flags: ignoreversion
; Arquivo de configuração de exemplo
Source: "..\config\lottery-optimizer.yaml.example"; DestDir: "{app}\config"; Flags: ignoreversion
; DLLs necessárias (se houver)
Source: "..\build\bin\*.dll"; DestDir: "{app}"; Flags: ignoreversion skipifsourcedoesntexist
; Documentação
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Milhões"; Filename: "{app}\milhoes.exe"
Name: "{group}\{cm:ProgramOnTheWeb,Milhões}"; Filename: "https://github.com/cccarv82/milhoes-desktop"
Name: "{group}\{cm:UninstallProgram,Milhões}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\Milhões"; Filename: "{app}\milhoes.exe"; Tasks: desktopicon
Name: "{userappdata}\Microsoft\Internet Explorer\Quick Launch\Milhões"; Filename: "{app}\milhoes.exe"; Tasks: quicklaunchicon

[Registry]
; Associações de arquivo (sistema)
Root: HKLM; Subkey: "Software\Classes\.lottery"; ValueType: string; ValueName: ""; ValueData: "MilhoesFile"; Flags: uninsdeletevalue; Tasks: associatefiles
Root: HKLM; Subkey: "Software\Classes\MilhoesFile"; ValueType: string; ValueName: ""; ValueData: "Arquivo de Estratégia Milhões"; Flags: uninsdeletekey; Tasks: associatefiles
Root: HKLM; Subkey: "Software\Classes\MilhoesFile\DefaultIcon"; ValueType: string; ValueName: ""; ValueData: "{app}\milhoes.exe,0"; Tasks: associatefiles
Root: HKLM; Subkey: "Software\Classes\MilhoesFile\shell\open\command"; ValueType: string; ValueName: ""; ValueData: """{app}\milhoes.exe"" ""%1"""; Tasks: associatefiles

; Chaves para auto-update (sistema)
Root: HKLM; Subkey: "Software\Milhoes"; ValueType: string; ValueName: "InstallPath"; ValueData: "{app}"; Flags: uninsdeletekey
Root: HKLM; Subkey: "Software\Milhoes"; ValueType: string; ValueName: "Version"; ValueData: "1.0.21"; Flags: uninsdeletekey

; Adicionar ao PATH do sistema (opcional)
Root: HKLM; Subkey: "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"; ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; Check: NeedsAddPath('{app}'); Tasks: addtopath

[Run]
Filename: "{app}\milhoes.exe"; Description: "{cm:LaunchProgram,Milhões}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{app}\config"
Type: filesandordirs; Name: "{app}\logs"
Type: filesandordirs; Name: "{app}\cache"

[Code]
// Função para verificar se o PATH já contém o diretório
function NeedsAddPath(Param: string): boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKEY_LOCAL_MACHINE,
    'SYSTEM\CurrentControlSet\Control\Session Manager\Environment',
    'Path', OrigPath)
  then begin
    Result := True;
    exit;
  end;
  // Verifica se o caminho já está no PATH
  Result := Pos(';' + Param + ';', ';' + OrigPath + ';') = 0;
end;

// Função para comparar versões
function CompareVersion(V1, V2: String): Integer;
var
  P, N1, N2: Integer;
begin
  Result := 0;
  while (Result = 0) and ((V1 <> '') or (V2 <> '')) do
  begin
    P := Pos('.', V1);
    if P > 0 then
    begin
      N1 := StrToIntDef(Copy(V1, 1, P - 1), 0);
      Delete(V1, 1, P);
    end
    else if V1 <> '' then
    begin
      N1 := StrToIntDef(V1, 0);
      V1 := '';
    end
    else
      N1 := 0;

    P := Pos('.', V2);
    if P > 0 then
    begin
      N2 := StrToIntDef(Copy(V2, 1, P - 1), 0);
      Delete(V2, 1, P);
    end
    else if V2 <> '' then
    begin
      N2 := StrToIntDef(V2, 0);
      V2 := '';
    end
    else
      N2 := 0;

    if N1 < N2 then
      Result := -1
    else if N1 > N2 then
      Result := 1;
  end;
end; 

// Função para verificar se uma versão mais nova já está instalada
function InitializeSetup(): Boolean;
var
  InstalledVersion: String;
  CurrentVersion: String;
begin
  Result := True;
  CurrentVersion := '1.0.21';
  
  if RegQueryStringValue(HKEY_LOCAL_MACHINE, 'Software\Milhoes', 'Version', InstalledVersion) then
  begin
    if CompareVersion(InstalledVersion, CurrentVersion) > 0 then
    begin
      if MsgBox('Uma versão mais recente (' + InstalledVersion + ') já está instalada. Deseja continuar?', 
                mbConfirmation, MB_YESNO) = IDNO then
        Result := False;
    end;
  end;
end; 