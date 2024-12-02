#!/bin/bash

BIN_DIR=bin
BUILD_DIR=build
DIST_DIR=dist

arch_to_file() {
        case $1 in
                arm64) echo "aarch64" ;;
                arm) echo "arm" ;;
                386) echo "i386" ;;
                amd64) echo "x86-64" ;;
                *) echo "unknown" ;;
        esac
}

print_verbose() {
        if $VERBOSE; then
                if [[ -z $2 ]]; then
                        echo "$1"
                else
                        echo $1 "$@"
                fi
        fi
}