// ========================================
// 增强版详情页脚本
// ========================================

let markdownRenderer = null;

// ========================================
// 初始化
// ========================================
document.addEventListener('DOMContentLoaded', () => {
    // 初始化 Markdown 渲染器
    markdownRenderer = new MarkdownRenderer();

    // 加载测试数据
    loadTestMessages();

    // 初始化事件监听器
    initEventListeners();
});

// ========================================
// 事件监听器
// ========================================
function initEventListeners() {
    document.getElementById('favoriteBtn').addEventListener('click', () => {
        alert('已添加到收藏（演示）');
    });

    document.getElementById('exportBtn').addEventListener('click', () => {
        alert('导出功能开发中...');
    });
}

// ========================================
// 加载测试消息
// ========================================
function loadTestMessages() {
    showLoading(true);

    // 模拟异步加载
    setTimeout(() => {
        const messages = TestDataGenerator.generateAllTests();
        renderMessages(messages);
        showLoading(false);

        // 更新消息数
        document.getElementById('detailMessages').textContent = `消息数: ${messages.length}`;

        // 渲染 Mermaid 图表
        setTimeout(() => {
            mermaid.run({
                querySelector: '.mermaid'
            });
        }, 100);
    }, 300);
}

// ========================================
// 渲染消息列表
// ========================================
function renderMessages(messages) {
    const messageList = document.getElementById('messageList');
    messageList.innerHTML = '';

    if (!messages || messages.length === 0) {
        messageList.innerHTML = '<p style="text-align: center; color: #1a1a1a; padding: 48px 0;">暂无消息</p>';
        return;
    }

    messages.forEach(message => {
        const messageItem = createEnhancedMessageItem(message);
        messageList.appendChild(messageItem);
    });
}

// ========================================
// 创建增强消息元素
// ========================================
function createEnhancedMessageItem(message) {
    const item = document.createElement('div');
    item.className = 'message-item';

    const time = message.created_at ?
        new Date(message.created_at).toLocaleString('zh-CN') : '未知时间';

    // 使用 Markdown 渲染器渲染内容
    const renderedContent = markdownRenderer.render(message.content);

    item.innerHTML = `
        <div class="message-header">
            <div class="message-role ${message.role}">${getRoleName(message.role)}</div>
            <div class="message-time">${time}</div>
        </div>
        <div class="markdown-content">${renderedContent}</div>
    `;

    return item;
}

// ========================================
// 获取角色名称
// ========================================
function getRoleName(role) {
    const roleMap = {
        'user': '用户',
        'assistant': '助手',
        'system': '系统'
    };
    return roleMap[role] || role;
}

// ========================================
// 显示/隐藏加载状态
// ========================================
function showLoading(show) {
    const loading = document.getElementById('loading');
    const messageList = document.getElementById('messageList');

    if (show) {
        loading.classList.add('active');
        messageList.style.display = 'none';
    } else {
        loading.classList.remove('active');
        messageList.style.display = 'flex';
    }
}

// ========================================
// 全局工具函数（供 markdown-renderer.js 调用）
// ========================================

// 切换代码块折叠
window.toggleCodeBlock = function(id) {
    const block = document.getElementById(id);
    if (block) {
        block.classList.toggle('collapsed');
    }
};

// 复制代码
window.copyCode = function(id) {
    const block = document.getElementById(id);
    if (!block) return;

    const code = block.querySelector('code');
    if (!code) return;

    const text = code.textContent;

    navigator.clipboard.writeText(text).then(() => {
        const btn = block.querySelector('.copy-btn');
        const originalText = btn.textContent;
        btn.textContent = '已复制';
        setTimeout(() => {
            btn.textContent = originalText;
        }, 2000);
    }).catch(err => {
        console.error('复制失败:', err);
        alert('复制失败');
    });
};

// 切换 Mermaid 图表折叠
window.toggleMermaidBlock = function(id) {
    const block = document.getElementById(id);
    if (block) {
        block.classList.toggle('collapsed');
    }
};

// 复制 Mermaid 代码
window.copyMermaidCode = function(id) {
    const block = document.getElementById(id);
    if (!block) return;

    const mermaidDiv = block.querySelector('.mermaid');
    if (!mermaidDiv) return;

    const text = mermaidDiv.textContent;

    navigator.clipboard.writeText(text).then(() => {
        const btn = block.querySelector('.copy-btn');
        const originalText = btn.textContent;
        btn.textContent = '已复制';
        setTimeout(() => {
            btn.textContent = originalText;
        }, 2000);
    }).catch(err => {
        console.error('复制失败:', err);
        alert('复制失败');
    });
};
