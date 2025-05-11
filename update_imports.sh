#!/usr/bin/env bash

# This script updates Go import paths in *.go files within the current
# directory and its subdirectories. It's designed to change paths
# from "satcom-code/..." to "github.com/yackko/satcom-code/...".

# Exit immediately if a command exits with a non-zero status.
set -e

OLD_MODULE_PREFIX_IN_IMPORTS="satcom-code/"
NEW_MODULE_PATH_PREFIX="github.com/yackko/satcom-code/"

echo "This script will attempt to update Go import paths in all .go files"
echo "in the current directory tree."
echo "It will replace import path prefixes from:"
echo "  \"${OLD_MODULE_PREFIX_IN_IMPORTS}..."
echo "to:"
echo "  \"${NEW_MODULE_PATH_PREFIX}..."
echo ""
echo "Backups of modified files will be created with a '.original_backup' extension."
echo "Current directory: $(pwd)"
echo ""

read -p "Are you sure you want to proceed? (y/N): " confirmation

if [[ ! "$confirmation" =~ ^([yY][eE][sS]|[yY])$ ]]; then
  echo "Operation aborted by the user."
  exit 1
fi

echo ""
echo "Starting import path update..."

# Find all .go files and process them.
# Using -print0 and read -r -d $'\0' to correctly handle filenames with spaces or special characters.
find . -type f -name "*.go" -print0 | while IFS= read -r -d $'\0' gofile; do
  echo "Processing file: $gofile"

  # Use sed to replace the import path prefix.
  # -i'.original_backup' creates a backup of the original file before modification.
  # We use '#' as the delimiter for sed's 's' command to avoid conflicts with slashes in the paths.
  # The pattern specifically looks for the prefix inside double quotes to target import paths.
  sed -i'.original_backup' "s#\"${OLD_MODULE_PREFIX_IN_IMPORTS}#\"${NEW_MODULE_PATH_PREFIX}#g" "$gofile"
done

echo ""
echo "Import path update process completed."
echo "Backup files have been created with the .original_backup extension."
echo "---------------------------------------------------------------------"
echo "IMPORTANT NEXT STEPS:"
echo "1. Review the changes carefully. You can use 'git diff' if your project is under Git version control."
echo "   For example:"
echo "   git diff -- '*.go'"
echo "2. If the changes are correct, you can commit them."
echo "3. After confirming, you may remove the backup files. Example command (run from project root):"
echo "   find . -type f -name '*.go.original_backup' -delete"
echo "4. Run 'go mod tidy' in your project root to update go.mod and go.sum based on the new import paths."
echo "   go mod tidy"
echo "5. Try building your project again from the project root:"
echo "   go build -o satcli ./cmd/satcli"
echo "---------------------------------------------------------------------"

exit 0
