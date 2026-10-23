#!/usr/bin/env bash

set -e

git config --global url."https://oauth2:$GITHUB_TOKEN@github.com".insteadOf https://github.com

git fetch --tags

tag_exists() {
    git show-ref --tags --verify --quiet "refs/tags/$1"
}

CURRENT_VERSION=$(grep -e  '^version' plugin.yaml | sed -e 's/^version: //')

if tag_exists "$CURRENT_VERSION"; then
    # The current version is already released, bump the patch version
    export NEXT_VERSION=$(echo ${CURRENT_VERSION} | awk -F. -v OFS=. '{$NF += 1 ; print}')
    if tag_exists "$NEXT_VERSION"; then
        echo "The tag $NEXT_VERSION already exists, exiting."
        exit 1
    fi
    sed -i -e "s/version: $CURRENT_VERSION/version: $NEXT_VERSION/" plugin.yaml
else
    # The version was set manually in plugin.yaml (e.g. a major or minor bump), release it as is
    export NEXT_VERSION=$CURRENT_VERSION
fi

echo "bump Version to $NEXT_VERSION"
echo "NEXT_VERSION=$NEXT_VERSION" >> $GITHUB_ENV
