#!/usr/bin/env bash

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <semantic version>"
    exit 1
fi

OS_ARCH="darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm openbsd/386 openbsd/amd64 solaris/amd64 windows/386 windows/amd64"
ASSETS_DIR="./release-bin"

mkdir -p $ASSETS_DIR
rm -rf $ASSETS_DIR/*

gox -osarch "$OS_ARCH" -output "$ASSETS_DIR/{{.Dir}}_${VERSION}_{{.OS}}_{{.Arch}}"

for f in $ASSETS_DIR/*; do
    zip -q -j "$f.zip" $f
    rm -f $f
done

