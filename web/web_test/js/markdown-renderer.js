/**
 * Markdown 渲染器 - 基于 Kelivo 项目的 Flutter 实现
 * 支持: Markdown, 代码高亮, 表格, Mermaid, LaTeX
 */

class MarkdownRenderer {
    constructor() {
        this.initMarked();
        this.initMermaid();
    }

    /**
     * 初始化 Marked.js 配置
     */
    initMarked() {
        // 自定义渲染器
        const renderer = new marked.Renderer();

        // 代码块渲染
        renderer.code = (code, language) => {
            if (language === 'mermaid') {
                return this.renderMermaid(code);
            }
            return this.renderCodeBlock(code, language);
        };

        // 行内代码渲染
        renderer.codespan = (code) => {
            return `<code>${this.escapeHtml(code)}</code>`;
        };

        // 表格渲染
        renderer.table = (header, body) => {
            return `
                <div class="table-wrapper">
                    <table>
                        <thead>${header}</thead>
                        <tbody>${body}</tbody>
                    </table>
                </div>
            `;
        };

        // 引用块渲染
        renderer.blockquote = (quote) => {
            return `<blockquote>${quote}</blockquote>`;
        };

        // 配置 Marked
        marked.setOptions({
            renderer: renderer,
            highlight: null, // 我们使用自定义高亮
            gfm: true,
            breaks: true,
            pedantic: false,
            sanitize: false,
            smartLists: true,
            smartypants: false
        });
    }

    /**
     * 初始化 Mermaid 配置
     */
    initMermaid() {
        const isDark = document.documentElement.getAttribute('data-theme') === 'dark';
        mermaid.initialize({
            startOnLoad: false,
            theme: isDark ? 'dark' : 'default',
            securityLevel: 'loose',
            fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
            gantt: {
                titleTopMargin: 25,
                barHeight: 40,
                barGap: 8,
                topPadding: 50,
                leftPadding: 75,
                gridLineStartPadding: 35,
                fontSize: 14,
                sectionFontSize: 14,
                numberSectionStyles: 4,
                useMaxWidth: true,
                useWidth: undefined
            }
        });
    }

    /**
     * 渲染代码块 (带折叠功能)
     */
    renderCodeBlock(code, language = '') {
        const lang = this.normalizeLanguage(language);
        const displayLang = lang || '代码';
        const id = 'code-' + Math.random().toString(36).substr(2, 9);

        // 使用 highlight.js 进行语法高亮
        let highlightedCode;
        if (lang && hljs.getLanguage(lang)) {
            highlightedCode = hljs.highlight(code, { language: lang }).value;
        } else {
            highlightedCode = hljs.highlightAuto(code).value;
        }

        return `
            <div class="code-block" id="${id}">
                <div class="code-header">
                    <span class="code-language">${displayLang}</span>
                    <div class="code-actions">
                        <button class="code-btn toggle-btn" onclick="window.toggleCodeBlock('${id}')">折叠</button>
                        <button class="code-btn copy-btn" onclick="window.copyCode('${id}')">复制</button>
                    </div>
                </div>
                <div class="code-content">
                    <pre><code class="hljs ${lang}">${highlightedCode}</code></pre>
                </div>
            </div>
        `;
    }

    /**
     * 渲染 Mermaid 图表
     */
    renderMermaid(code) {
        const id = 'mermaid-' + Math.random().toString(36).substr(2, 9);

        return `
            <div class="mermaid-block" id="${id}">
                <div class="mermaid-header">
                    <span class="mermaid-label">Mermaid</span>
                    <div class="code-actions">
                        <button class="code-btn toggle-btn" onclick="window.toggleMermaidBlock('${id}')">折叠</button>
                        <button class="code-btn copy-btn" onclick="window.copyMermaidCode('${id}')">复制代码</button>
                    </div>
                </div>
                <div class="mermaid-content">
                    <div class="mermaid">${this.escapeHtml(code)}</div>
                </div>
            </div>
        `;
    }

    /**
     * 语言标识规范化 (参考 Flutter 版本)
     */
    normalizeLanguage(lang) {
        if (!lang) return '';

        const normalized = lang.trim().toLowerCase();
        const map = {
            'js': 'javascript',
            'ts': 'typescript',
            'py': 'python',
            'rb': 'ruby',
            'kt': 'kotlin',
            'sh': 'bash',
            'zsh': 'bash',
            'shell': 'bash',
            'yml': 'yaml',
            'md': 'markdown',
            'cs': 'csharp',
            'c#': 'csharp',
            'objc': 'objectivec'
        };

        return map[normalized] || normalized;
    }

    /**
     * 预处理 Markdown 文本 (参考 Flutter 版本的 _preprocessFences)
     */
    preprocessMarkdown(text) {
        let result = text;

        // 规范化换行符
        result = result.replace(/\r\n/g, '\n');

        // 自动闭合未闭合的代码块
        const fenceCount = (result.match(/^```/gm) || []).length;
        if (fenceCount % 2 === 1) {
            if (!result.endsWith('\n')) result += '\n';
            result += '```';
        }

        return result;
    }

    /**
     * 处理 LaTeX 数学公式
     */
    processLaTeX(text) {
        // 块级公式: $$...$$ 或 \[...\]
        text = text.replace(/\$\$([\s\S]+?)\$\$/g, (match, tex) => {
            try {
                return katex.renderToString(tex, { displayMode: true, throwOnError: false });
            } catch (e) {
                return match;
            }
        });

        text = text.replace(/\\\[([\s\S]+?)\\\]/g, (match, tex) => {
            try {
                return katex.renderToString(tex, { displayMode: true, throwOnError: false });
            } catch (e) {
                return match;
            }
        });

        // 行内公式: $...$ 或 \(...\)
        text = text.replace(/\$([^\$\n]+?)\$/g, (match, tex) => {
            try {
                return katex.renderToString(tex, { displayMode: false, throwOnError: false });
            } catch (e) {
                return match;
            }
        });

        text = text.replace(/\\\(([^\)]+?)\\\)/g, (match, tex) => {
            try {
                return katex.renderToString(tex, { displayMode: false, throwOnError: false });
            } catch (e) {
                return match;
            }
        });

        return text;
    }

    /**
     * 完整渲染 Markdown
     */
    render(markdown) {
        // 预处理
        let text = this.preprocessMarkdown(markdown);

        // 处理 LaTeX (在 Markdown 解析之前)
        text = this.processLaTeX(text);

        // 解析 Markdown
        let html = marked.parse(text);

        return html;
    }

    /**
     * HTML 转义
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    /**
     * 渲染完成后的处理 (初始化 Mermaid 图表)
     */
    afterRender(container) {
        // 渲染所有 Mermaid 图表
        const mermaidElements = container.querySelectorAll('.mermaid');
        mermaidElements.forEach(async (element) => {
            try {
                const code = element.textContent;
                const { svg } = await mermaid.render('mermaid-svg-' + Math.random().toString(36).substr(2, 9), code);
                element.innerHTML = svg;
            } catch (error) {
                element.innerHTML = `<pre style="color: red;">Mermaid 渲染错误:\n${error.message}</pre>`;
            }
        });
    }
}

// 全局工具函数

/**
 * 切换代码块折叠状态
 */
window.toggleCodeBlock = function(id) {
    const block = document.getElementById(id);
    if (block) {
        block.classList.toggle('collapsed');
        const btn = block.querySelector('.toggle-btn');
        if (btn) {
            btn.textContent = block.classList.contains('collapsed') ? '展开' : '折叠';
        }
    }
};

/**
 * 切换 Mermaid 图表折叠状态
 */
window.toggleMermaidBlock = function(id) {
    const block = document.getElementById(id);
    if (block) {
        block.classList.toggle('collapsed');
        const btn = block.querySelector('.toggle-btn');
        if (btn) {
            btn.textContent = block.classList.contains('collapsed') ? '展开' : '折叠';
        }
    }
};

/**
 * 复制代码到剪贴板
 */
window.copyCode = function(id) {
    const block = document.getElementById(id);
    if (block) {
        const code = block.querySelector('code').textContent;
        navigator.clipboard.writeText(code).then(() => {
            showToast('代码已复制到剪贴板');
        }).catch(err => {
            showToast('复制失败: ' + err.message);
        });
    }
};

/**
 * 复制 Mermaid 代码到剪贴板
 */
window.copyMermaidCode = function(id) {
    const block = document.getElementById(id);
    if (block) {
        const mermaidDiv = block.querySelector('.mermaid');
        // 获取原始代码 (从 data 属性或重新从 SVG 推断)
        const code = mermaidDiv.getAttribute('data-code') || mermaidDiv.textContent;
        navigator.clipboard.writeText(code).then(() => {
            showToast('Mermaid 代码已复制');
        }).catch(err => {
            showToast('复制失败: ' + err.message);
        });
    }
};

/**
 * 显示提示消息
 */
function showToast(message) {
    // 简单的 Toast 实现
    const toast = document.createElement('div');
    toast.textContent = message;
    toast.style.cssText = `
        position: fixed;
        bottom: 20px;
        left: 50%;
        transform: translateX(-50%);
        background: rgba(0, 0, 0, 0.8);
        color: white;
        padding: 12px 24px;
        border-radius: 8px;
        font-size: 14px;
        z-index: 10000;
        animation: fadeInOut 2s ease-in-out;
    `;

    document.body.appendChild(toast);
    setTimeout(() => toast.remove(), 2000);
}

// CSS 动画
const style = document.createElement('style');
style.textContent = `
    @keyframes fadeInOut {
        0%, 100% { opacity: 0; }
        10%, 90% { opacity: 1; }
    }
`;
document.head.appendChild(style);
