@echo off
REM BadgerSync Build Script for Windows
REM Build and management commands for the BadgerSync service

set BINARY_NAME=badgersync.exe
set BUILD_DIR=build

if "%1"=="clean" goto clean
if "%1"=="deps" goto deps
if "%1"=="test" goto test
if "%1"=="fmt" goto fmt
if "%1"=="run" goto run
if "%1"=="help" goto help
if "%1"=="build-all" goto build-all

REM Default: build for current platform
echo Building BadgerSync for Windows...
if not exist %BUILD_DIR% mkdir %BUILD_DIR%
go build -ldflags "-X main.Version=dev -X main.BuildTime=%date% %time%" -o %BUILD_DIR%\%BINARY_NAME% main.go
echo Build complete: %BUILD_DIR%\%BINARY_NAME%
goto end

:clean
echo Cleaning build artifacts...
if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
if exist %BINARY_NAME% del %BINARY_NAME%
echo Clean complete
goto end

:deps
echo Installing dependencies...
go mod download
go mod tidy
echo Dependencies installed
goto end

:test
echo Running tests...
go test -v ./...
echo Tests complete
goto end

:fmt
echo Formatting code...
go fmt ./...
echo Code formatting complete
goto end

:run
echo Running BadgerSync...
go run main.go
goto end

:build-all
echo Building BadgerSync for multiple platforms...
if not exist %BUILD_DIR% mkdir %BUILD_DIR%

REM Windows
go build -ldflags "-X main.Version=dev -X main.BuildTime=%date% %time%" -o %BUILD_DIR%\badgersync-windows-amd64.exe main.go
go build -ldflags "-X main.Version=dev -X main.BuildTime=%date% %time%" -o %BUILD_DIR%\badgersync-windows-arm64.exe main.go

REM Linux
set GOOS=linux
set GOARCH=amd64
go build -ldflags "-X main.Version=dev -X main.BuildTime=%date% %time%" -o %BUILD_DIR%\badgersync-linux-amd64 main.go
set GOARCH=arm64
go build -ldflags "-X main.Version=dev -X main.BuildTime=%date% %time%" -o %BUILD_DIR%\badgersync-linux-arm64 main.go

REM macOS
set GOOS=darwin
set GOARCH=amd64
go build -ldflags "-X main.Version=dev -X main.BuildTime=%date% %time%" -o %BUILD_DIR%\badgersync-darwin-amd64 main.go
set GOARCH=arm64
go build -ldflags "-X main.Version=dev -X main.BuildTime=%date% %time%" -o %BUILD_DIR%\badgersync-darwin-arm64 main.go

REM Reset to Windows
set GOOS=windows
set GOARCH=amd64

echo Multi-platform build complete in %BUILD_DIR%\
goto end

:help
echo BadgerSync Build Commands:
echo   build.bat         - Build for Windows
echo   build.bat clean   - Clean build artifacts
echo   build.bat deps    - Install dependencies
echo   build.bat test    - Run tests
echo   build.bat fmt     - Format code
echo   build.bat run     - Run the application
echo   build.bat build-all - Build for multiple platforms
echo   build.bat help    - Show this help
goto end

:end 