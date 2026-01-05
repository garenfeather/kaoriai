/**
 * 测试数据加载器 - 从 web_test 复制简化版
 */

const TestDataGenerator = {
    /**
     * 生成所有测试消息
     */
    generateAllTests() {
        return [
            this.generateBasicMarkdown(),
            this.generateCodeSamples(),
            this.generateHtmlSample(),
            this.generateTableSample(),
            this.generateMermaidSample(),
            this.generateMathSample(),
            this.generateComplexMixed()
        ];
    },

    /**
     * 1. 基础 Markdown 测试
     */
    generateBasicMarkdown() {
        return {
            id: 1,
            role: 'assistant',
            created_at: new Date().toISOString(),
            content: `# 基础 Markdown 测试

## 文本格式

这是一段普通文本,包含 **粗体**、*斜体*、~~删除线~~ 和 \`行内代码\`。

## 列表

### 无序列表
- 列表项 1
- 列表项 2
  - 嵌套项 2.1
  - 嵌套项 2.2
- 列表项 3

### 有序列表
1. 第一步
2. 第二步
3. 第三步

## 引用

> 这是一段引用文本。
> 可以有多行。
>
> — 某位智者

## 分隔线

---

## 链接

访问 [GitHub](https://github.com) 了解更多信息。`
        };
    },

    /**
     * 2. 代码示例测试
     */
    generateCodeSamples() {
        return {
            id: 2,
            role: 'assistant',
            created_at: new Date().toISOString(),
            content: `# 代码示例测试

## Python 代码

\`\`\`python
def fibonacci(n):
    """计算斐波那契数列"""
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

# 测试
for i in range(10):
    print(f"fibonacci({i}) = {fibonacci(i)}")
\`\`\`

## JavaScript 代码

\`\`\`javascript
class User {
    constructor(name, email) {
        this.name = name;
        this.email = email;
    }

    greet() {
        return \`Hello, my name is \${this.name}\`;
    }
}

const user = new User('Alice', 'alice@example.com');
console.log(user.greet());
\`\`\`

## Bash 脚本

\`\`\`bash
#!/bin/bash
for file in *.txt; do
    echo "Processing $file"
    wc -l "$file"
done
\`\`\``
        };
    },

    /**
     * 3. HTML 代码测试
     */
    generateHtmlSample() {
        return {
            id: 3,
            role: 'assistant',
            created_at: new Date().toISOString(),
            content: `# HTML 代码测试

## 基础 HTML 结构

\`\`\`html
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>示例页面</title>
    <link rel="stylesheet" href="style.css">
</head>
<body>
    <header class="header">
        <h1>欢迎使用</h1>
        <nav>
            <a href="#home">首页</a>
            <a href="#about">关于</a>
        </nav>
    </header>

    <main class="container">
        <section id="home">
            <h2>主页内容</h2>
            <p>这是一个示例段落。</p>
        </section>
    </main>

    <footer>
        <p>&copy; 2025 示例网站</p>
    </footer>

    <script src="app.js"></script>
</body>
</html>
\`\`\`

## CSS 样式

\`\`\`css
.header {
    display: flex;
    justify-content: space-between;
    padding: 20px;
    border-bottom: 1px solid #1a1a1a;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 24px;
}
\`\`\``
        };
    },

    /**
     * 4. 表格示例测试
     */
    generateTableSample() {
        return {
            id: 4,
            role: 'assistant',
            created_at: new Date().toISOString(),
            content: `# 表格示例测试

## 项目对比表

| 特性 | 方案 A | 方案 B | 方案 C |
|------|--------|--------|--------|
| 性能 | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| 成本 | ¥1000 | ¥2000 | ¥1500 |
| 维护性 | 简单 | 中等 | 复杂 |
| 推荐度 | 🟢 | 🟢 | 🟡 |

## 技术栈选择

| 技术 | 用途 | 难度 | 备注 |
|------|------|------|------|
| React | 前端框架 | 中等 | 适合大型应用 |
| Vue | 前端框架 | 简单 | 易学易用 |
| Node.js | 后端 | 中等 | JavaScript 全栈 |
| Python | 后端 | 简单 | 生态丰富 |`
        };
    },

    /**
     * 5. Mermaid 图表测试
     */
    generateMermaidSample() {
        return {
            id: 5,
            role: 'assistant',
            created_at: new Date().toISOString(),
            content: `# Mermaid 图表测试

## 流程图

\`\`\`mermaid
graph TD
    A[开始] --> B{条件判断}
    B -->|是| C[执行操作A]
    B -->|否| D[执行操作B]
    C --> E[结束]
    D --> E
\`\`\`

## 时序图

\`\`\`mermaid
sequenceDiagram
    participant 用户
    participant 前端
    participant 后端
    participant 数据库

    用户->>前端: 发起请求
    前端->>后端: API 调用
    后端->>数据库: 查询数据
    数据库-->>后端: 返回结果
    后端-->>前端: 返回数据
    前端-->>用户: 显示结果
\`\`\``
        };
    },

    /**
     * 6. 数学公式测试
     */
    generateMathSample() {
        return {
            id: 6,
            role: 'assistant',
            created_at: new Date().toISOString(),
            content: `# 数学公式测试

行内公式: $E = mc^2$

块级公式:

$$
\\frac{-b \\pm \\sqrt{b^2 - 4ac}}{2a}
$$

矩阵:

$$
\\begin{bmatrix}
a & b \\\\
c & d
\\end{bmatrix}
$$`
        };
    },

    /**
     * 7. 复杂混合内容测试
     */
    generateComplexMixed() {
        return {
            id: 7,
            role: 'assistant',
            created_at: new Date().toISOString(),
            content: `# 复杂混合内容测试

## 项目开发流程

### 1. 需求分析

首先我们需要明确项目需求,包括:

- **功能需求**: 系统必须实现的功能
- **性能需求**: 响应时间、并发量等
- **安全需求**: 数据加密、权限控制等

### 2. 技术选型

#### 前端技术栈

\`\`\`javascript
// React + TypeScript 示例
interface Props {
    title: string;
    count: number;
}

const Counter: React.FC<Props> = ({ title, count }) => {
    return (
        <div>
            <h2>{title}</h2>
            <p>Count: {count}</p>
        </div>
    );
};
\`\`\`

#### 后端架构

\`\`\`mermaid
graph LR
    A[客户端] --> B[API Gateway]
    B --> C[Auth Service]
    B --> D[User Service]
    B --> E[Data Service]
    D --> F[Database]
    E --> F
\`\`\`

### 3. 性能指标

| 指标 | 目标值 | 当前值 |
|------|--------|--------|
| 响应时间 | < 100ms | 85ms |
| 并发量 | > 10000 | 12000 |
| 可用性 | 99.9% | 99.95% |

### 4. 算法复杂度

时间复杂度: $O(n \\log n)$

空间复杂度: $O(n)$

> 💡 **提示**: 在实际开发中，需要根据具体场景选择合适的算法和数据结构。`
        };
    }
};
