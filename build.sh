#!/bin/bash
# Multi-platform build script for SSH MCP Server

set -e

VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="build"
DIST_DIR="dist"

echo "🚀 Building SSH MCP Server v${VERSION}"
echo "=========================================="

# Clean previous builds
rm -rf ${BUILD_DIR} ${DIST_DIR}
mkdir -p ${BUILD_DIR} ${DIST_DIR}

# Build variables
APP_NAME="sshmcp"
REPO="github.com/Cigarliu/ssh-mcp-server"
LDFLAGS="-s -w -X ${REPO}/pkg/version.Version=${VERSION} -X ${REPO}/pkg/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# Supported platforms
PLATFORMS=(
    "windows/amd64"
    "windows/386"
    "windows/arm64"
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
)

echo ""
echo "📦 Building binaries for ${#PLATFORMS[@]} platforms..."
echo ""

for PLATFORM in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "${PLATFORM}"

    OUTPUT_NAME="${APP_NAME}-${GOOS}-${GOARCH}"
    if [ "${GOOS}" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    echo "Building ${OUTPUT_NAME}..."

    GOOS=${GOOS} GOARCH=${GOARCH} go build \
        -ldflags "${LDFLAGS}" \
        -o "${BUILD_DIR}/${OUTPUT_NAME}" \
        ./cmd/server

    # Create archive
    cd ${BUILD_DIR}
    if [ "${GOOS}" = "windows" ]; then
        zip "${DIST_DIR}/${OUTPUT_NAME}-${VERSION}.zip" "${OUTPUT_NAME}"
    else
        tar czf "${DIST_DIR}/${OUTPUT_NAME}-${VERSION}.tar.gz" "${OUTPUT_NAME}"
    fi
    cd ..

    # Generate checksums
    if [ "${GOOS}" = "windows" ]; then
        sha256sum "${DIST_DIR}/${OUTPUT_NAME}-${VERSION}.zip" >> "${DIST_DIR}/checksums.txt"
    else
        sha256sum "${DIST_DIR}/${OUTPUT_NAME}-${VERSION}.tar.gz" >> "${DIST_DIR}/checksums.txt"
    fi

    echo "✅ ${OUTPUT_NAME} built successfully"
done

echo ""
echo "✨ Build completed!"
echo "📁 Binaries are in: ${DIST_DIR}/"
echo ""
echo "📋 Generated files:"
ls -lh ${DIST_DIR}/
echo ""
echo "🔐 Checksums:"
cat ${DIST_DIR}/checksums.txt
echo ""
echo "🎯 Ready to create GitHub Release!"
echo ""
echo "To create a release:"
echo "1. git tag v${VERSION}"
echo "2. git push origin v${VERSION}"
echo "3. gh release create v${VERSION} --title 'v${VERSION}' --notes 'See CHANGELOG.md'"
echo "4. gh release upload v${VERSION} ${DIST_DIR}/*"
