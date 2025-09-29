#!/bin/bash

# build-releases.sh 6.1.1

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi
VERSION=$1

mkdir -p releases

OSES=("linux" "freebsd" "openbsd" "darwin")
ARCHS=("amd64" "arm64")

for os in "${OSES[@]}"; do
  for arch in "${ARCHS[@]}"; do
    output_name="antibody-${os}-${arch}"
    
    if [ "$os" == "darwin" ] && [ "$arch" == "amd64" ]; then
      archive_name="antibody_${VERSION}_Darwin_x86_64.tar.gz"
    else
      archive_name="antibody_${VERSION}_$(tr 'a-z' 'A-Z' <<< ${os:0:1})${os:1}_${arch}.tar.gz"
    fi

    echo "Building ${output_name}..."
    CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build -ldflags="-s -w -X 'main.version=${VERSION}'" -tags netgo -o ${output_name} main.go
    echo "Packaging ${archive_name}..."
    tar -czf ${archive_name} ${output_name}
    mv ${archive_name} releases/
    rm ${output_name}
    
    echo "Done."
    echo
  done
done

echo "All releases have been built and packaged in the 'releases' directory."
