#!/bin/bash

# This script cross-compiles the application for macOS, Linux, and Windows
# using the Fyne CLI, and places the binaries in the build directory.
# It assumes you have the necessary cross-compilation toolchains installed.
# For Windows: mingw-w64 (e.g., `brew install mingw-w64`)
# For Linux: a Linux GCC toolchain (e.g., `brew install x86_64-unknown-linux-gnu`)

export CGO_ENABLED=1

# Default to cleaning, but check for a -k flag to keep files
CLEAN_UP=true
while getopts "k" opt; do
  case ${opt} in
    k )
      CLEAN_UP=false
      echo "Running with -k, will not remove intermediate files."
      ;;
    \? )
      echo "Invalid option: $OPTARG" 1>&2
      exit 1
      ;;
  esac
done

# Exit on error
set -e

# Clean the build directory
if [ "$CLEAN_UP" = true ]; then
  echo "Cleaning build directory..."
  rm -rf build/*
  rm -rf BadgerMapsSync.app BadgerMapsSync
fi

# Compile for macOS
echo "Compiling for macOS..."
fyne package -os darwin -release
echo "Compressing macOS app..."
zip -r build/BadgerMapsSync_macOS.zip BadgerMapsSync.app
if [ "$CLEAN_UP" = true ]; then
  rm -rf BadgerMapsSync.app
fi

# Compile for Linux
# NOTE: Requires a Linux cross-compiler like x86_64-unknown-linux-gnu
# echo "Compiling for Linux..."
# export GOOS=linux
# export GOARCH=amd64
# export CC=x86_64-elf-gcc
# If your linux cross-compiler has a different name, set the CC env var.
# export CC=x86_64-linux-gnu-gcc
# fyne package -os linux -release
# tar -cJf build/BadgerMapsSync_linux_amd64.tar.xz BadgerMapsSync
# if [ "$CLEAN_UP" = true ]; then
#   rm -rf BadgerMapsSync
# fi
# unset GOOS
# unset GOARCH
# unset CC

# Compile for Windows
# NOTE: Requires a Windows cross-compiler like mingw-w64
echo "Compiling for Windows..."
export GOOS=windows
export GOARCH=amd64
export CC=x86_64-w64-mingw32-gcc

# Build the executable first with the correct linker flags
go build -o BadgerMapsSync.exe -ldflags="-H windowsgui -s -w" -tags release

# This command embeds the icon and metadata from FyneApp.toml into the .exe
fyne package --os windows --executable BadgerMapsSync.exe -release

# Create a temporary directory for packaging the final zip
echo "Packaging Windows build with DLL..."
TEMP_DIR="build/windows_temp"
mkdir -p "$TEMP_DIR"

# Copy the final executable and the DLL to the temp directory
cp BadgerMapsSync.exe "$TEMP_DIR/"
if [ -f "assets/opengl32.dll" ]; then
  echo "Including opengl32.dll..."
  cp "assets/opengl32.dll" "$TEMP_DIR/"
else
  echo "Warning: assets/opengl32.dll not found. Skipping."
fi

# Create the final zip file from the contents of the temp directory
(cd "$TEMP_DIR" && zip -q -r ../BadgerMapsSync_windows_amd64.zip .)

# Clean up intermediate files
if [ "$CLEAN_UP" = true ]; then
  rm -f BadgerMapsSync.exe
  rm -rf "$TEMP_DIR"
fi
unset GOOS
unset GOARCH
unset CC

echo "Build complete."
