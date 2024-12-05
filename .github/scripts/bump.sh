#!/usr/bin/env bash

set -e

git config --global url."https://oauth2:$GITHUB_TOKEN@github.com".insteadOf https://github.com

git fetch --tags

CURRENT_VERSION=$(grep -e  '^version' plugin.yaml | sed -e 's/^version: //')
NEXT_VERSION=$(echo ${CURRENT_VERSION} | awk -F. -v OFS=. '{$NF += 1 ; print}')
if git show-ref --tags | grep -q "refs/tags/$NEXT_VERSION"; then
    echo "The tag $NEXT_VERSION already exists, exiting."
    exit 1
fi

sed -i -e "s/version: $CURRENT_VERSION/version: $CURRENT_VERSION/" plugin.yaml

echo "bump Version to $NEXT_VERSION"
echo "VERSION=$NEXT_VERSION" >> $GITHUB_ENV
