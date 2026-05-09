#!/bin/bash
# AlphaPulse auto-restart watcher
# Monitors Go source files for changes, rebuilds and restarts on change

WATCH_DIR="/home/finn/code/AlphaPulse"
LOG_FILE="/tmp/alphapulse_restart.log"

cd "$WATCH_DIR"

# Get initial checksum of all .go files
get_checksum() {
    find internal cmd -name '*.go' -type f -exec md5sum {} + 2>/dev/null | sort | md5sum | cut -d' ' -f1
}

LAST_CHECKSUM=$(get_checksum)

while true; do
    sleep 10
    CURRENT_CHECKSUM=$(get_checksum)
    
    if [ "$CURRENT_CHECKSUM" != "$LAST_CHECKSUM" ]; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] Go files changed, rebuilding..." >> "$LOG_FILE"
        
        # Rebuild
        if ! go build -o bin/server ./cmd/server 2>>"$LOG_FILE"; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] Build FAILED" >> "$LOG_FILE"
            LAST_CHECKSUM=$CURRENT_CHECKSUM
            continue
        fi
        
        # Kill old server
        lsof -ti :8080 | xargs kill 2>/dev/null
        sleep 1
        
        # Start new server
        export $(cat .env | xargs)
        nohup ./bin/server >> "$LOG_FILE" 2>&1 &
        
        sleep 2
        # Health check
        HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/auth/login -X POST -H 'Content-Type: application/json' -d '{"username":"admin","password":"admin123"}' 2>/dev/null)
        
        if [ "$HTTP_CODE" = "200" ]; then
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] Restart OK (HTTP $HTTP_CODE)" >> "$LOG_FILE"
        else
            echo "[$(date '+%Y-%m-%d %H:%M:%S')] Restart WARNING: HTTP $HTTP_CODE" >> "$LOG_FILE"
        fi
        
        LAST_CHECKSUM=$CURRENT_CHECKSUM
    fi
done
