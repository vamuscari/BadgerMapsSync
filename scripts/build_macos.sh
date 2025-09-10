#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# Define application variables
APP_NAME="BadgerMaps.app"
APP_PATH="$APP_NAME"
EXECUTABLE_NAME="badgermaps"
ICON_SOURCE_PATH="assets/icon.png" # Assuming you have an icon here

# Clean up previous build
echo "Cleaning up previous build..."
rm -rf "$APP_PATH"

# Create the .app bundle structure
echo "Creating .app bundle structure..."
mkdir -p "$APP_PATH/Contents/MacOS"
mkdir -p "$APP_PATH/Contents/Resources"

# Build the Go application
echo "Building Go application..."
go build -o "$APP_PATH/Contents/MacOS/$EXECUTABLE_NAME"

# Create the Info.plist file
echo "Creating Info.plist..."
cat > "$APP_PATH/Contents/Info.plist" <<EOL
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>${EXECUTABLE_NAME}</string>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>CFBundleIdentifier</key>
    <string>com.example.badgermaps</string>
    <key>CFBundleName</key>
    <string>BadgerMaps</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
    <key>CFBundleShortVersionString</key>
    <string>1.0</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOL

# Convert and copy the icon
# macOS .app bundles use .icns files. A .png can be converted.
# We'll create an icon set and then pack it into a .icns file.
if [ -f "$ICON_SOURCE_PATH" ]; then
    echo "Processing application icon..."
    
    # Create a temporary rounded icon
    TEMP_ICON_PATH="/tmp/rounded_icon.png"
    echo "Rounding icon corners..."
    
    # Get image dimensions
    WIDTH=$(magick identify -format %w "$ICON_SOURCE_PATH")
    HEIGHT=$(magick identify -format %h "$ICON_SOURCE_PATH")
    
    # Define corner radius (e.g., 25% of the smallest dimension)
    RADIUS=$(awk -v w="$WIDTH" -v h="$HEIGHT" 'BEGIN { r = (w<h?w:h) * 0.25; print int(r) }')

    magick convert "$ICON_SOURCE_PATH" \
        \( +clone -alpha extract -draw "fill black polygon 0,0 0,$RADIUS $RADIUS,0 fill white circle $RADIUS,$RADIUS $RADIUS,0" \
        \( +clone -flip \) -compose Multiply -composite \
        \( +clone -flop \) -compose Multiply -composite \) \
        -alpha off -compose CopyOpacity -composite "$TEMP_ICON_PATH"

    ICONSET_DIR="icon.iconset"
    mkdir -p "$ICONSET_DIR"
    sips -z 16 16     "$TEMP_ICON_PATH" --out "$ICONSET_DIR/icon_16x16.png"
    sips -z 32 32     "$TEMP_ICON_PATH" --out "$ICONSET_DIR/icon_16x16@2x.png"
    sips -z 32 32     "$TEMP_ICON_PATH" --out "$ICONSET_DIR/icon_32x32.png"
    sips -z 64 64     "$TEMP_ICON_PATH" --out "$ICONSET_DIR/icon_32x32@2x.png"
    sips -z 128 128   "$TEMP_ICON_PATH" --out "$ICONSET_DIR/icon_128x128.png"
    sips -z 256 256   "$TEMP_ICON_PATH" --out "$ICONSET_DIR/icon_128x128@2x.png"
    sips -z 256 256   "$TEMP_ICON_PATH" --out "$ICONSET_DIR/icon_256x256.png"
    sips -z 512 512   "$TEMP_ICON_PATH" --out "$ICONSET_DIR/icon_256x256@2x.png"
    sips -z 512 512   "$TEMP_ICON_PATH" --out "$ICONSET_DIR/icon_512x512.png"
    sips -z 1024 1024 "$TEMP_ICON_PATH" --out "$ICONSET_DIR/icon_512x512@2x.png"
    
    iconutil -c icns "$ICONSET_DIR" -o "$APP_PATH/Contents/Resources/icon.icns"
    rm -rf "$ICONSET_DIR"
    rm -f "$TEMP_ICON_PATH"
else

    echo "Warning: Icon file not found at $ICON_SOURCE_PATH. Skipping icon creation."
fi

echo "Build complete. The application is located at $APP_PATH"
