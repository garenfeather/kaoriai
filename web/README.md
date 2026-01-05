# Session History Web 前端

基于参考设计图的点线式简洁风格实现的对话历史查看前端。

## 文件结构

```
web/
├── index.html      # 对话列表页
├── detail.html     # 对话详情页
├── style.css       # 统一样式文件
├── app.js          # 列表页逻辑
├── detail.js       # 详情页逻辑
├── IMG_7868.JPG    # 设计参考图
└── README.md       # 本文档
```

## 设计特点

严格参照 IMG_7868.JPG 的设计风格：

### 1. 点线式简洁设计
- 清晰的黑色边框线条
- 圆角矩形卡片
- 黑白灰主色调
- 按钮轮廓边框

### 2. 布局结构
- 顶部导航栏（Logo + 导航链接 + 设置按钮）
- 搜索框
- 标签过滤器（圆角标签按钮）
- 3列卡片网格布局
- 底部分页
- 页脚信息

### 3. 卡片设计
- 白色背景 + 黑色边框
- 悬停时加深边框颜色
- 标题 + 描述 + 元信息
- 底部标签和操作按钮
- 右上角统计信息

## 使用方法

### 1. 直接打开（使用模拟数据）

```bash
# 在浏览器中打开
open web/index.html
```

### 2. 连接后端 API

确保后端服务运行在 `http://localhost:8080`，页面会自动连接 API。

如果 API 不可用，页面会自动降级显示模拟数据。

### 3. 使用简单 HTTP 服务器

```bash
cd web
python3 -m http.server 3000
```

然后访问 `http://localhost:3000`

## 功能说明

### 对话列表页 (index.html)
- ✅ 显示对话卡片（标题、来源、消息数、创建时间）
- ✅ 按来源过滤（GPT/Claude/Codex/Claude Code）
- ✅ 搜索功能
- ✅ 分页浏览
- ✅ 点击卡片跳转详情页

### 对话详情页 (detail.html)
- ✅ 显示对话基本信息
- ✅ 显示完整消息列表
- ✅ 区分用户/助手消息
- ✅ 添加到收藏
- ✅ 返回列表

## API 接口

页面会调用以下 API：

```
GET  /api/v1/conversations              # 获取对话列表
GET  /api/v1/conversations/:uuid        # 获取对话详情
GET  /api/v1/conversations/:uuid/messages  # 获取对话消息
POST /api/v1/favorites                  # 添加收藏
```

## 响应式设计

- 桌面端：3列卡片网格
- 平板/移动端：自动调整为1列布局

## 浏览器兼容性

支持现代浏览器：
- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+
