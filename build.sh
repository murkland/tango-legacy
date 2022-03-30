#!/bin/bash
go build -ldflags "-X main.version=$(git describe --tags)$(test -n "$(git status --porcelain)" && echo '-dirty')" -trimpath -o build/ .
go build -ldflags "-X main.version=$(git describe --tags)$(test -n "$(git status --porcelain)" && echo '-dirty')" -trimpath -o build/ ./tools/replayview
