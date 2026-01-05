#!/bin/bash

# AI对话数据管理系统 - 数据库初始化脚本

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="${PROJECT_ROOT}/data"
DB_FILE="${DATA_DIR}/conversation.db"
SQL_FILE="${PROJECT_ROOT}/scripts/init_database.sql"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}AI对话数据管理系统 - 数据库初始化${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查 SQLite 是否安装
if ! command -v sqlite3 &> /dev/null; then
    echo -e "${RED}错误: sqlite3 未安装${NC}"
    echo "请先安装 SQLite3"
    exit 1
fi

# 显示 SQLite 版本
SQLITE_VERSION=$(sqlite3 --version | awk '{print $1}')
echo -e "${GREEN}SQLite 版本:${NC} ${SQLITE_VERSION}"
echo ""

# 创建数据目录
if [ ! -d "${DATA_DIR}" ]; then
    echo -e "${YELLOW}创建数据目录:${NC} ${DATA_DIR}"
    mkdir -p "${DATA_DIR}"
fi

# 创建图片目录（统一存放所有来源的图片）
IMAGES_DIR="${DATA_DIR}/images"
mkdir -p "${IMAGES_DIR}"
echo -e "${GREEN}创建图片目录:${NC} ${IMAGES_DIR}"

# 检查数据库文件是否已存在
if [ -f "${DB_FILE}" ]; then
    echo -e "${YELLOW}警告: 数据库文件已存在${NC}"
    read -p "是否要删除现有数据库并重新初始化? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}删除现有数据库...${NC}"
        rm -f "${DB_FILE}"
        rm -rf "${DB_FILE}-shm"
        rm -rf "${DB_FILE}-wal"
    else
        echo -e "${GREEN}保留现有数据库，退出${NC}"
        exit 0
    fi
fi

# 检查 SQL 文件是否存在
if [ ! -f "${SQL_FILE}" ]; then
    echo -e "${RED}错误: SQL 文件不存在${NC}"
    echo "文件路径: ${SQL_FILE}"
    exit 1
fi

# 初始化数据库
echo -e "${GREEN}正在初始化数据库...${NC}"
sqlite3 "${DB_FILE}" < "${SQL_FILE}"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ 数据库初始化成功!${NC}"
    echo ""
    echo -e "${GREEN}数据库位置:${NC} ${DB_FILE}"

    # 显示数据库信息
    echo ""
    echo -e "${GREEN}数据库表列表:${NC}"
    sqlite3 "${DB_FILE}" ".tables"

    echo ""
    echo -e "${GREEN}数据库大小:${NC}"
    du -h "${DB_FILE}"

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}初始化完成!${NC}"
    echo -e "${GREEN}========================================${NC}"
else
    echo -e "${RED}✗ 数据库初始化失败${NC}"
    exit 1
fi
