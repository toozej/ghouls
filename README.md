# ghouls

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/toozej/ghouls)
[![Go Report Card](https://goreportcard.com/badge/github.com/toozej/ghouls)](https://goreportcard.com/report/github.com/toozej/ghouls)
![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/toozej/ghouls/cicd.yaml)
![Docker Pulls](https://img.shields.io/docker/pulls/toozej/ghouls)
![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/toozej/ghouls/total)


# ARCHIVED AS OF JUNE 2025


Simple Go-based URL Bookmarking Service

## Endpoints
- /add
- /delete
- /list
- /health

## Add new URL to ghouls via cURL
With HTTP Basic Auth:
```bash
curl -X POST -u username:password -d "url=https://exampleurltoadd.com" http://ghouls-hostname-here/add
```

## changes required to update golang version
- `make update-golang-version`
