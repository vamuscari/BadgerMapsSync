#!/bin/bash

# This script cross-compiles the application for macOS, Linux, and Windows
# using fyne-cross, and places the binaries in the build directory.

# Exit on error
set -e

# Clean the build directory
echo "Cleaning build directory..."
rm -rf build/*

# go build -o build/badgermaps

# Compile for macOS
# Build with Fyne instead of fyne-cross until bug fixed
echo "Compiling for macOS..."
fyne package -os darwin -name BadgerMapsSync --app-id com.badgermapssync --app-build=1 -release -icon ./assets/icon.png -tags gui
echo "Compressing macOS app..."
zip -r BadgerMapsSync_macOS.zip BadgerMapsSync.app
mv BadgerMapsSync_macOS.zip build/

# Compile for Linux
echo "Compiling for Linux..."
fyne-cross linux -arch amd64 --app-id com.badgermapssync --app-build=1 -release -icon ./assets/icon.png -tags gui
mv fyne-cross/dist/linux-amd64/BadgerMapsSync.tar.xz build/BadgerMapsSync_linux_amd64.tar.xz

# Compile for Windows
echo "Compiling for Windows..."
fyne-cross windows -arch amd64 --app-id com.badgermapssync --app-build=1 -icon ./assets/icon.png -tags gui -ldflags '-H=windowsgui'

# Create a temporary directory for packaging
echo "Packaging Windows build with DLL..."
TEMP_DIR="build/windows_temp"
mkdir -p "$TEMP_DIR"

# Unzip the original package
unzip -q fyne-cross/dist/windows-amd64/BadgerMapsSync.exe.zip -d "$TEMP_DIR"

# Copy the DLL if it exists
if [ -f "assets/opengl32.dll" ]; then
    echo "Including opengl32.dll..."
    cp "assets/opengl32.dll" "$TEMP_DIR/"
else
    echo "Warning: assets/opengl32.dll not found. Skipping."
fi

# Re-zip the contents from the temp directory
(cd "$TEMP_DIR" && zip -q -r ../BadgerMapsSync_windows_amd64.exe.zip .)

# Clean up the temporary directory
rm -rf "$TEMP_DIR"

rm -rf ./fyne-cross

echo "Build complete."
