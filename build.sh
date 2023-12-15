#!/bin/bash
set -e

SATH_HOME="$( cd -- "$( dirname -- "${BASH_SOURCE[0]:-$0}"; )" &> /dev/null && pwd 2> /dev/null; )";

mkdir -p $SATH_HOME/bin
cd $SATH_HOME/cmd/cli && go build && mv -f cli $SATH_HOME/bin/sath
cd $SATH_HOME/cmd/daemon && go build && mv -f daemon $SATH_HOME/bin/sath-daemon

echo successfully build sath executables