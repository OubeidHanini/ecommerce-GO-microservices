#!/usr/bin/env bash
# Generates local development secrets. Nothing it writes is committed (see .gitignore).
#
#   .env                                  read by Docker Compose
#   deploy/overlays/dev/secrets/*.env     read by kustomize (kubectl apply -k deploy/overlays/dev)
#
# Existing values in .env are kept, so running it again never rotates a password under a
# database that was initialised with the old one. Delete .env to start from scratch.
set -euo pipefail
cd "$(dirname "$0")/.."

random_password() { LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32 || true; }

# 1. .env: copy the template and fill every empty value with a random password.
touch .env
while IFS= read -r line; do
  if [[ "$line" =~ ^([A-Z_]+)=(.*)$ ]]; then
    key="${BASH_REMATCH[1]}"
    grep -q "^$key=." .env && continue
    value="${BASH_REMATCH[2]}"
    [[ -z "$value" ]] && value="$(random_password)"
    echo "$key=$value" >> .env
  fi
done < .env.example
chmod 600 .env

# 2. Kubernetes dev secrets, from the same values: one file per Secret, so each
#    pod only gets the credentials it needs.
get() { sed -n "s/^$1=//p" .env; }
dir=deploy/overlays/dev/secrets
mkdir -p "$dir"
write() { printf '%s\n' "${@:2}" > "$dir/$1.env"; chmod 600 "$dir/$1.env"; }

write postgres-superuser   "password=$(get POSTGRES_PASSWORD)"
write user-service-db      "password=$(get USER_SERVICE_DB_PASSWORD)"
write product-service-db   "password=$(get PRODUCT_SERVICE_DB_PASSWORD)"
write order-service-db     "password=$(get ORDER_SERVICE_DB_PASSWORD)"
write rabbitmq             "username=$(get RABBITMQ_USER)" "password=$(get RABBITMQ_PASSWORD)"
write grafana-admin        "password=$(get GRAFANA_ADMIN_PASSWORD)"

echo "Secrets ready: .env and $dir/ (both git-ignored)."
