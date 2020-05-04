#!/bin/bash

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <semantic version>"
    exit 1
fi

OS_ARCH="darwin,amd64 freebsd,386 freebsd,amd64 freebsd,arm linux,386 linux,amd64 linux,arm openbsd,386 openbsd,amd64 solaris,amd64 windows,386 windows,amd64"
ASSETS_DIR="release-bin"

mkdir -p $ASSETS_DIR
rm -rf $ASSETS_DIR/*

for osarch in $OS_ARCH; do
    OS=$(echo "$osarch" | cut -d, -f1)
    ARCH=$(echo "$osarch" | cut -d, -f2)
    FILENAME="terraform-provider-kubernetes-alpha_${VERSION}_${OS}_${ARCH}"
    GOOS=$OS GOARCH=$ARCH go build -o $ASSETS_DIR/$FILENAME
    ZIPFILENAME="$ASSETS_DIR/$FILENAME.zip"
    zip -q $ZIPFILENAME $ASSETS_DIR/$FILENAME
    rm -f $ASSETS_DIR/$FILENAME
    echo $ZIPFILENAME
done

