#!/bin/bash
# AlphaPulse 重启脚本（后端 + 前端）
# 用法: ./restart.sh [--frontend-only] [--backend-only]

set -e

cd "$(dirname "$0")"

RESTART_BACKEND=true
RESTART_FRONTEND=true

for arg in "$@"; do
  case "$arg" in
    --frontend-only) RESTART_BACKEND=false ;;
    --backend-only)  RESTART_FRONTEND=false ;;
  esac
done

# =============================================
# 后端重启
# =============================================
if [ "$RESTART_BACKEND" = true ]; then

  echo "=== 停旧后端 ==="

  OLD_PID=$(lsof -ti:8080 2>/dev/null || true)
  if [ -z "$OLD_PID" ]; then
    OLD_PID=$(fuser 8080/tcp 2>/dev/null || true)
  fi

  if [ -n "$OLD_PID" ]; then
    echo "  Port 8080 被 PID: $OLD_PID 占用， stopping..."
    kill "$OLD_PID" 2>/dev/null
    sleep 1
    if [ -n "$(lsof -ti:8080 2>/dev/null || fuser 8080/tcp 2>/dev/null || true)" ]; then
      kill -9 "$OLD_PID" 2>/dev/null
      sleep 1
    fi
    echo "  ✅ 端口已释放"
  else
    echo "  Port 8080 无占用"
  fi

  echo ""
  echo "=== 重新编译后端 ==="
  go build -o server ./cmd/server
  echo "  ✅ 编译成功"

  echo ""
  echo "=== 启动新后端 ==="
  nohup ./server > /tmp/alphapulse_server.log 2>&1 &
  echo "  PID: $!"
  sleep 2
  if ss -tlnp | grep -q ':8080'; then
    echo "  ✅ 后端已启动 (port 8080)"
  else
    echo "  ⚠️  启动中，请稍后检查..."
  fi

fi

# =============================================
# 前端重启
# =============================================
if [ "$RESTART_FRONTEND" = true ]; then

  echo ""
  echo "=== 停旧前端 ==="

  OLD_FE_PID=$(lsof -ti:3000 2>/dev/null || true)
  if [ -z "$OLD_FE_PID" ]; then
    OLD_FE_PID=$(fuser 3000/tcp 2>/dev/null || true)
  fi

  if [ -n "$OLD_FE_PID" ]; then
    echo "  Port 3000 被 PID: $OLD_FE_PID 占用， stopping..."
    kill "$OLD_FE_PID" 2>/dev/null
    sleep 1
    if [ -n "$(lsof -ti:3000 2>/dev/null || fuser 3000/tcp 2>/dev/null || true)" ]; then
      kill -9 "$OLD_FE_PID" 2>/dev/null
      sleep 1
    fi
    echo "  ✅ 端口已释放"
  else
    echo "  Port 3000 无占用"
  fi

  echo ""
  echo "=== 重新编译前端 ==="
  cd web
  npm run build 2>&1 | tail -5
  echo "  ✅ 编译成功"

  echo ""
  echo "=== 启动新前端 ==="
  nohup npm run start > /tmp/alphapulse_frontend.log 2>&1 &
  echo "  PID: $!"
  sleep 3
  if curl -s -o /dev/null -w '%{http_code}' http://localhost:3000 2>/dev/null | grep -q 200; then
    echo "  ✅ 前端已启动 (port 3000)"
  else
    echo "  ⚠️  启动中，请稍后检查..."
  fi || true

  cd ..

fi

echo ""
echo "=== 完成 ==="
echo "  前端: http://localhost:3000"
echo "  后端: http://localhost:8080"
