#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

# Function to create a database if it doesn't exist
# Arguments: $1 = database_name, $2 = owner_username (optional, defaults to POSTGRES_USER)
create_database_if_not_exists() {
    local dbname="$1"
    local owner="${2:-$POSTGRES_USER}" # Use provided owner or default to POSTGRES_USER

    # Check if database exists
    if psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "postgres" -tAc "SELECT 1 FROM pg_database WHERE datname='$dbname'" | grep -q 1; then
        echo "Database '$dbname' already exists."
    else
        echo "Creating database '$dbname' with owner '$owner'..."
        # Create the database. We might need to create the user first if it's different from POSTGRES_USER
        # For simplicity, if using different users per DB, you'd add USER creation logic here too
        # or ensure POSTGRES_USER has rights to create DBs owned by others.
        psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "postgres" <<-EOSQL
            CREATE DATABASE "$dbname" OWNER "$owner";
EOSQL
        echo "Database '$dbname' created."
    fi
}

# Retrieve database names and users from environment variables or use defaults

DB_USER_NAME="${POSTGRES_DB_USER:-ay_user_db}"
DB_THREAD_NAME="${POSTGRES_DB_THREAD:-ay_thread_db}"
DB_MEDIA_NAME="${POSTGRES_DB_MEDIA:-ay_media_db}"

# Create the databases
create_database_if_not_exists "$DB_USER_NAME"
create_database_if_not_exists "$DB_THREAD_NAME"
create_database_if_not_exists "$DB_MEDIA_NAME"

echo "Multiple database initialization complete."