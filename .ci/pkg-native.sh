#!/bin/bash

set -xe

case $TYPE in
  deb)
    case $ARCH in
      amd64 | x86_64)
        ARCH="x86_64"
        ;;
      386 | i386)
        ARCH="i386"
        ;;
      *)
        echo "Unsupported arch: $ARCH"
        exit 1
        ;;
    esac
    ;;
  rpm)
    case $ARCH in
      amd64 | x86_64)
        ARCH="x86_64"
        ;;
      386 | i386)
        ARCH="i386"
        ;;
      *)
        echo "Unsupported arch: $ARCH"
        exit 1
        ;;
    esac
    ;;
  *)
    echo "Unsupported package type: $TYPE"
    exit 1
    ;;
esac

[[ -z "$NAME" ]] && { echo "No package name provided"; exit 1; }
[[ -z "$TYPE" ]] && { echo "No package type provided"; exit 1; }
[[ -z "$ARCH" ]] && { echo "No package architecture provided"; exit 1; }
[[ -z "$@" ]] && { echo "No package files provided"; exit 1; }
[[ -z "$OUT" ]] && { echo "No package filename provided"; exit 1; }
[[ -z "$LICENSE" ]] && { echo "No package license provided"; exit 1; }
[[ -z "$DESCRIPTION" ]] && { echo "No package description provided"; exit 1; }
[[ -z "$URL" ]] && { echo "No package URL provided"; exit 1; }
[[ -z "$MAINTAINER" ]] && { echo "No package maintainer provided"; exit 1; }

VERSION=$(/usr/bin/version-from-ref)

fpm -s dir -t $TYPE -n $NAME -p "$OUT" \
  -v $VERSION --iteration $CI_COMMIT_SHORT_SHA \
  -a $ARCH --license "$LICENSE" \
  -m "$MAINTAINER" --url "$URL" \
  --description "$DESCRIPTION" \
  $@ ||  { echo 'Failed to build package' ; exit 1; }

echo "Packaged $(echo $OUT | awk -F '/' '{print $2}')"

case $TYPE in
  deb)
    dpkg -c $OUT
    ;;
  rpm)
    rpm -qlp $OUT
    ;;
  *)
    echo "Unsupported package type: $TYPE"
    exit 1
    ;;
esac
