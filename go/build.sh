#!/bin/bash

BUILD_DIR="./builds"
BUILD_PACKAGE_DIR="./cmd/internal/dev"
BUILD_CONFIGS=(
    "android:arm64:Android ARM64"
    "linux:amd64:Linux AMD64"
)

echo "Cleaning $BUILD_DIR folder"
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"
echo "Builds folder cleaned"

echo "Creating INFO"
go run ./cmd/internal/pre-build
export $(grep -v '^#' "$BUILD_DIR/INFO.env" | xargs)

echo "PROGRAM_NAME=$PROGRAM_NAME"
echo "PROGRAM_VERSION=$PROGRAM_VERSION"

for config in "${BUILD_CONFIGS[@]}"; do
    IFS=':' read -r goos goarch description <<< "$config"
    
    echo "Building for $description ($goos/$goarch)"
    
    GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build -ldflags="-s -w" -o "$BUILD_DIR/$PROGRAM_NAME-v$PROGRAM_VERSION-$goos-$goarch" "$BUILD_PACKAGE_DIR"
    
    if [ $? -eq 0 ]; then
        echo "Success"
    else
        echo "Error building for $description"
        exit 1
    fi
done

echo "All builds completed"
ls -la "$BUILD_DIR"
