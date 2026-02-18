#!/bin/bash
## refresh is used after reverting migrations and deleting the migration files

cd ..

case "$1" in
"revert")
    atlas migrate down 1 --env app --url "sqlite://dbs/app.db"
    if [ $? -eq 0 ]; then
        echo "✓ Successfully reverted"
    else
        echo "✗ Failed to revert"
    fi
    ;;
"refresh")
    atlas migrate hash --env app
    echo "✓ Successfully refreshed"
    ;;
"apply")
    atlas migrate apply --env app --url "sqlite://dbs/app.db"
    if [ $? -eq 0 ]; then
        echo "✓ Successfully migrated: $db_file"
    else
        echo "✗ Failed to migrate: $db_file"
    fi
    ;;
"add")
    atlas migrate diff --env app
    ;;
*)
    echo "$1 not a command"
    echo "Commands:"
    echo "  add"
    echo "  apply"
    ;;
esac
