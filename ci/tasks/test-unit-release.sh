#!/usr/bin/env bash

set -e

source /etc/profile.d/chruby.sh
chruby 2.4

pushd bosh-dns-release/
  scripts/test-unit-release
popd
