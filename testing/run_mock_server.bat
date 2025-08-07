@echo off

REM Build and run the mock BadgerMaps API server

echo Building mock server...
go build -o mock_server.exe mock_server.go

if %ERRORLEVEL% EQU 0 (
    echo Build successful!
    echo Starting mock server on port 8080...
    echo Press Ctrl+C to stop the server
    echo.
    mock_server.exe 8080
) else (
    echo Build failed!
    pause
    exit /b 1
) 