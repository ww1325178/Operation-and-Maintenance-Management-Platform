@echo off
cd /d %~dp0

if not exist "agent.exe" (
    echo [ERROR] agent.exe not found in this folder.
    echo Put agent.exe and install.bat in the same folder.
    pause
    exit /b 1
)

set TOKEN=45D9A76102BEF3C8
set /p ADDR=Server address (ws://IP:port/ws/agent):

if "%ADDR%"=="" (
    echo [ERROR] Empty server address.
    pause
    exit /b 1
)

agent.exe -install -server "%ADDR%" -token "%TOKEN%"
echo.
echo Done. If you did not see a popup, check C:\ProgramData\opsagent\agent.log
pause
