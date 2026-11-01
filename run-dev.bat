@echo off
cd /d "%~dp0"
go mod tidy
if errorlevel 1 pause & exit /b 1
go run .
pause
