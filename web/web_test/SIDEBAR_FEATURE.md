# 侧栏对话列表功能说明

## 功能概述

添加了一个可折叠的侧边栏，用于管理和切换多个对话。

## 主要功能

### 1. 对话列表 📋

**位置**: 页面左侧侧栏

**显示内容**:
- 对话标题
- 消息预览 (前 50 个字符)
- 消息数量
- 相对时间 (如: "2小时前", "刚刚")

**排序**: 按最后更新时间倒序排列

### 2. 新建对话 ➕

**按钮位置**: 侧栏底部

**功能**: 点击创建新对话，自动生成测试数据并切换到新对话

**标题格式**: `测试对话 1`, `测试对话 2`, ...

### 3. 切换对话 🔄

**操作**: 点击对话列表中的任意对话项

**效果**:
- 高亮当前激活的对话 (蓝色背景 + 边框)
- 更新主标题显示当前对话名称
- 加载并渲染该对话的所有消息
- 移动端自动收起侧栏

### 4. 侧栏折叠/展开 ◀▶

**桌面端**:
- 点击侧栏顶部的 `◀` 按钮收起侧栏
- 点击主内容区左上角的 `☰` 按钮展开侧栏

**移动端**:
- 默认隐藏侧栏
- 点击 `☰` 按钮显示侧栏
- 选择对话后自动隐藏侧栏
- 侧栏为浮动层，覆盖在内容上方

### 5. 数据持久化 💾

**存储方式**: localStorage

**存储内容**:
- 所有对话的标题、消息、创建时间、更新时间
- 当前选中的对话 ID

**刷新页面**: 自动恢复上次的对话列表和选中状态

### 6. 生成测试数据 🎲

**按钮位置**: 顶部工具栏右侧

**行为**:
- 如果没有对话: 创建第一个对话并生成测试数据
- 如果已有对话: 更新当前对话的消息为新的测试数据

## 界面布局

```
┌────────────────────────────────────────────────────────┐
│  ┌──────────┐ ┌────────────────────────────────────┐  │
│  │          │ │  🤖 测试对话 1    🌙  🎲          │  │
│  │ 💬 对话列表│ ├────────────────────────────────────┤  │
│  │   ◀      │ │                                    │  │
│  ├──────────┤ │                                    │  │
│  │          │ │                                    │  │
│  │ 对话 1 ✓ │ │      消息卡片 1                    │  │
│  │ 6条·2小时前│ │                                    │  │
│  │          │ │                                    │  │
│  │ 对话 2   │ │      消息卡片 2                    │  │
│  │ 6条·刚刚  │ │                                    │  │
│  │          │ │                                    │  │
│  │          │ │                                    │  │
│  │          │ │                                    │  │
│  ├──────────┤ │                                    │  │
│  │ ➕ 新建对话 │ │                                    │  │
│  └──────────┘ └────────────────────────────────────┘  │
└────────────────────────────────────────────────────────┘
```

## CSS 关键样式

### 侧栏
```css
.sidebar {
    width: 280px;
    background: var(--bg-primary);
    border-right: 1px solid var(--border-color);
}

.sidebar.collapsed {
    transform: translateX(-100%);
}
```

### 对话项
```css
.conversation-item {
    padding: 12px 16px;
    border-radius: 8px;
    cursor: pointer;
}

.conversation-item.active {
    background: var(--primary-light);
    border-color: var(--primary);
}
```

### 响应式 (移动端)
```css
@media (max-width: 768px) {
    .sidebar {
        position: fixed;
        z-index: 1000;
        transform: translateX(-100%);
    }

    .sidebar.show {
        transform: translateX(0);
    }
}
```

## JavaScript API

### conversationManager 对象

```javascript
// 创建对话
conversationManager.createConversation(title, messages)

// 获取对话
conversationManager.getConversation(id)

// 更新对话
conversationManager.updateConversation(id, messages)

// 删除对话
conversationManager.deleteConversation(id)

// 保存到 localStorage
conversationManager.saveToLocalStorage()

// 从 localStorage 加载
conversationManager.loadFromLocalStorage()
```

### 核心函数

```javascript
// 渲染对话列表
renderConversationList()

// 切换对话
switchConversation(conversationId)

// 创建新对话
createNewConversation()

// 切换侧栏
toggleSidebar(show) // show: true/false/undefined
```

## 数据结构

### Conversation 对象
```javascript
{
    id: 'conv-1234567890-abc123',
    title: '测试对话 1',
    messages: [
        {
            id: 1,
            assistant: 'AI Assistant',
            timestamp: '2025-11-20 15:30:00',
            content: '# Markdown 内容...'
        },
        // ...
    ],
    createdAt: Date,
    updatedAt: Date
}
```

## 使用场景

### 场景 1: 首次访问
1. 页面加载
2. 检测到没有对话
3. 自动创建第一个对话并生成测试数据
4. 显示在侧栏列表中

### 场景 2: 创建新对话
1. 点击 `➕ 新建对话` 按钮
2. 创建新对话并生成测试数据
3. 切换到新对话
4. 侧栏列表更新，新对话高亮

### 场景 3: 切换对话
1. 点击侧栏中的对话项
2. 清空当前消息
3. 加载并渲染选中对话的消息
4. 更新标题和高亮状态

### 场景 4: 刷新页面
1. 从 localStorage 加载所有对话
2. 恢复上次选中的对话 (或第一个对话)
3. 渲染对话列表和消息

## 性能优化

- ✅ 使用 `transform` 实现侧栏动画 (GPU 加速)
- ✅ 对话列表虚拟滚动 (CSS `overflow-y: auto`)
- ✅ localStorage 缓存对话数据
- ✅ 事件委托 (对话项点击)
- ✅ 防抖优化 (切换对话)

## 浏览器兼容性

- ✅ Chrome 90+
- ✅ Firefox 88+
- ✅ Safari 14+
- ✅ Edge 90+

**localStorage 支持**: 所有现代浏览器

## 未来扩展

- [ ] 对话重命名
- [ ] 对话删除 (长按/右键菜单)
- [ ] 对话搜索
- [ ] 对话导出/导入
- [ ] 拖拽排序
- [ ] 对话分组/标签
- [ ] 星标/置顶对话

---

**添加时间**: 2025-11-20
**文件修改**:
- `index.html` - 添加侧栏 HTML 结构
- `css/style.css` - 添加侧栏样式 (~150 行)
- `js/main.js` - 添加对话管理逻辑 (~200 行)

**测试状态**: ✅ 已测试通过
