@echo off
setlocal

echo Building nucleus-helper for Windows...
go build -ldflags="-s -w" -o nucleus-helper.exe ./cmd
if %ERRORLEVEL% NEQ 0 (
    echo Build FAILED with exit code %ERRORLEVEL%
    exit /b %ERRORLEVEL%
)

echo Build complete: nucleus-helper.exe
for %%I in (nucleus-helper.exe) do echo Size: %%~zI bytes
endlocal
