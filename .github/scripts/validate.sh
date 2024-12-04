#!/usr/bin/env bash

set -e

git config --global url."https://oauth2:$GITHUB_TOKEN@github.com".insteadOf https://github.com

git fetch --tags

CHANGELOG_VERSION="$(awk '/##/ {print $2; exit}' CHANGELOG.md)"
if [ -z "$CHANGELOG_VERSION" ]; then
  echo "Changelog version is empty, exiting."
  exit 1
fi

echo "Changelog version is $CHANGELOG_VERSION"

if ! [[ "$CHANGELOG_VERSION" =~ ^[v]*[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Version $CHANGELOG_VERSION does not match expected format of MAJOR.MINOR.PATCH, exiting."
  exit 1
fi

if git show-ref --tags | grep -q "refs/tags/$CHANGELOG_VERSION"; then
    echo "The tag $CHANGELOG_VERSION already exists, exiting."
    exit 1
fi

echo "Version $CHANGELOG_VERSION successfully validated"
echo "IMAGEVERSION=$CHANGELOG_VERSION" >> $GITHUB_ENV
