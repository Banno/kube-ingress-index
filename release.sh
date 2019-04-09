#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

main () {
  local -r bump="${1:-patch}"

  local -r next_version="$(./bin/svu "${bump}")"

  if ! docker ps -q >/dev/null; then
    echo "Error: Docker daemon needs to be running"
    exit 1
  fi

  if ! type -t git-chglog >/dev/null; then
    echo "Error: git-chglog needs to be installed"
    exit 1
  fi

  git-chglog -o CHANGELOG.md --next-tag "${next_version}"
  git commit -am "Bump version to ${next_version} and update the changelog"
  git tag "${next_version}"
  ./bin/goreleaser --rm-dist --release-notes CHANGELOG.md
}

main "$@"
