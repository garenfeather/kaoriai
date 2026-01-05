/**
 * 主应用逻辑
 */

// 初始化 Markdown 渲染器
const renderer = new MarkdownRenderer();

// 对话管理
const conversationManager = {
    conversations: [],
    currentConversationId: null,

    /**
     * 创建新对话
     */
    createConversation(title, messages = []) {
        const id = 'conv-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
        const conversation = {
            id,
            title: title || `对话 ${this.conversations.length + 1}`,
            messages,
            createdAt: new Date(),
            updatedAt: new Date()
        };
        this.conversations.push(conversation);
        this.saveToLocalStorage();
        return conversation;
    },

    /**
     * 获取对话
     */
    getConversation(id) {
        return this.conversations.find(c => c.id === id);
    },

    /**
     * 更新对话消息
     */
    updateConversation(id, messages) {
        const conv = this.getConversation(id);
        if (conv) {
            conv.messages = messages;
            conv.updatedAt = new Date();
            this.saveToLocalStorage();
        }
    },

    /**
     * 删除对话
     */
    deleteConversation(id) {
        const index = this.conversations.findIndex(c => c.id === id);
        if (index !== -1) {
            this.conversations.splice(index, 1);
            this.saveToLocalStorage();
            return true;
        }
        return false;
    },

    /**
     * 保存到 localStorage
     */
    saveToLocalStorage() {
        try {
            localStorage.setItem('conversations', JSON.stringify(this.conversations));
        } catch (e) {
            console.error('保存对话失败:', e);
        }
    },

    /**
     * 从 localStorage 加载
     */
    loadFromLocalStorage() {
        try {
            const data = localStorage.getItem('conversations');
            if (data) {
                this.conversations = JSON.parse(data);
                // 转换日期字符串为 Date 对象
                this.conversations.forEach(c => {
                    c.createdAt = new Date(c.createdAt);
                    c.updatedAt = new Date(c.updatedAt);
                });
            }
        } catch (e) {
            console.error('加载对话失败:', e);
        }
    }
};

/**
 * 创建 Assistant 消息卡片
 */
function createMessageCard(message) {
    const card = document.createElement('div');
    card.className = 'message-card';
    card.setAttribute('data-message-id', message.id);

    // 渲染 Markdown 内容
    const htmlContent = renderer.render(message.content);

    card.innerHTML = `
        <div class="message-header">
            <div class="avatar">🤖</div>
            <div class="message-info">
                <div class="assistant-name">${message.assistant}</div>
                <div class="timestamp">${message.timestamp}</div>
            </div>
        </div>
        <div class="message-content">
            ${htmlContent}
        </div>
    `;

    return card;
}

/**
 * 渲染消息到容器
 */
function renderMessage(message) {
    const container = document.getElementById('chat-container');
    const card = createMessageCard(message);
    container.appendChild(card);

    // 执行渲染后处理 (初始化 Mermaid)
    renderer.afterRender(card);

    // 滚动到底部
    setTimeout(() => {
        card.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }, 100);
}

/**
 * 清空所有消息
 */
function clearMessages() {
    const container = document.getElementById('chat-container');
    container.innerHTML = '';
}

/**
 * 渲染对话列表
 */
function renderConversationList() {
    const listContainer = document.getElementById('conversation-list');
    listContainer.innerHTML = '';

    if (conversationManager.conversations.length === 0) {
        listContainer.innerHTML = '<div style="padding: 20px; text-align: center; color: var(--text-secondary); font-size: 14px;">暂无对话<br>点击下方按钮新建</div>';
        return;
    }

    // 按更新时间倒序排列
    const sorted = [...conversationManager.conversations].sort((a, b) =>
        b.updatedAt.getTime() - a.updatedAt.getTime()
    );

    sorted.forEach(conv => {
        const item = document.createElement('div');
        item.className = 'conversation-item';
        if (conv.id === conversationManager.currentConversationId) {
            item.classList.add('active');
        }

        const preview = conv.messages.length > 0
            ? conv.messages[0].content.substring(0, 50)
            : '空对话';

        const timeStr = formatRelativeTime(conv.updatedAt);

        item.innerHTML = `
            <div class="conversation-item-title">${conv.title}</div>
            <div class="conversation-item-preview">${preview}...</div>
            <div class="conversation-item-meta">
                <span>${conv.messages.length} 条消息</span>
                <span>${timeStr}</span>
            </div>
        `;

        item.addEventListener('click', () => switchConversation(conv.id));
        listContainer.appendChild(item);
    });
}

/**
 * 格式化相对时间
 */
function formatRelativeTime(date) {
    const now = new Date();
    const diff = now - date;
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) return `${days}天前`;
    if (hours > 0) return `${hours}小时前`;
    if (minutes > 0) return `${minutes}分钟前`;
    return '刚刚';
}

/**
 * 切换对话
 */
function switchConversation(conversationId) {
    conversationManager.currentConversationId = conversationId;
    const conv = conversationManager.getConversation(conversationId);

    if (conv) {
        // 更新标题
        document.getElementById('conversation-title').textContent = conv.title;

        // 清空并渲染消息
        clearMessages();
        conv.messages.forEach(message => renderMessage(message));

        // 更新列表高亮
        renderConversationList();

        // 移动端自动关闭侧栏
        if (window.innerWidth <= 768) {
            toggleSidebar(false);
        }
    }
}

/**
 * 创建新对话
 */
function createNewConversation() {
    const title = `测试对话 ${conversationManager.conversations.length + 1}`;
    const messages = TestDataGenerator.generateAllTests();
    const conv = conversationManager.createConversation(title, messages);

    renderConversationList();
    switchConversation(conv.id);
}

/**
 * 生成并显示测试数据
 */
function generateAndRenderTests() {
    // 如果没有对话，创建第一个
    if (conversationManager.conversations.length === 0) {
        createNewConversation();
    } else {
        // 如果有当前对话，重新生成消息
        if (conversationManager.currentConversationId) {
            const messages = TestDataGenerator.generateAllTests();
            conversationManager.updateConversation(conversationManager.currentConversationId, messages);
            switchConversation(conversationManager.currentConversationId);
        }
    }
}

/**
 * 主题切换
 */
function toggleTheme() {
    const html = document.documentElement;
    const currentTheme = html.getAttribute('data-theme');
    const newTheme = currentTheme === 'dark' ? 'light' : 'dark';

    html.setAttribute('data-theme', newTheme);
    localStorage.setItem('theme', newTheme);

    // 更新按钮文本
    const btn = document.getElementById('theme-toggle');
    btn.textContent = newTheme === 'dark' ? '☀️ 切换主题' : '🌙 切换主题';

    // 更新代码高亮主题
    const darkStyle = document.getElementById('highlight-dark');
    const lightStyle = document.getElementById('highlight-light');

    if (newTheme === 'dark') {
        darkStyle.removeAttribute('disabled');
        lightStyle.setAttribute('disabled', 'disabled');
    } else {
        lightStyle.removeAttribute('disabled');
        darkStyle.setAttribute('disabled', 'disabled');
    }

    // 重新初始化 Mermaid (主题切换)
    renderer.initMermaid();

    // 重新渲染所有 Mermaid 图表
    document.querySelectorAll('.mermaid-content').forEach(container => {
        const mermaidDiv = container.querySelector('.mermaid');
        if (mermaidDiv && mermaidDiv.textContent) {
            renderer.afterRender(container.parentElement);
        }
    });
}

/**
 * 切换侧栏显示/隐藏
 */
function toggleSidebar(show) {
    const sidebar = document.getElementById('sidebar');
    const showBtn = document.getElementById('show-sidebar');

    if (show === undefined) {
        // 切换
        sidebar.classList.toggle('collapsed');
    } else if (show) {
        // 显示
        sidebar.classList.remove('collapsed');
    } else {
        // 隐藏
        sidebar.classList.add('collapsed');
    }

    // 更新按钮显示状态
    const isCollapsed = sidebar.classList.contains('collapsed');
    showBtn.style.display = isCollapsed ? 'block' : 'none';
}

/**
 * 初始化应用
 */
function initApp() {
    // 恢复主题设置
    const savedTheme = localStorage.getItem('theme') || 'light';
    document.documentElement.setAttribute('data-theme', savedTheme);

    const btn = document.getElementById('theme-toggle');
    btn.textContent = savedTheme === 'dark' ? '☀️ 切换主题' : '🌙 切换主题';

    // 设置代码高亮主题
    const darkStyle = document.getElementById('highlight-dark');
    const lightStyle = document.getElementById('highlight-light');

    if (savedTheme === 'dark') {
        darkStyle.removeAttribute('disabled');
        lightStyle.setAttribute('disabled', 'disabled');
    } else {
        lightStyle.removeAttribute('disabled');
        darkStyle.setAttribute('disabled', 'disabled');
    }

    // 加载对话数据
    conversationManager.loadFromLocalStorage();

    // 绑定事件
    document.getElementById('theme-toggle').addEventListener('click', toggleTheme);
    document.getElementById('generate-test').addEventListener('click', generateAndRenderTests);
    document.getElementById('new-conversation').addEventListener('click', createNewConversation);
    document.getElementById('toggle-sidebar').addEventListener('click', () => toggleSidebar());
    document.getElementById('show-sidebar').addEventListener('click', () => toggleSidebar(true));

    // 渲染对话列表
    renderConversationList();

    // 自动加载测试数据或第一个对话
    if (conversationManager.conversations.length === 0) {
        generateAndRenderTests();
    } else {
        // 加载第一个对话
        const firstConv = conversationManager.conversations[0];
        switchConversation(firstConv.id);
    }

    // 移动端默认隐藏侧栏
    if (window.innerWidth <= 768) {
        toggleSidebar(false);
    }
}

// DOM 加载完成后初始化
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initApp);
} else {
    initApp();
}
