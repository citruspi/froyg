#!/bin/bash

rm -rf "$TAR_DIR" && mkdir "$TAR_DIR"
rm -rf _build && mkdir _build

for FILE in "$BIN_DIR/*"; do
  version=$(echo $FILE | awk -F '-' '{print $2,$3}' | xargs | tr ' ' '-')

  os=$(echo $version | awk -F '.' '{print $1}')
  arch=$(echo $version | awk -F '.' '{print $2}')

  label="$NAME-$CI_COMMIT_REF_SLUG-$os-$arch"

  mkdir "_build/$label"

  cp LICENSE "_build/$label/LICENSE"
  cp $FILE "_build/$label/froyg"

  cd _build && tar cfJ "../$TAR_DIR/$label.tar.xz" $label && echo "packaged $label.tar.xz"
  cd ..
done