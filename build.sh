#!/bin/bash
go build -ldflags "-X main.commitHash=$(git rev-list -1 HEAD)$(test -n "$(git status --porcelain)" && echo '-dirty')" -trimpath -o build/ .
