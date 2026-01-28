@echo off
setlocal

rem Find WinLibs/MinGW
for %%d in (
    "%LOCALAPPDATA%\Microsoft\WinGet\Packages\BrechtSanders.WinLibs.POSIX.UCRT_Microsoft.Winget.Source_8wekyb3d8bbwe\mingw64\bin"
    "C:\WinLibs\mingw64\bin"
    "C:\mingw64\bin"
    "C:\msys64\mingw64\bin"
    "C:\Program Files\WinLibs\mingw64\bin"
    "%LOCALAPPDATA%\WinLibs\mingw64\bin"
    "%LOCALAPPDATA%\Programs\mingw64\bin"
) do (
    if exist "%%~d\gcc.exe" (
        set "MINGW_PATH=%%~d"
        goto :found_mingw
    )
)

echo MinGW/GCC not found. Please install WinLibs via:
echo   winget install -e --id=BrechtSanders.WinLibs.POSIX.UCRT
exit /b 1

:found_mingw
echo Found MinGW at: %MINGW_PATH%
set "PATH=%MINGW_PATH%;%PATH%"
set CGO_ENABLED=1

cd /d "%~dp0"

rem Create build directories if they don't exist
if not exist "build\windows" mkdir build\windows
if not exist "build\linux" mkdir build\linux

rem Build Windows versions
echo Building Windows binaries...
set GOOS=windows
set GOARCH=amd64

go build -o build\windows\bazzite-devkit.exe ./cmd/bazzite-devkit
if %ERRORLEVEL% NEQ 0 (
    echo Build failed: bazzite-devkit (windows)
    exit /b 1
)

cd steam-shortcut-manager
go build -o ..\build\windows\steam-shortcut-manager.exe .
if %ERRORLEVEL% NEQ 0 (
    echo Build failed: steam-shortcut-manager (windows)
    exit /b 1
)
cd ..

rem Build Linux versions
echo Building Linux binaries...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

go build -o build\linux\bazzite-devkit ./cmd/bazzite-devkit
if %ERRORLEVEL% NEQ 0 (
    echo Build failed: bazzite-devkit (linux)
    exit /b 1
)

cd steam-shortcut-manager
go build -o ..\build\linux\steam-shortcut-manager .
if %ERRORLEVEL% NEQ 0 (
    echo Build failed: steam-shortcut-manager (linux)
    exit /b 1
)
cd ..

echo.
echo Build successful!
echo   build\windows\bazzite-devkit.exe
echo   build\windows\steam-shortcut-manager.exe
echo   build\linux\bazzite-devkit
echo   build\linux\steam-shortcut-manager
