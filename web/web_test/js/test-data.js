/**
 * 测试数据生成器
 * 生成包含各种复杂格式的 Markdown 内容
 */

const TestDataGenerator = {
    /**
     * 生成所有测试消息
     */
    generateAllTests() {
        return [
            this.generateBasicMarkdown(),
            this.generateCodeSamples(),
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
            assistant: 'AI Assistant',
            timestamp: new Date().toLocaleString('zh-CN'),
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
            assistant: 'Code Assistant',
            timestamp: new Date().toLocaleString('zh-CN'),
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
    print(f"F({i}) = {fibonacci(i)}")
\`\`\`

## JavaScript 代码

\`\`\`javascript
class MessageRenderer {
    constructor(container) {
        this.container = container;
        this.messages = [];
    }

    async render(message) {
        const html = await this.processMarkdown(message);
        this.container.innerHTML += html;
    }

    processMarkdown(text) {
        return marked.parse(text);
    }
}

// 使用示例
const renderer = new MessageRenderer(document.getElementById('app'));
renderer.render('# Hello World');
\`\`\`

## SQL 查询

\`\`\`sql
SELECT
    u.name,
    u.email,
    COUNT(o.id) as order_count,
    SUM(o.total) as total_spent
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
WHERE u.created_at > '2024-01-01'
GROUP BY u.id
HAVING order_count > 5
ORDER BY total_spent DESC
LIMIT 10;
\`\`\`

## Bash 脚本

\`\`\`bash
#!/bin/bash
# 批量处理图片

for file in *.jpg; do
    echo "Processing: \\$file"
    convert "\\$file" -resize 800x600 "resized_\\$file"
done

echo "Done!"
\`\`\`

## HTML + CSS

\`\`\`html
<!DOCTYPE html>
<html>
<head>
    <style>
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }
        .card {
            background: white;
            border-radius: 8px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.1);
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="card">
            <h1>Hello World</h1>
        </div>
    </div>
</body>
</html>
\`\`\`

行内代码示例: \`npm install\`, \`git commit\`, \`docker run\``
        };
    },

    /**
     * 3. 表格测试
     */
    generateTableSample() {
        return {
            id: 3,
            assistant: 'Data Analyst',
            timestamp: new Date().toLocaleString('zh-CN'),
            content: `# 表格渲染测试

## 产品对比表

| 产品名称 | 价格 | 性能评分 | 用户评价 | 推荐指数 |
|---------|------|---------|---------|---------|
| MacBook Pro M3 | ¥14,999 | ⭐⭐⭐⭐⭐ | 优秀 | 95% |
| Dell XPS 15 | ¥12,999 | ⭐⭐⭐⭐ | 良好 | 88% |
| ThinkPad X1 | ¥11,499 | ⭐⭐⭐⭐ | 良好 | 85% |
| Surface Laptop | ¥9,999 | ⭐⭐⭐ | 中等 | 78% |

## 编程语言特性对比

| 语言 | 类型系统 | 并发模型 | 性能 | 学习曲线 | 生态系统 |
|-----|---------|---------|-----|---------|---------|
| Rust | 静态强类型 | 所有权系统 | 极高 | 陡峭 | 快速增长 |
| Go | 静态强类型 | Goroutine | 高 | 平缓 | 成熟 |
| Python | 动态强类型 | GIL + 多进程 | 中 | 平缓 | 非常成熟 |
| JavaScript | 动态弱类型 | Event Loop | 中 | 平缓 | 最成熟 |
| TypeScript | 静态类型(可选) | 继承JS | 中 | 中等 | 快速增长 |

## 项目进度表

| 任务 | 负责人 | 状态 | 开始时间 | 完成时间 | 进度 |
|-----|-------|------|---------|---------|------|
| 需求分析 | 张三 | ✅ 已完成 | 2024-01-01 | 2024-01-05 | 100% |
| UI 设计 | 李四 | ✅ 已完成 | 2024-01-06 | 2024-01-15 | 100% |
| 前端开发 | 王五 | 🚧 进行中 | 2024-01-16 | 2024-02-10 | 65% |
| 后端开发 | 赵六 | 🚧 进行中 | 2024-01-16 | 2024-02-15 | 55% |
| 测试 | 钱七 | ⏳ 未开始 | 2024-02-16 | 2024-02-28 | 0% |

## 对齐测试

| 左对齐 | 居中 | 右对齐 |
|:------|:----:|------:|
| Left | Center | Right |
| 左 | 中 | 右 |
| 123 | 456 | 789 |`
        };
    },

    /**
     * 4. Mermaid 图表测试
     */
    generateMermaidSample() {
        return {
            id: 4,
            assistant: 'Diagram Expert',
            timestamp: new Date().toLocaleString('zh-CN'),
            content: `# Mermaid 图表测试

## 流程图

\`\`\`mermaid
graph TD
    A[开始] --> B{是否登录?}
    B -->|是| C[显示主页]
    B -->|否| D[跳转登录页]
    D --> E[用户输入账号密码]
    E --> F{验证成功?}
    F -->|是| C
    F -->|否| G[显示错误信息]
    G --> E
    C --> H[结束]
\`\`\`

## 时序图

\`\`\`mermaid
sequenceDiagram
    participant U as 用户
    participant C as 客户端
    participant S as 服务器
    participant D as 数据库

    U->>C: 发送消息
    C->>S: HTTP POST /api/message
    S->>D: 保存消息
    D-->>S: 返回结果
    S-->>C: 200 OK
    C-->>U: 显示成功
\`\`\`

## 甘特图

\`\`\`mermaid
gantt
    title 项目开发计划
    dateFormat  YYYY-MM-DD
    section 设计阶段
    需求分析           :done,    des1, 2024-01-01, 2024-01-05
    UI设计            :done,    des2, 2024-01-06, 2024-01-15
    section 开发阶段
    前端开发           :active,  dev1, 2024-01-16, 2024-02-10
    后端开发           :active,  dev2, 2024-01-16, 2024-02-15
    section 测试阶段
    功能测试           :         test1, 2024-02-16, 2024-02-25
    性能测试           :         test2, 2024-02-26, 2024-02-28
\`\`\`

## 状态图

\`\`\`mermaid
stateDiagram-v2
    [*] --> 未登录
    未登录 --> 已登录: 登录成功
    已登录 --> 浏览中: 进入主页
    浏览中 --> 搜索中: 执行搜索
    搜索中 --> 浏览中: 返回
    浏览中 --> 详情页: 点击项目
    详情页 --> 浏览中: 返回
    已登录 --> 未登录: 登出
    未登录 --> [*]
\`\`\`

## 类图

\`\`\`mermaid
classDiagram
    class Animal {
        +String name
        +int age
        +makeSound()
        +eat()
    }
    class Dog {
        +String breed
        +bark()
        +fetch()
    }
    class Cat {
        +String color
        +meow()
        +scratch()
    }
    Animal <|-- Dog
    Animal <|-- Cat
\`\`\``
        };
    },

    /**
     * 5. 数学公式测试
     */
    generateMathSample() {
        return {
            id: 5,
            assistant: 'Math Tutor',
            timestamp: new Date().toLocaleString('zh-CN'),
            content: `# 数学公式测试

## 行内公式

二次方程 $ax^2 + bx + c = 0$ 的解为 $x = \\frac{-b \\pm \\sqrt{b^2-4ac}}{2a}$

圆周率 $\\pi \\approx 3.14159$,自然对数底 $e \\approx 2.71828$

## 块级公式

欧拉公式:

$$e^{i\\pi} + 1 = 0$$

傅里叶变换:

$$F(\\omega) = \\int_{-\\infty}^{\\infty} f(t) e^{-i\\omega t} dt$$

麦克斯韦方程组:

$$\\begin{aligned}
\\nabla \\cdot \\mathbf{E} &= \\frac{\\rho}{\\epsilon_0} \\\\
\\nabla \\cdot \\mathbf{B} &= 0 \\\\
\\nabla \\times \\mathbf{E} &= -\\frac{\\partial \\mathbf{B}}{\\partial t} \\\\
\\nabla \\times \\mathbf{B} &= \\mu_0\\mathbf{J} + \\mu_0\\epsilon_0\\frac{\\partial \\mathbf{E}}{\\partial t}
\\end{aligned}$$

矩阵表示:

$$\\begin{bmatrix}
a_{11} & a_{12} & a_{13} \\\\
a_{21} & a_{22} & a_{23} \\\\
a_{31} & a_{32} & a_{33}
\\end{bmatrix}$$

求和公式:

$$\\sum_{i=1}^{n} i = \\frac{n(n+1)}{2}$$

积分:

$$\\int_0^1 x^2 dx = \\frac{1}{3}$$

极限:

$$\\lim_{x \\to 0} \\frac{\\sin x}{x} = 1$$`
        };
    },

    /**
     * 6. 综合复杂测试
     */
    generateComplexMixed() {
        return {
            id: 6,
            assistant: 'Senior Assistant',
            timestamp: new Date().toLocaleString('zh-CN'),
            content: `# 综合复杂场景测试

## 场景: Web 应用架构设计

### 1. 系统架构概述

现代 Web 应用通常采用 **三层架构**:

| 层级 | 技术栈 | 职责 |
|-----|-------|------|
| 前端层 | React + TypeScript | 用户界面、交互逻辑 |
| 后端层 | Node.js + Express | API服务、业务逻辑 |
| 数据层 | PostgreSQL + Redis | 数据存储、缓存 |

### 2. 架构流程图

\`\`\`mermaid
graph LR
    A[用户浏览器] --> B[Nginx 负载均衡]
    B --> C[前端服务器 1]
    B --> D[前端服务器 2]
    C --> E[API Gateway]
    D --> E
    E --> F[用户服务]
    E --> G[订单服务]
    E --> H[支付服务]
    F --> I[(用户数据库)]
    G --> J[(订单数据库)]
    H --> K[(支付数据库)]
    F --> L[Redis 缓存]
    G --> L
    H --> L
\`\`\`

### 3. 核心代码实现

#### 前端组件 (React + TypeScript)

\`\`\`typescript
interface User {
    id: string;
    name: string;
    email: string;
    role: 'admin' | 'user';
}

const UserProfile: React.FC<{ userId: string }> = ({ userId }) => {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        fetchUser(userId)
            .then(setUser)
            .finally(() => setLoading(false));
    }, [userId]);

    if (loading) return <Spinner />;
    if (!user) return <ErrorMessage />;

    return (
        <div className="user-profile">
            <h2>{user.name}</h2>
            <p>Email: {user.email}</p>
            <Badge type={user.role} />
        </div>
    );
};
\`\`\`

#### 后端 API (Node.js + Express)

\`\`\`javascript
const express = require('express');
const { Pool } = require('pg');
const redis = require('redis');

const app = express();
const db = new Pool({ connectionString: process.env.DATABASE_URL });
const cache = redis.createClient();

// 获取用户信息 (带缓存)
app.get('/api/users/:id', async (req, res) => {
    const { id } = req.params;
    const cacheKey = \\\`user:\\\${id}\\\`;

    try {
        // 1. 先查缓存
        const cached = await cache.get(cacheKey);
        if (cached) {
            return res.json(JSON.parse(cached));
        }

        // 2. 查数据库
        const result = await db.query(
            'SELECT * FROM users WHERE id = $1',
            [id]
        );

        if (result.rows.length === 0) {
            return res.status(404).json({ error: 'User not found' });
        }

        const user = result.rows[0];

        // 3. 写入缓存 (TTL 1小时)
        await cache.setEx(cacheKey, 3600, JSON.stringify(user));

        res.json(user);
    } catch (error) {
        console.error(error);
        res.status(500).json({ error: 'Internal server error' });
    }
});

app.listen(3000, () => console.log('Server running on port 3000'));
\`\`\`

### 4. 性能优化策略

> **关键指标**: 响应时间 < 200ms,吞吐量 > 1000 req/s

优化措施:

1. **缓存层**: Redis 缓存热点数据
2. **数据库**:
   - 添加索引: \`CREATE INDEX idx_users_email ON users(email)\`
   - 连接池: 最大连接数 20
3. **CDN**: 静态资源使用 CloudFlare
4. **压缩**: Gzip 压缩响应体

### 5. 性能测试结果

| 场景 | QPS | P50延迟 | P95延迟 | P99延迟 |
|-----|-----|--------|--------|--------|
| 用户查询(冷启动) | 850 | 45ms | 120ms | 180ms |
| 用户查询(有缓存) | 3200 | 8ms | 15ms | 25ms |
| 订单创建 | 650 | 85ms | 200ms | 350ms |
| 列表查询 | 1200 | 35ms | 80ms | 120ms |

### 6. 成本分析

月度运营成本 (假设 100万 DAU):

\`\`\`python
# 成本计算
costs = {
    'server': 500,      # 服务器
    'database': 200,    # 数据库
    'cdn': 150,         # CDN
    'monitoring': 100   # 监控
}

total = sum(costs.values())
per_user = total / 1_000_000  # 每用户成本

print("总成本: " + str(total) + "/月")
print("人均成本: " + format(per_user, '.4f') + "/月")

# 输出:
# 总成本: 950/月
# 人均成本: 0.0010/月
\`\`\`

### 7. 扩展性规划

\`\`\`mermaid
gantt
    title 系统扩展路线图
    dateFormat YYYY-MM-DD
    section 发展阶段
    单体应用(10万DAU)      :done, phase1, 2024-01-01, 2024-03-31
    服务拆分(50万DAU)      :active, phase2, 2024-04-01, 2024-06-30
    微服务化(100万DAU)     :phase3, 2024-07-01, 2024-09-30
    多区域部署(500万DAU)   :phase4, 2024-10-01, 2024-12-31
\`\`\`

### 8. 关键公式

请求成功率计算:

$$SLA = \\frac{成功请求数}{总请求数} \\times 100\\%$$

系统容量估算:

$$Capacity = \\frac{QPS_{max} \\times ResponseTime_{avg}}{ConcurrentUsers}$$

---

**总结**: 通过合理的架构设计、缓存策略和性能优化,可以构建高性能、可扩展的 Web 应用。`
        };
    }
};
