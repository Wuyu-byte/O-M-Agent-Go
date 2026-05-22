
@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0start-all.ps1"
if errorlevel 1 (
    echo.
    echo Startup failed. Please check the error above.
)
pause
