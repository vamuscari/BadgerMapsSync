#!/bin/bash

# This script cross-compiles the application for macOS, Linux, and Windows
# using the Fyne CLI, and places the binaries in the build directory.
# It assumes you have the necessary cross-compilation toolchains installed.
# For Windows: mingw-w64 (e.g., `brew install mingw-w64`)
# For Linux: a Linux GCC toolchain (e.g., `brew install x86_64-unknown-linux-gnu`)

export CGO_ENABLED=1

usage() {
  cat <<EOF
Usage: $0 [-k] [-o targets]

  -k           Keep intermediate files after packaging.
  -o targets   Comma-separated list of operating systems to build (darwin, windows).
               Defaults to building both targets.
EOF
  exit "${1:-0}"
}

# Default options
CLEAN_UP=true
TARGETS=("darwin" "windows")

# Parse flags
while getopts ":ko:h" opt; do
  case ${opt} in
  k)
    CLEAN_UP=false
    echo "Running with -k, will not remove intermediate files."
    ;;
  o)
    if [ -z "${OPTARG}" ]; then
      echo "Option -o requires a comma-separated list of targets." 1>&2
      usage 1
    fi
    TARGETS=()
    IFS=',' read -ra REQUESTED_TARGETS <<< "${OPTARG}"
    for raw_target in "${REQUESTED_TARGETS[@]}"; do
      target=$(printf '%s' "${raw_target}" | tr '[:upper:]' '[:lower:]')
      case ${target} in
      darwin|mac|macos)
        TARGETS+=("darwin")
        ;;
      windows|win)
        TARGETS+=("windows")
        ;;
      *)
        echo "Unsupported target \"${raw_target}\"." 1>&2
        usage 1
        ;;
      esac
    done
    ;;
  h)
    usage 0
    ;;
  :)
    echo "Option -${OPTARG} requires an argument." 1>&2
    usage 1
    ;;
  \?)
    echo "Invalid option: -${OPTARG}" 1>&2
    usage 1
    ;;
  esac
done

if [ ${#TARGETS[@]} -eq 0 ]; then
  echo "No build targets specified." 1>&2
  usage 1
fi

TARGET_DARWIN=false
TARGET_WINDOWS=false
for target in "${TARGETS[@]}"; do
  case ${target} in
  darwin)
    TARGET_DARWIN=true
    ;;
  windows)
    TARGET_WINDOWS=true
    ;;
  esac
done

# Exit on error
set -e

# Clean the build directory
if [ "$CLEAN_UP" = true ]; then
  echo "Cleaning build directory..."
  rm -rf dist/*
  rm -rf BadgerMapsSync
  if [ "$TARGET_DARWIN" = true ]; then
    rm -rf BadgerMapsSync.app
  fi
  if [ "$TARGET_WINDOWS" = true ]; then
    rm -f BadgerMapsSync.exe
    rm -rf dist/windows_temp
  fi
fi

mkdir -p dist

if [ "$TARGET_DARWIN" = true ]; then
  # Compile for macOS
  echo "Compiling for macOS..."
  fyne package -os darwin -release
  echo "Compressing macOS app..."
  zip -r dist/BadgerMapsSync_macOS.zip BadgerMapsSync.app
  if [ "$CLEAN_UP" = true ]; then
    rm -rf BadgerMapsSync.app
  fi
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

if [ "$TARGET_WINDOWS" = true ]; then
  # Compile for Windows
  # NOTE: Requires a Windows cross-compiler like mingw-w64
  echo "Compiling for Windows..."
  env CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
    go build -o BadgerMapsSync.exe -ldflags="-H windowsgui -s -w" -tags release

  # This command embeds the icon and metadata from FyneApp.toml into the .exe
  env CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
    fyne package --os windows --executable BadgerMapsSync.exe -release

  # Create a temporary directory for packaging the final zip
  echo "Packaging Windows build with DLL..."
  TEMP_DIR="dist/windows_temp"
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
fi
echo "Build complete."
