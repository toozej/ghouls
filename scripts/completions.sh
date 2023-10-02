#!/bin/sh
set -e
rm -rf completions
mkdir completions
for sh in bash zsh fish; do
	go run ./cmd/golang-starter/ completion "$sh" >"completions/golang-starter.$sh"
done
