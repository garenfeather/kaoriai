#!/bin/bash

# 数据库清理脚本
# 用法:
#   ./db_clear.sh rebuild                 # 重建整个数据库
#   ./db_clear.sh gpt                     # 清空 gpt 来源的数据
#   ./db_clear.sh claude                  # 清空 claude 来源的数据
#   ./db_clear.sh codex                   # 清空 codex 来源的数据
#   ./db_clear.sh -y rebuild              # 不询问直接重建
#   ./db_clear.sh -y gpt                  # 不询问直接清空 gpt

# 工作目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$PROJECT_ROOT/bin"

# 创建 bin 目录
mkdir -p "$BIN_DIR"

# 配置参数
DB_PATH="${DB_PATH:-data/conversations.db}"
AUTO_YES=false

# 解析参数
ARGS=()
for arg in "$@"; do
    if [ "$arg" = "-y" ] || [ "$arg" = "--yes" ]; then
        AUTO_YES=true
    else
        ARGS+=("$arg")
    fi
done

# 检查参数
if [ ${#ARGS[@]} -eq 0 ]; then
    echo "错误: 缺少参数"
    echo ""
    echo "用法:"
    echo "  $0 [-y] rebuild              # 重建整个数据库"
    echo "  $0 [-y] <source_type>        # 清空指定来源的数据"
    echo ""
    echo "选项:"
    echo "  -y, --yes                    # 自动确认，不询问"
    echo ""
    echo "示例:"
    echo "  $0 rebuild                   # 删除并重建数据库（需确认）"
    echo "  $0 -y rebuild                # 删除并重建数据库（不询问）"
    echo "  $0 gpt                       # 清空所有 gpt 来源的数据（需确认）"
    echo "  $0 -y claude                 # 清空所有 claude 来源的数据（不询问）"
    echo "  $0 codex                     # 清空所有 codex 来源的数据（需确认）"
    exit 1
fi

# 编译清理工具
echo "正在编译清理工具..."
cd "$PROJECT_ROOT/scripts/go"
go build -o "$BIN_DIR/db_clear" db_clear.go

if [ $? -ne 0 ]; then
    echo "编译失败!"
    exit 1
fi

echo "编译成功!"
echo ""

# 切换回项目根目录
cd "$PROJECT_ROOT"

# 执行清理操作
if [ "${ARGS[0]}" = "rebuild" ]; then
    # 重建数据库
    echo "==================================="
    echo "警告: 即将删除并重建整个数据库!"
    echo "数据库: $DB_PATH"
    echo "==================================="
    echo ""

    if [ "$AUTO_YES" = false ]; then
        read -p "确认操作? (输入 yes 继续): " confirm
        if [ "$confirm" != "yes" ]; then
            echo "操作已取消"
            exit 0
        fi
        echo ""
    else
        echo "自动确认模式: 继续执行"
        echo ""
    fi

    "$BIN_DIR/db_clear" -db "$DB_PATH" -mode rebuild
else
    # 清空指定来源的数据
    SOURCE_TYPE="${ARGS[0]}"

    echo "==================================="
    echo "警告: 即将清空来源为 '$SOURCE_TYPE' 的所有数据!"
    echo "数据库: $DB_PATH"
    echo "==================================="
    echo ""

    if [ "$AUTO_YES" = false ]; then
        read -p "确认操作? (输入 yes 继续): " confirm
        if [ "$confirm" != "yes" ]; then
            echo "操作已取消"
            exit 0
        fi
        echo ""
    else
        echo "自动确认模式: 继续执行"
        echo ""
    fi

    "$BIN_DIR/db_clear" -db "$DB_PATH" -mode clear-source -source "$SOURCE_TYPE"
fi
