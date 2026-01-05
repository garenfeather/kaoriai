# Web Test - Assistant 对话框渲染测试

基于 Kelivo 项目 Flutter 实现的 Web 版 Markdown 渲染测试页面。

## 功能特性

✅ 完整支持 Markdown 基础语法
✅ 代码语法高亮 (Highlight.js)
✅ 表格渲染 (支持对齐)
✅ Mermaid 图表 (流程图、时序图、甘特图等)
✅ LaTeX 数学公式 (KaTeX)
✅ 可折叠代码块
✅ 亮色/暗色主题切换
✅ 响应式设计

## 使用方法

### 1. 直接打开

```bash
# 进入目录
cd web_test

# 使用浏览器打开 index.html
open index.html  # macOS
# 或者
start index.html # Windows
```

### 2. 使用本地服务器 (推荐)

```bash
# Python 3
python3 -m http.server 8000

# 或者使用 Node.js
npx http-server -p 8000
```

然后访问 `http://localhost:8000`

## 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Marked.js | 11.1.1 | Markdown 解析 |
| Highlight.js | 11.9.0 | 代码语法高亮 |
| Mermaid | 10.6.1 | 图表渲染 |
| KaTeX | 0.16.9 | 数学公式渲染 |

## 对照原项目实现

### 代码块

**Flutter 版本**: `lib/shared/widgets/markdown_with_highlight.dart:808-1113`

- ✅ 主题混合背景 (`Color.alphaBlend`)
- ✅ 可折叠功能
- ✅ 复制按钮
- ✅ 语法高亮 (`flutter_highlight`)
- ✅ 圆角边框 (12px)

**Web 版本**: `css/style.css:263-391`, `js/markdown-renderer.js:72-102`

### 表格

**Flutter 版本**: `lib/shared/widgets/markdown_with_highlight.dart:303-489`

- ✅ 横向滚动 (移动端)
- ✅ 内部边框 (`TableBorder.horizontalInside`, `verticalInside`)
- ✅ 标题背景混合 (`Color.alphaBlend`)
- ✅ 对齐支持 (左/中/右)

**Web 版本**: `css/style.css:393-401`

### Mermaid 图表

**Flutter 版本**: `lib/shared/widgets/markdown_with_highlight.dart:1120-1528`

- ✅ 主题适配 (亮/暗)
- ✅ 可折叠
- ✅ 复制代码按钮

**Web 版本**: `css/style.css:457-493`, `js/markdown-renderer.js:104-123`

## 测试数据

测试页面包含 6 种复杂场景:

1. **基础 Markdown** - 标题、列表、引用、链接等
2. **代码示例** - Python, JavaScript, SQL, Bash, HTML
3. **表格** - 产品对比、编程语言特性、项目进度
4. **Mermaid 图表** - 流程图、时序图、甘特图、状态图、类图
5. **数学公式** - 行内公式、块级公式、矩阵、积分等
6. **综合场景** - 包含所有上述元素的复杂文档

## 目录结构

```
web_test/
├── index.html           # 主页面
├── css/
│   └── style.css       # 完整样式 (参照 Flutter 版本)
├── js/
│   ├── markdown-renderer.js  # Markdown 渲染器
│   ├── test-data.js         # 测试数据生成器
│   └── main.js              # 主应用逻辑
├── assets/             # 静态资源 (预留)
└── README.md           # 说明文档
```

## 主要差异

| 特性 | Flutter 版本 | Web 版本 |
|------|-------------|---------|
| Markdown 引擎 | gpt_markdown | Marked.js |
| 代码高亮 | flutter_highlight | Highlight.js |
| 数学公式 | flutter_math_fork | KaTeX |
| Mermaid | WebView + JS | Mermaid.js (原生) |
| 主题切换 | Provider | localStorage |

## 参考文件

- Flutter 核心渲染器: `lib/shared/widgets/markdown_with_highlight.dart`
- Flutter 消息组件: `lib/features/chat/widgets/chat_message_widget.dart`
- 依赖配置: `pubspec.yaml`

## 浏览器兼容性

- ✅ Chrome/Edge 90+
- ✅ Firefox 88+
- ✅ Safari 14+

## 作者

基于 Kelivo 项目的 Flutter 实现改编
