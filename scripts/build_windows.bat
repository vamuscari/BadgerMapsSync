@echo off
echo "Building for Windows..."
set GOOS=windows
set GOARCH=amd64
go build -o badgermaps.exe
echo "Build complete. The application is located at badgermaps.exe"
