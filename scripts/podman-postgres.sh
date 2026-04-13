#!/bin/bash

# Script to run PostgreSQL with Podman

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CONTAINER_NAME="dal-postgres"
POSTGRES_IMAGE="docker.io/library/postgres:15-alpine"
POSTGRES_DB="crud_db"
POSTGRES_USER="dal_user"
POSTGRES_PASSWORD="dal_pass"
HOST_PORT="5432"

# Function to check if container exists
container_exists() {
    podman ps -a --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"
}

# Function to check if container is running
container_running() {
    podman ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"
}

# Start PostgreSQL
start_postgres() {
    echo -e "${YELLOW}Starting PostgreSQL with Podman...${NC}"

    if container_exists; then
        if container_running; then
            echo -e "${GREEN}PostgreSQL container is already running!${NC}"
        else
            echo "Starting existing container..."
            podman start ${CONTAINER_NAME}
            echo -e "${GREEN}PostgreSQL container started!${NC}"
        fi
    else
        echo "Creating and starting new container..."

        # Get absolute path to init.sql
        SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
        INIT_SQL="${SCRIPT_DIR}/init.sql"

        podman run -d \
            --name ${CONTAINER_NAME} \
            -e POSTGRES_DB=${POSTGRES_DB} \
            -e POSTGRES_USER=${POSTGRES_USER} \
            -e POSTGRES_PASSWORD=${POSTGRES_PASSWORD} \
            -p ${HOST_PORT}:5432 \
            -v postgres-data:/var/lib/postgresql/data:Z \
            -v ${INIT_SQL}:/docker-entrypoint-initdb.d/init.sql:Z,ro \
            ${POSTGRES_IMAGE}

        echo "Waiting for PostgreSQL to be ready..."
        sleep 5

        # Check if PostgreSQL is ready
        for i in {1..10}; do
            if podman exec ${CONTAINER_NAME} pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB} &>/dev/null; then
                echo -e "${GREEN}PostgreSQL is ready!${NC}"
                echo -e "${GREEN}Connection string: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${HOST_PORT}/${POSTGRES_DB}?sslmode=disable${NC}"
                return 0
            fi
            echo "Waiting for PostgreSQL to be ready... (attempt $i/10)"
            sleep 2
        done

        echo -e "${RED}PostgreSQL failed to start properly${NC}"
        return 1
    fi
}

# Stop PostgreSQL
stop_postgres() {
    echo -e "${YELLOW}Stopping PostgreSQL...${NC}"
    if container_exists; then
        podman stop ${CONTAINER_NAME}
        echo -e "${GREEN}PostgreSQL stopped!${NC}"
    else
        echo "PostgreSQL container does not exist"
    fi
}

# Remove PostgreSQL container and volume
reset_postgres() {
    echo -e "${YELLOW}Resetting PostgreSQL...${NC}"
    if container_exists; then
        podman stop ${CONTAINER_NAME} 2>/dev/null
        podman rm ${CONTAINER_NAME}
    fi
    podman volume rm postgres-data 2>/dev/null
    echo -e "${GREEN}PostgreSQL reset complete!${NC}"
}

# Show PostgreSQL logs
show_logs() {
    if container_exists; then
        podman logs ${CONTAINER_NAME}
    else
        echo "PostgreSQL container does not exist"
    fi
}

# Show PostgreSQL status
show_status() {
    if container_running; then
        echo -e "${GREEN}PostgreSQL is running${NC}"
        echo "Container: ${CONTAINER_NAME}"
        echo "Port: ${HOST_PORT}"
        echo "Database: ${POSTGRES_DB}"
        echo "User: ${POSTGRES_USER}"
    else
        if container_exists; then
            echo -e "${YELLOW}PostgreSQL container exists but is not running${NC}"
        else
            echo -e "${RED}PostgreSQL container does not exist${NC}"
        fi
    fi
}

# Main script logic
case "$1" in
    start)
        start_postgres
        ;;
    stop)
        stop_postgres
        ;;
    reset)
        reset_postgres
        ;;
    logs)
        show_logs
        ;;
    status)
        show_status
        ;;
    *)
        echo "Usage: $0 {start|stop|reset|logs|status}"
        echo ""
        echo "Commands:"
        echo "  start  - Start PostgreSQL container"
        echo "  stop   - Stop PostgreSQL container"
        echo "  reset  - Remove container and volume (fresh start)"
        echo "  logs   - Show PostgreSQL logs"
        echo "  status - Show PostgreSQL status"
        exit 1
        ;;
esac