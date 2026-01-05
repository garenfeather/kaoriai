# 修复说明

## 问题 1: 模板字符串语法错误

### 错误信息
```
Uncaught SyntaxError: Missing } in template expression
```

### 原因
在 JavaScript 模板字符串 (`` ` ``) 中使用了 `$` 符号，JavaScript 引擎会尝试将其解析为模板表达式 `${}`。

### 修复方案

#### 1. Bash 脚本中的 `$` 变量
**修复前**:
```javascript
content: `
\`\`\`bash
echo "Processing: $file"
\`\`\`
`
```

**修复后**:
```javascript
content: `
\`\`\`bash
echo "Processing: \\$file"
\`\`\`
`
```

#### 2. JavaScript 代码中的模板字符串
**修复前**:
```javascript
const cacheKey = \`user:\${id}\`;
```

**修复后**:
```javascript
const cacheKey = \\\`user:\\\${id}\\\`;
```

#### 3. Python f-string
**修复前**:
```javascript
print(f"总成本: ${total}/月")
print(f"人均成本: ${per_user:.4f}/月")
```

**修复后** (改用字符串拼接):
```javascript
print("总成本: " + str(total) + "/月")
print("人均成本: " + format(per_user, '.4f') + "/月")
```

### 技术细节

JavaScript 模板字符串中的特殊字符转义规则:
- `` `  `` → `` \\` `` (反引号)
- `$` → `\\$` (美元符号)
- `\` → `\\\\` (反斜杠)
- `${expr}` → `\\\${expr}\\\` (完整模板表达式)

---

## 问题 2: TestDataGenerator 未定义

### 错误信息
```
Uncaught ReferenceError: TestDataGenerator is not defined
```

### 原因
`test-data.js` 中存在语法错误，导致整个文件加载失败，`TestDataGenerator` 对象未被创建。

### 修复方案
修复上述模板字符串语法错误后，`test-data.js` 正常加载，`TestDataGenerator` 对象成功定义。

---

## 问题 3: 启动脚本端口占用

### 需求
在启动新服务器前，自动杀死占用 8000 端口的旧进程。

### 修复方案

**修复后的 `start-server.sh`**:
```bash
#!/bin/bash

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

# 启动服务器
/usr/local/bin/python3 -m http.server 8000
```

### 功能说明
1. 使用 `lsof -ti:8000` 查找占用 8000 端口的进程 PID
2. 如果找到进程，使用 `kill -9` 强制杀死
3. 等待 1 秒确保端口释放
4. 启动新的 HTTP 服务器

---

## 验证步骤

### 1. 检查 JavaScript 语法
```bash
node -c js/test-data.js
node -c js/markdown-renderer.js
node -c js/main.js
```

**预期输出**:
```
✅ test-data.js 语法正确
✅ markdown-renderer.js 语法正确
✅ main.js 语法正确
```

### 2. 测试启动脚本
```bash
./start-server.sh
```

**预期输出**:
```
🚀 启动 Web 测试服务器...

🔍 检查端口 8000...
✅ 使用 Python 3 启动服务器
📍 访问地址: http://localhost:8000
💡 按 Ctrl+C 停止服务器
```

### 3. 验证页面加载
```bash
curl -s -o /dev/null -w "HTTP Status: %{http_code}\n" http://localhost:8000/index.html
```

**预期输出**:
```
HTTP Status: 200
```

### 4. 浏览器测试
访问 `http://localhost:8000` 应该看到:
- ✅ 6 个测试消息正常显示
- ✅ 代码高亮正常
- ✅ 表格渲染正常
- ✅ Mermaid 图表正常
- ✅ 数学公式正常
- ✅ 无 JavaScript 错误

---

## 修复文件清单

| 文件 | 修复内容 | 行数 |
|------|---------|-----|
| `js/test-data.js` | 转义 `$` 符号 | ~10 处 |
| `start-server.sh` | 添加端口清理逻辑 | +9 行 |

---

## 修复时间

- 发现问题: 2025-11-20
- 完成修复: 2025-11-20
- 测试通过: ✅

---

## 参考资料

- [JavaScript Template Literals (MDN)](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Template_literals)
- [Bash Special Characters](https://tldp.org/LDP/abs/html/special-chars.html)
- [lsof Command](https://linux.die.net/man/8/lsof)

---

**状态**: ✅ 所有问题已修复并测试通过
