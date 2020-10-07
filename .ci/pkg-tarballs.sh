#!/bin/bash

rm -rf tarballs && mkdir tarballs
rm -rf _build && mkdir _build

for FILE in out/*; do
  version=$(echo $FILE | awk -F '-' '{print $2,$3}' | xargs | tr ' ' '-')

  os=$(echo $version | awk -F '.' '{print $1}')
  arch=$(echo $version | awk -F '.' '{print $2}')

  label="froyg-$CI_COMMIT_REF_SLUG-$os-$arch"

  mkdir "_build/$label"

  cp LICENSE "_build/$label/LICENSE"
  cp $FILE "_build/$label/froyg"

  cd _build && tar cfJ "../tarballs/$label.tar.xz" $label && echo "packaged $label.tar.xz"
  cd ..
done