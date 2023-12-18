#!/bin/bash
set -e

SATH_HOME="$( cd -- "$( dirname -- "${BASH_SOURCE[0]:-$0}"; )" &> /dev/null && pwd 2> /dev/null; )"

pkg=github.com/sath-run/engine/constants
now=$(date +%s)
sha1=$(git rev-parse HEAD)
version=$(head -n1 VERSION)
ldflags="-X '$pkg.BuildTime=$now' -X '$pkg.Version=$version' -X '$pkg.Sha1Ver=$sha1'"

mkdir -p $SATH_HOME/build
cd $SATH_HOME/cli && go build -v -ldflags "$ldflags" && mv -f cli $SATH_HOME/build/sath
cd $SATH_HOME/engine && go build -v -ldflags "$ldflags" && mv -f engine $SATH_HOME/build/sath-engine

echo successfully build sath executables