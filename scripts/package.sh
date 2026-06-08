#!/bin/bash
# scripts/package.sh
# Cross-compiles the Go orchestrator application for multiple platforms,
# compresses them into .tar.gz and .zip packages, and packages debians if dpkg-deb is present.

set -e

VERSION=${1:-"0.0.1"}
SRC_DIR="qpi-interface"
DIST_DIR="bin/dist"

echo "=== Starting compilation and packaging for version: $VERSION ==="
mkdir -p "$DIST_DIR"

# Platforms to compile: OS ARCH EXTENSION ARCHIVE_FORMAT
PLATFORMS=(
    "darwin amd64 .app .tar.gz"
    "darwin arm64 .app .tar.gz"
    "windows amd64 .exe .zip"
    "windows arm64 .exe .zip"
    "linux amd64 "" .tar.gz"
    "linux arm64 "" .tar.gz"
)

for PLATFORM in "${PLATFORMS[@]}"; do
    read -r OS ARCH EXT ARCHIVE_FORMAT <<< "$PLATFORM"
    
    BIN_NAME="qpi"
    if [ "$OS" = "windows" ]; then
        BIN_NAME="qpi.exe"
    fi

    BUILD_DIR="bin/builds/qpi-${OS}-${ARCH}"
    mkdir -p "$BUILD_DIR"

    echo "Building for $OS/$ARCH..."
    GOOS=$OS GOARCH=$ARCH go build -ldflags="-X 'main.Version=v${VERSION#v}'" -C "$SRC_DIR" -o "../$BUILD_DIR/$BIN_NAME" .

    # Copy metadata or license files if needed
    cp README.md "$BUILD_DIR/" 2>/dev/null || true
    cp qpi.config.example.yml "$BUILD_DIR/" 2>/dev/null || true

    # Create archive package
    ARCHIVE_NAME="qpi-${VERSION}-${OS}-${ARCH}"
    if [ "$ARCHIVE_FORMAT" = ".zip" ]; then
        echo "Packaging $OS/$ARCH as ZIP..."
        (cd bin/builds && zip -r "../dist/${ARCHIVE_NAME}.zip" "qpi-${OS}-${ARCH}")
    else
        echo "Packaging $OS/$ARCH as TAR.GZ..."
        (cd bin/builds && tar -czf "../dist/${ARCHIVE_NAME}.tar.gz" "qpi-${OS}-${ARCH}")
    fi
done

# Build Debian packages for linux-amd64 and linux-arm64 if dpkg-deb is available
if command -v dpkg-deb &> /dev/null; then
    DEB_ARCHS=("amd64" "arm64")
    for ARCH in "${DEB_ARCHS[@]}"; do
        echo "Building Debian (.deb) package for linux/$ARCH..."
        DEB_DIR="bin/builds/deb/qpi_${VERSION}_${ARCH}"
        mkdir -p "$DEB_DIR/DEBIAN"
        mkdir -p "$DEB_DIR/usr/local/bin"

        # Copy compiled binary
        cp "bin/builds/qpi-linux-${ARCH}/qpi" "$DEB_DIR/usr/local/bin/"

        # Generate control file
        cat <<EOF > "$DEB_DIR/DEBIAN/control"
Package: qpi
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Maintainer: SopherApps <team.sopherapps@gmail.com>
Description: Quantum Processing Interface (QPI) Orchestrator
 QPI is a distributed quantum control stack architecture designed
 to control multiple Quantum Processing Units (QPUs).
EOF

        # Build Debian package
        dpkg-deb --build "$DEB_DIR" "$DIST_DIR/qpi_${VERSION}_${ARCH}.deb"
    done
else
    echo "dpkg-deb utility not found. Skipping Debian package creation (this is normal on macOS/Windows)."
fi

echo "=== Packaging completed successfully! Artifacts available in $DIST_DIR ==="
ls -l "$DIST_DIR"
