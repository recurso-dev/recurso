#!/bin/bash
set -e

echo "Recurso: Starting Infrastructure..."

# 1. Check if Docker Daemon is running
if ! docker info > /dev/null 2>&1; then
  echo "❌ Error: Docker Daemon is not running."
  echo "👉 Please start Docker Desktop or the Docker service."
  exit 1
fi

# 2. Determine command (docker compose vs docker-compose)
if docker compose version > /dev/null 2>&1; then
  CMD="docker compose"
elif command -v docker-compose > /dev/null 2>&1; then
  CMD="docker-compose"
else
  echo "❌ Error: Neither 'docker compose' nor 'docker-compose' found."
  exit 1
fi

# 3. Start Services
echo "🐳 Using: $CMD"
$CMD up -d

echo "✅ Infrastructure is running."
echo "   - Postgres: :5432"
echo "   - TigerBeetle: :3000"
