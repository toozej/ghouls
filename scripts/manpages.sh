#!/bin/sh
set -e
rm -rf manpages
mkdir manpages
go run ./cmd/ghouls/ man | gzip -c -9 >manpages/ghouls.1.gz
