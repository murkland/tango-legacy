#!/bin/bash
set -euo pipefail
mkdir -p build
go build -o build .
DYLD_LIBRARY_PATH=external/mgba/build ./build/bbn6 "$@"
