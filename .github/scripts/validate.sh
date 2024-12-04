#!/usr/bin/env bash

set -e

git config --global url."https://oauth2:$GITHUB_TOKEN@github.com".insteadOf https://github.com

git fetch --tags

PLUGIN_VERSION=$(grep -e  '^version' plugin.yaml | sed -e 's/^version: //')
if [[ $PLUGIN_VERSION != $CHANGELOG_VERSION ]]; then
    echo "found [$PLUGIN_VERSION] in plugin.yaml and mismatch with Change Log version [$CHANGELOG_VERSION]"
    exit 1
fi


if git show-ref --tags | grep -q "refs/tags/$PLUGIN_VERSION"; then
    echo "The tag $PLUGIN_VERSION already exists, exiting."
    exit 1
fi

echo "Version $PLUGIN_VERSION successfully validated"
echo "IMAGEVERSION=$PLUGIN_VERSION" >> $GITHUB_ENV
