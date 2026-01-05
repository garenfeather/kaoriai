#!/bin/bash

# Web Test - 快速启动脚本

echo "🚀 启动 Web 测试服务器..."
echo ""

# 杀死已存在的进程
echo "🔍 检查端口 8000..."
PID=$(lsof -ti:8000 2>/dev/null)
if [ -n "$PID" ]; then
    echo "⚠️  发现端口 8000 被占用 (PID: $PID)"
    echo "🔪 正在杀死进程..."
    kill -9 $PID 2>/dev/null
    sleep 1
    echo "✅ 进程已清理"
fi

# 检查是否安装 Python 3
if command -v /usr/local/bin/python3 &> /dev/null
then
    echo "✅ 使用 Python 3 启动服务器"
    echo "📍 访问地址: http://localhost:8000"
    echo "💡 按 Ctrl+C 停止服务器"
    echo ""
    /usr/local/bin/python3 -m http.server 8000
elif command -v python3 &> /dev/null
then
    echo "✅ 使用 Python 3 启动服务器"
    echo "📍 访问地址: http://localhost:8000"
    echo "💡 按 Ctrl+C 停止服务器"
    echo ""
    python3 -m http.server 8000
else
    echo "❌ 未找到 Python 3"
    echo "请安装 Python 3 或使用其他方式启动服务器"
    exit 1
fi
