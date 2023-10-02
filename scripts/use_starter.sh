#!/usr/bin/env bash
set -Eeuo pipefail

OLD_PROJECT_NAME="golang-starter"
NEW_PROJECT_NAME="${1}"
GITHUB_USERNAME="${2:-toozej}"

GIT_REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${GIT_REPO_ROOT}"

# truncate existing CREDITS.md file and replace its contents with link to template repo's CREDITS.md file
echo -e "# Credits and Acknowledgements\n\n- https://raw.githubusercontent.com/toozej/golang-starter/main/CREDITS.md" > CREDITS.md

# remove golang-starter.pub key
rm -f ./golang-starter.pub

# update go module name
# shellcheck disable=SC2086
go mod edit -module=github.com/${GITHUB_USERNAME}/${NEW_PROJECT_NAME}

# move directories
mv "cmd/${OLD_PROJECT_NAME}" "cmd/${NEW_PROJECT_NAME}"

# rename from $OLD_PROJECT_NAME to $NEW_PROJECT_NAME
grep -rl --exclude-dir=.git/ ${OLD_PROJECT_NAME} . | xargs sed -i "" -e "s/${OLD_PROJECT_NAME}/${NEW_PROJECT_NAME}/g"

# show diff output so user can verify their changes
git diff
