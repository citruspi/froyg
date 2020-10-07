#!/bin/bash

TYPE=${1:-""}
ARCH=${2:-""}
BIN=${3:-""}

OUT=""

case $TYPE in
  deb)
    case $ARCH in
      x86_64 | amd64)
        OUT="packages/froyg-amd64.deb"
        ARCH="x86_64"
        ;;
      i386)
        OUT="packages/froyg-386.deb"
      *)
        echo "Unsupported arch: $ARCH"
        exit 1
        ;;
    esac
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

[[ -z "$TYPE" ]] && { echo "No package type provided" ; exit 1; }
[[ -z "$ARCH" ]] && { echo "No package architecture provided" ; exit 1; }
[[ -z "$BIN" ]] && { echo "No binary provided" ; exit 1; }
[[ -z "$OUT" ]] && { echo "Failed to determine package name" ; exit 1; }

fpm -s dir -t $TYPE -n froyg -p $OUT \
  -v $CI_COMMIT_REF_SLUG --iteration $CI_COMMIT_SHORT_SHA \
  -a $ARCH --license "Public Domain" \
  -m "Mihir Singh (@citruspi)" --url "https://src.doom.fm/citruspi/froyg" \
  --description "Multi-region, multi-bucket HTTP Gateway for S3 Objects" \
  $BIN=/usr/bin/froyg ||  { echo 'Failed to build package' ; exit 1; }

echo "Packaged $(echo $OUT | awk -F '/' '{print $2}')"

case $TYPE in
  deb)
    dpkg -c $OUT
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac
