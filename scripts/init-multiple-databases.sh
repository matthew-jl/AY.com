set -e

create_database_if_not_exists() {
    local dbname="$1"
    local owner="${2:-$POSTGRES_USER}"

    # Check if database exists
    if psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "postgres" -tAc "SELECT 1 FROM pg_database WHERE datname='$dbname'" | grep -q 1; then
        echo "Database '$dbname' already exists."
    else
        echo "Creating database '$dbname' with owner '$owner'..."
        psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "postgres" <<-EOSQL
            CREATE DATABASE "$dbname" OWNER "$owner";
EOSQL
        echo "Database '$dbname' created."
    fi
}


DB_USER_NAME="${POSTGRES_DB_USER:-ay_user_db}"
DB_THREAD_NAME="${POSTGRES_DB_THREAD:-ay_thread_db}"
DB_MEDIA_NAME="${POSTGRES_DB_MEDIA:-ay_media_db}"
DB_NOTIFICATION_NAME="${DB_NAME_NOTIFICATION:-ay_notification_db}"

create_database_if_not_exists "$DB_USER_NAME"
create_database_if_not_exists "$DB_THREAD_NAME"
create_database_if_not_exists "$DB_MEDIA_NAME"
create_database_if_not_exists "$DB_NOTIFICATION_NAME"

echo "Multiple database initialization complete."