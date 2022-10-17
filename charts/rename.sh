#!/bin/bash
FILENAME=$(basename "$1")
NEW_FILENAME="${FILENAME//$2/$3}"
NEW_PATH="$(dirname $1)/$NEW_FILENAME"
echo "Moving $1 -> $NEW_PATH"
mv "$1" "$NEW_PATH"
