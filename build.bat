@echo off

REM Windows Automation Assistant Build Script

echo Building Windows Automation Assistant...

REM Build the assistant as .exe
go build -o assistant.exe .

if %ERRORLEVEL% EQU 0 (
    echo Build successful!
    echo Run 'assistant.exe --help' to see available options
    echo See README.md for documentation
) else (
    echo Build failed!
    exit /b 1
)
