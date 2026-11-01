@echo off
setlocal
cd /d "%~dp0"

where go >nul 2>nul
if errorlevel 1 (
  echo Go SDK not found. Install from https://go.dev/dl/
  pause
  exit /b 1
)

echo [1/4] Downloading and checking dependencies...
go mod tidy
if errorlevel 1 goto error

if exist "assets\app.ico" (
  echo [2/4] Embedding Windows application icon...
  go install github.com/tc-hib/go-winres@latest
  if errorlevel 1 goto error
  go-winres simply --icon assets/app.ico --manifest gui
  if errorlevel 1 goto error
) else (
  echo [2/4] No assets\app.ico found; using the default executable icon.
)

echo [3/4] Building Emdee Viewer...
set CGO_ENABLED=0
go build -trimpath -ldflags="-s -w -H=windowsgui" -o EmdeeViewer.exe .
if errorlevel 1 goto error

echo [4/4] Build complete.
echo.
echo BUILD SUCCESSFUL: %CD%\EmdeeViewer.exe
echo.
pause
exit /b 0

:error
echo.
echo BUILD FAILED. Check the error above.
pause
exit /b 1
