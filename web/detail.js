// ========================================
// API 配置
// ========================================
const API_BASE_URL = 'http://localhost:8080/api/v1';

// ========================================
// 当前对话 UUID
// ========================================
let conversationUuid = null;

// ========================================
// 初始化
// ========================================
document.addEventListener('DOMContentLoaded', () => {
    // 从 URL 获取 UUID
    const urlParams = new URLSearchParams(window.location.search);
    conversationUuid = urlParams.get('uuid');

    if (!conversationUuid) {
        alert('缺少对话ID');
        window.location.href = 'index.html';
        return;
    }

    loadConversationDetail();
    loadMessages();
    initEventListeners();
});

// ========================================
// 事件监听器
// ========================================
function initEventListeners() {
    document.getElementById('favoriteBtn').addEventListener('click', addToFavorites);
    document.getElementById('exportBtn').addEventListener('click', exportConversation);
}

// ========================================
// 加载对话详情
// ========================================
async function loadConversationDetail() {
    try {
        const response = await fetch(`${API_BASE_URL}/conversations/${conversationUuid}`);
        const result = await response.json();

        if (result.code === 0) {
            renderConversationDetail(result.data);
        } else {
            showError('加载对话详情失败: ' + result.message);
        }
    } catch (error) {
        console.error('加载对话详情失败:', error);
        // 显示模拟数据
        renderMockConversationDetail();
    }
}

// ========================================
// 渲染对话详情
// ========================================
function renderConversationDetail(conversation) {
    document.getElementById('detailTitle').textContent = conversation.title || '无标题';

    const createdDate = conversation.created_at ?
        new Date(conversation.created_at).toLocaleString('zh-CN') : '未知';

    document.getElementById('detailSource').textContent = `来源: ${conversation.source_type || 'unknown'}`;
    document.getElementById('detailDate').textContent = `创建时间: ${createdDate}`;

    // 标签
    const tagsContainer = document.getElementById('detailTags');
    tagsContainer.innerHTML = `<span class="detail-tag">${conversation.source_type || 'unknown'}</span>`;
}

// ========================================
// 加载消息列表
// ========================================
async function loadMessages() {
    showLoading(true);

    try {
        const response = await fetch(`${API_BASE_URL}/conversations/${conversationUuid}/messages?page=1&page_size=100`);
        const result = await response.json();

        if (result.code === 0) {
            renderMessages(result.data.items);
            document.getElementById('detailMessages').textContent = `消息数: ${result.data.total}`;
        } else {
            showError('加载消息失败: ' + result.message);
        }
    } catch (error) {
        console.error('加载消息失败:', error);
        // 显示模拟数据
        renderMockMessages();
    } finally {
        showLoading(false);
    }
}

// ========================================
// 渲染消息列表
// ========================================
function renderMessages(messages) {
    const messageList = document.getElementById('messageList');
    messageList.innerHTML = '';

    if (!messages || messages.length === 0) {
        messageList.innerHTML = '<p style="text-align: center; color: #999; padding: 48px 0;">暂无消息</p>';
        return;
    }

    messages.forEach(message => {
        const messageItem = createMessageItem(message);
        messageList.appendChild(messageItem);
    });
}

// ========================================
// 创建消息元素
// ========================================
function createMessageItem(message) {
    const item = document.createElement('div');
    item.className = 'message-item';

    const time = message.created_at ?
        new Date(message.created_at).toLocaleString('zh-CN') : '未知时间';

    // 提取文本内容
    let content = '(无内容)';
    if (message.content) {
        try {
            const contentObj = JSON.parse(message.content);
            if (contentObj.text) {
                content = contentObj.text;
            } else if (Array.isArray(contentObj) && contentObj.length > 0) {
                content = contentObj.map(item => item.text || '').join('\n');
            }
        } catch (e) {
            content = message.content;
        }
    }

    item.innerHTML = `
        <div class="message-header">
            <div class="message-role ${message.role}">${getRoleName(message.role)}</div>
            <div class="message-time">${time}</div>
        </div>
        <div class="message-content">${escapeHtml(content)}</div>
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
// 添加到收藏
// ========================================
async function addToFavorites() {
    try {
        const response = await fetch(`${API_BASE_URL}/favorites`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                target_type: 'conversation',
                target_id: conversationUuid,
                category: 'default',
                notes: ''
            })
        });

        const result = await response.json();
        if (result.code === 0) {
            alert('已添加到收藏');
        } else {
            alert('添加收藏失败: ' + result.message);
        }
    } catch (error) {
        console.error('添加收藏失败:', error);
        alert('已添加到收藏（模拟）');
    }
}

// ========================================
// 导出对话
// ========================================
function exportConversation() {
    alert('导出功能开发中...');
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
// 模拟数据
// ========================================
function renderMockConversationDetail() {
    const mockConversation = {
        title: '自主开发示例对话',
        source_type: 'claude',
        created_at: '2025-01-15T10:30:00Z'
    };
    renderConversationDetail(mockConversation);
}

function renderMockMessages() {
    const mockMessages = [
        {
            uuid: 'msg-1',
            role: 'user',
            content: JSON.stringify({ text: '你好，我想了解如何使用 Claude Code 开发一个命令行工具。' }),
            created_at: '2025-01-15T10:30:00Z'
        },
        {
            uuid: 'msg-2',
            role: 'assistant',
            content: JSON.stringify({ text: '你好！我很乐意帮助你了解如何使用 Claude Code 开发命令行工具。Claude Code 是一个强大的工具，可以帮助你快速开发和测试命令行应用。\n\n首先，让我们从基础开始...' }),
            created_at: '2025-01-15T10:31:00Z'
        },
        {
            uuid: 'msg-3',
            role: 'user',
            content: JSON.stringify({ text: '能给我一个具体的例子吗？' }),
            created_at: '2025-01-15T10:32:00Z'
        },
        {
            uuid: 'msg-4',
            role: 'assistant',
            content: JSON.stringify({ text: '当然可以！让我为你展示一个简单的例子。\n\n我们来创建一个基础的待办事项管理工具...' }),
            created_at: '2025-01-15T10:33:00Z'
        }
    ];

    renderMessages(mockMessages);
    document.getElementById('detailMessages').textContent = `消息数: ${mockMessages.length}`;
}

// ========================================
// 工具函数
// ========================================
function escapeHtml(text) {
    const map = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, m => map[m]);
}

function showError(message) {
    alert(message);
}
