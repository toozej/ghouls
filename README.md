# ghouls
Simple Go-based URL Bookmarking Service

## Endpoints
- /add
- /delete
- /list

## Add new URL to ghouls via cURL

With HTTP Basic Auth:
```bash
curl -X POST -u username:password -d "url=https://exampleurltoadd.com" http://ghouls-hostname-here/add
```

## changes required to use this repo
- generate a GitHub fine-grained access token (used in repo as "GITHUB_TOKEN" and in GitHub Actions Secrets as "GH_TOKEN") with the following read/write permissions
    - actions
    - code scanning alerts
    - commit statuses
    - contents
    - dependabot alerts
    - dependabot secrets
    - deployments
    - environments
    - issues
    - pages
    - pull requests
    - secret scanning alerts
    - secrets
    - webhooks
    - workflows
- generate cosign keypair
    - `cosign generate-key-pair`
    - `mv cosign.key $REPO_NAME.key`
    - `mv cosign.pub $REPO_NAME.pub`
- ensure new repo has the following GitHub Actions Secrets and local shell environment variables stored in `./.env`
    - (`cp .env.sample .env` is a good place to start for local shell environment variables)
    - GH_TOKEN
    - GH_GHCR_TOKEN
    - DOCKERHUB_USERNAME
    - DOCKERHUB_TOKEN
    - QUAY_USERNAME
    - QUAY_TOKEN
    - SNYK_TOKEN
    - COSIGN_PRIVATE_KEY
    - COSIGN_PASSWORD
- set up new repository in quay.io web console
    - (DockerHub and GitHub Container Registry do this automatically on first push/publish)
    - name must match Git repo name
    - grant robot user with username stored in QUAY_USERNAME "write" permissions (your quay.io account should already have admin permissions)
- set built packages visibility in GitHub packages to public
    - navigate to https://github.com/users/$USERNAME/packages/container/$REPO/settings
    - scroll down to "Danger Zone"
    - change visibility to public

## changes required to update golang version
- run `./scripts/update_golang_version.sh $NEW_VERSION_GOES_HERE`
