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
fyne package -os darwin -name BadgerMapsSync --app-id com.badgermapssync --app-build=1 -release -icon ./assets/icon.png
echo "Compressing macOS app..."
zip -r BadgerMapsSync_macOS.zip BadgerMapsSync.app
mv BadgerMapsSync_macOS.zip build/

# Compile for Linux
echo "Compiling for Linux..."
fyne-cross linux -arch amd64 --app-id com.badgermapssync --app-build=1 -release -icon ./assets/icon.png
mv fyne-cross/dist/linux-amd64/BadgerMapsSync.tar.xz build/BadgerMapsSync_linux_amd64.tar.xz

# Compile for Windows
echo "Compiling for Windows..."
fyne-cross windows -arch amd64 --app-id com.badgermapssync --app-build=1 -icon ./assets/icon.png
mv fyne-cross/dist/windows-amd64/BadgerMapsSync.exe.zip build/BadgerMapsSync_windows_amd64.exe.zip

rm -rf ./fyne-cross

echo "Build complete."
