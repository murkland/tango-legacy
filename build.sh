#!/bin/bash
go-winres make --product-version=git-tag
go build -ldflags "-X main.version=$(git describe --tags)$(test -n "$(git status --porcelain)" && echo '-dirty')" -trimpath -o build/ -ldflags "-H windowsgui" .
go build -ldflags "-X main.version=$(git describe --tags)$(test -n "$(git status --porcelain)" && echo '-dirty')" -trimpath -o build/ ./tools/replayview
