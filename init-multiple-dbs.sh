#!/bin/bash

set -e
set -u

# Wait for PostgreSQL to be ready
until pg_isready -U "$POSTGRES_USER"; do
  echo "Waiting for PostgreSQL to be ready..."
  sleep 2
done

function create_user_and_database() {
	local database=$1
	local user=$2
	local password=$3
	echo "  Creating user and database '$database'"
	psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
	    CREATE USER $user WITH PASSWORD '$password';
	    CREATE DATABASE $database;
	    GRANT ALL PRIVILEGES ON DATABASE $database TO $user;
	    \c $database
	    GRANT ALL ON SCHEMA public TO $user;
	    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $user;
	    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $user;
EOSQL
}

if [ -n "$POSTGRES_MULTIPLE_DATABASES" ]; then
	echo "Multiple database creation requested: $POSTGRES_MULTIPLE_DATABASES"
	for db in $(echo $POSTGRES_MULTIPLE_DATABASES | tr ',' ' '); do
		case $db in
			news_db)
				create_user_and_database $db news_user news_password
				# Apply init.sql to news_db
				psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d news_db -f /docker-entrypoint-initdb.d/init.sql
				# Grant ownership of all tables to news_user
				psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d news_db <<-EOSQL
				    ALTER TABLE sources OWNER TO news_user;
				    ALTER TABLE rss_feeds OWNER TO news_user;
				    ALTER TABLE news OWNER TO news_user;
				    ALTER SEQUENCE sources_id_seq OWNER TO news_user;
				    ALTER SEQUENCE rss_feeds_id_seq OWNER TO news_user;
				    ALTER SEQUENCE news_id_seq OWNER TO news_user;
EOSQL
				;;
			comments_db)
				create_user_and_database $db comments_user comments_password
				;;
			*)
				echo "Unknown database: $db"
				;;
		esac
	done
	echo "Multiple databases created"
fi 