#!/usr/bin/env bash

# Version of ninja to build -- can be any git revision
VERSION="v1.8.0"

set -ev

SCRIPT_HASH=$(sha1sum ${BASH_SOURCE[0]} | awk '{print $1}')

cd $TRAVIS_WORK_DIR
if [[ -d ninjabin && "$SCRIPT_HASH" == "$(cat ninjabin/script_hash)" ]]; then
    exit 0
fi

# Please remember Travis HOME is clean each build we only store ninjabin in cache
git clone https://github.com/martine/ninja
cd ninja
./configure.py --bootstrap

mkdir -p ../ninjabin
rm -f ../ninjabin/ninja
echo -n $SCRIPT_HASH >../ninjabin/script_hash
mv ninja ../ninjabin/
