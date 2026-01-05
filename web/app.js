// ========================================
// API 配置
// ========================================
const API_BASE_URL = 'http://localhost:8080/api/v1';

// ========================================
// 状态管理
// ========================================
let currentPage = 1;
let pageSize = 12;
let currentFilter = 'all';
let searchKeyword = '';

// ========================================
// 初始化
// ========================================
document.addEventListener('DOMContentLoaded', () => {
    initEventListeners();
    loadConversations();
});

// ========================================
// 事件监听器
// ========================================
function initEventListeners() {
    // 搜索框
    const searchInput = document.getElementById('searchInput');
    searchInput.addEventListener('input', debounce((e) => {
        searchKeyword = e.target.value.trim();
        currentPage = 1;
        loadConversations();
    }, 300));

    // 标签过滤器
    const tagFilters = document.querySelectorAll('.tag-filter');
    tagFilters.forEach(filter => {
        filter.addEventListener('click', (e) => {
            tagFilters.forEach(f => f.classList.remove('active'));
            e.target.classList.add('active');
            currentFilter = e.target.dataset.filter;
            currentPage = 1;
            loadConversations();
        });
    });

    // 分页按钮
    document.getElementById('prevPage').addEventListener('click', () => {
        if (currentPage > 1) {
            currentPage--;
            loadConversations();
        }
    });

    document.getElementById('nextPage').addEventListener('click', () => {
        const totalPages = parseInt(document.getElementById('totalPages').textContent);
        if (currentPage < totalPages) {
            currentPage++;
            loadConversations();
        }
    });
}

// ========================================
// 加载对话列表
// ========================================
async function loadConversations() {
    showLoading(true);

    try {
        // 构建查询参数
        const params = new URLSearchParams({
            page: currentPage,
            page_size: pageSize
        });

        if (currentFilter !== 'all') {
            params.append('source_type', currentFilter);
        }

        // 调用 API
        const response = await fetch(`${API_BASE_URL}/conversations?${params}`);
        const result = await response.json();

        if (result.code === 0) {
            renderConversations(result.data.items);
            updatePagination(result.data.total, result.data.page, result.data.page_size);
        } else {
            showError('加载失败: ' + result.message);
        }
    } catch (error) {
        console.error('加载对话列表失败:', error);
        // 显示模拟数据
        renderMockData();
    } finally {
        showLoading(false);
    }
}

// ========================================
// 渲染对话卡片
// ========================================
function renderConversations(conversations) {
    const cardGrid = document.getElementById('cardGrid');
    cardGrid.innerHTML = '';

    if (!conversations || conversations.length === 0) {
        cardGrid.innerHTML = '<p style="text-align: center; color: #999; padding: 48px 0;">暂无对话数据</p>';
        return;
    }

    conversations.forEach(conv => {
        const card = createCard(conv);
        cardGrid.appendChild(card);
    });

    // 更新统计信息
    document.getElementById('totalCount').textContent = `已找到 ${conversations.length} 个对话`;
}

// ========================================
// 创建卡片元素
// ========================================
function createCard(conversation) {
    const card = document.createElement('div');
    card.className = 'card';
    card.onclick = () => goToDetail(conversation.uuid);

    // 格式化日期
    const date = conversation.created_at ? new Date(conversation.created_at).toLocaleDateString('zh-CN') : '未知';

    // 截取描述
    const description = conversation.title || '无标题';
    const truncatedDesc = description.length > 60 ? description.substring(0, 60) + '...' : description;

    card.innerHTML = `
        <div class="card-header">
            <div class="card-title">${escapeHtml(conversation.title || '无标题')}</div>
            <div class="card-count">${conversation.message_count || 0} 条消息</div>
        </div>
        <div class="card-description">${escapeHtml(truncatedDesc)}</div>
        <div class="card-meta">
            <span>${conversation.source_type || 'unknown'}</span> ·
            <span>${date}</span>
        </div>
        <div class="card-footer">
            <div class="card-tags">
                <span class="card-tag">${conversation.source_type || 'unknown'}</span>
            </div>
            <div class="card-actions">
                <button class="card-btn" onclick="viewDetail(event, '${conversation.uuid}')">详览</button>
            </div>
        </div>
    `;

    return card;
}

// ========================================
// 跳转到详情页
// ========================================
function goToDetail(uuid) {
    window.location.href = `detail.html?uuid=${uuid}`;
}

function viewDetail(event, uuid) {
    event.stopPropagation();
    goToDetail(uuid);
}

// ========================================
// 更新分页信息
// ========================================
function updatePagination(total, page, pageSize) {
    const totalPages = Math.ceil(total / pageSize);
    document.getElementById('currentPage').textContent = page;
    document.getElementById('totalPages').textContent = totalPages;

    // 更新按钮状态
    document.getElementById('prevPage').disabled = page <= 1;
    document.getElementById('nextPage').disabled = page >= totalPages;
}

// ========================================
// 显示/隐藏加载状态
// ========================================
function showLoading(show) {
    const loading = document.getElementById('loading');
    const cardGrid = document.getElementById('cardGrid');

    if (show) {
        loading.classList.add('active');
        cardGrid.style.display = 'none';
    } else {
        loading.classList.remove('active');
        cardGrid.style.display = 'grid';
    }
}

// ========================================
// 模拟数据（API 不可用时）
// ========================================
function renderMockData() {
    const mockConversations = [
        {
            uuid: 'mock-1',
            title: '自主开发',
            source_type: 'claude',
            message_count: 45,
            created_at: '2025-01-15T10:30:00Z'
        },
        {
            uuid: 'mock-2',
            title: '提示优化器',
            source_type: 'gpt',
            message_count: 18,
            created_at: '2025-01-14T15:20:00Z'
        },
        {
            uuid: 'mock-3',
            title: '工作流',
            source_type: 'claude_code',
            message_count: 22,
            created_at: '2025-01-13T09:15:00Z'
        },
        {
            uuid: 'mock-4',
            title: '文档生成器',
            source_type: 'gpt',
            message_count: 95,
            created_at: '2025-01-12T14:45:00Z'
        },
        {
            uuid: 'mock-5',
            title: 'Bug 修复',
            source_type: 'claude',
            message_count: 22,
            created_at: '2025-01-11T11:30:00Z'
        },
        {
            uuid: 'mock-6',
            title: '代码审查助手',
            source_type: 'codex',
            message_count: 30,
            created_at: '2025-01-10T16:00:00Z'
        }
    ];

    renderConversations(mockConversations);
    updatePagination(mockConversations.length, 1, pageSize);
}

// ========================================
// 工具函数
// ========================================
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

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
