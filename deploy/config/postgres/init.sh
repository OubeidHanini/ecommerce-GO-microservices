#!/bin/sh
# Runs once, on the first start of an empty data volume.
# One database and one owner role per service: a compromised service cannot read the others' data.
# Passwords come from mounted secret files (*_FILE), never from this repository.
set -eu

read_secret() { # read_secret VAR: value of the file named by VAR_FILE, else of VAR
  file_var="${1}_FILE"
  eval "file=\${$file_var:-}"
  if [ -n "$file" ]; then cat "$file"; else eval "printf '%s' \"\${$1:?$1 or $file_var is required}\""; fi
}

create_service_db() { # create_service_db DATABASE ROLE PASSWORD
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres \
    -v db="$1" -v role="$2" -v password="$3" <<'SQL'
CREATE ROLE :"role" LOGIN PASSWORD :'password';
CREATE DATABASE :"db" OWNER :"role";
REVOKE ALL ON DATABASE :"db" FROM PUBLIC;
SQL
}

create_service_db users_db    user_service    "$(read_secret USER_SERVICE_DB_PASSWORD)"
create_service_db products_db product_service "$(read_secret PRODUCT_SERVICE_DB_PASSWORD)"
create_service_db orders_db   order_service   "$(read_secret ORDER_SERVICE_DB_PASSWORD)"
