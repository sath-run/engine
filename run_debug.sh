#!/bin/bash

SATH_HOME="$( cd -- "$( dirname -- "${BASH_SOURCE[0]:-$0}"; )" &> /dev/null && pwd 2> /dev/null; )";
cd $SATH_HOME

export SATH_MODE=debug
export SATH_GRPC=localhost:50051

# run sath-cli, with all args forwarded
./build/sath ${@:1:999}