# 搜索索引设计文档

## 一、概述

### 1.1 Bleve简介

**Bleve** 是一个用Go编写的全文搜索引擎库,灵感来自Apache Lucene。

**核心特性:**
- 纯Go实现,零外部依赖
- 支持多种分析器(Analyzer),包括CJK(中文/日文/韩文)
- 支持多种存储后端(BoltDB/BadgerDB/LevelDB)
- 支持BM25相关性评分
- 支持高亮(Highlighting)
- 支持facet聚合查询

**官方文档:** https://blevesearch.com/

### 1.2 为什么选择Bleve?

| 对比维度 | Bleve | Elasticsearch | Meilisearch |
|---------|-------|---------------|-------------|
| 语言 | Go | Java | Rust |
| 部署 | 嵌入式(库) | 独立服务 | 独立服务 |
| CJK支持 | 内置CJK Analyzer | 需要插件 | 内置 |
| 性能(5GB数据) | <50ms | <100ms | <30ms |
| 集成复杂度 | 低(原生) | 中(HTTP) | 中(HTTP) |
| 运维成本 | 低 | 高 | 中 |
| 适用场景 | 单机嵌入式搜索 | 分布式搜索集群 | 即时搜索体验 |

**本项目选择Bleve的理由:**
1. ✅ **嵌入式**: 与Go API Server编译到同一二进制,无需独立进程
2. ✅ **CJK优化**: 内置CJK Analyzer,支持中文/韩文分词
3. ✅ **性能足够**: 5GB数据规模下,查询延迟<50ms
4. ✅ **零运维**: 索引目录随应用一起部署,无需管理独立服务
5. ✅ **成本低**: 无需额外服务器资源

---

## 二、索引架构

### 2.1 Bleve + SQLite分层架构

```
┌──────────────────────────────────────────────────┐
│              搜索请求                             │
└─────────────────┬────────────────────────────────┘
                  │
                  ▼
┌──────────────────────────────────────────────────┐
│            Bleve全文索引                          │
│  - 存储: message_uuid + content_text + 过滤字段   │
│  - 功能: 分词、搜索、排序、高亮                   │
│  - 返回: message_uuid列表 + 高亮片段              │
└─────────────────┬────────────────────────────────┘
                  │ 返回message_uuids
                  ▼
┌──────────────────────────────────────────────────┐
│            SQLite数据库                           │
│  - 存储: 完整的message数据(JSON)                  │
│  - 功能: 根据message_uuids批量查询                │
│  - 返回: 完整消息数据                             │
└──────────────────────────────────────────────────┘
```

**职责分工:**

| 组件 | 职责 | 存储内容 |
|------|------|---------|
| **Bleve** | 全文搜索 + 过滤 | message_uuid, content_text, source_type, role, created_at |
| **SQLite** | 结构化数据存储 | 完整的conversations, messages, favorites, tags等 |

**为什么这样设计?**
1. Bleve专注搜索,存储轻量化(只存搜索必需字段)
2. SQLite存储完整数据,便于复杂关联查询
3. 避免数据重复存储,节省空间
4. 保持数据源单一(SQLite为主),Bleve为只读索引

---

### 2.2 索引目录结构

```
/opt/conversation-manager/data/bleve_index/
├── index_meta.json              # 索引元数据
├── store/                       # BadgerDB存储目录
│   ├── 000000.vlog
│   ├── 000001.sst
│   ├── MANIFEST
│   └── ...
└── ...
```

**索引大小估算:**
- 原始数据: 5GB
- 索引大小: 约15-20GB (3-4倍原始数据)
- 主要占用: 倒排索引 + 正向存储

---

## 三、索引数据结构

### 3.1 Bleve文档结构

```go
// BleveMessageDocument 索引到Bleve的文档结构
type BleveMessageDocument struct {
    // ===== 标识字段 =====
    MessageUUID       string    `json:"message_uuid"`       // 消息UUID(用于回查SQLite)
    ConversationUUID  string    `json:"conversation_uuid"`  // 对话UUID

    // ===== 搜索字段 =====
    ContentText       string    `json:"content_text"`       // 纯文本内容(全文索引)

    // ===== 过滤字段 =====
    SourceType        string    `json:"source_type"`        // 数据来源: gpt|claude|...
    Role              string    `json:"role"`               // 角色: user|assistant|system
    CreatedAt         time.Time `json:"created_at"`         // 创建时间

    // ===== 元数据字段(不索引,用于返回) =====
    ConversationTitle string    `json:"conversation_title"` // 对话标题
}
```

**字段说明:**

| 字段 | 类型 | 索引方式 | 用途 |
|------|------|---------|------|
| message_uuid | string | Stored(不索引) | 回查SQLite |
| conversation_uuid | string | Stored(不索引) | 回查SQLite |
| content_text | string | **CJK Analyzer全文索引** | 搜索主体 |
| source_type | string | Keyword(精确匹配) | 过滤来源(避免回查性能损失) |
| role | string | Keyword(精确匹配) | 过滤角色(区分用户提问/AI回答) |
| created_at | datetime | DateTimeField | 时间范围过滤 |
| conversation_title | string | Stored(不索引) | 搜索结果展示 |

### 3.2 索引Mapping定义

```go
func createIndexMapping() (*mapping.IndexMappingImpl, error) {
    // 创建CJK文本字段映射(用于content_text)
    cjkFieldMapping := bleve.NewTextFieldMapping()
    cjkFieldMapping.Analyzer = "cjk"                  // 使用CJK分析器
    cjkFieldMapping.Store = false                     // 不存储原文(节省空间)
    cjkFieldMapping.IncludeInAll = false              // 不包含在_all字段
    cjkFieldMapping.IncludeTermVectors = true         // 启用词向量(用于高亮)

    // 关键词字段映射(用于source_type, role)
    keywordFieldMapping := bleve.NewKeywordFieldMapping()
    keywordFieldMapping.Store = false

    // 日期时间字段映射(用于created_at)
    dateFieldMapping := bleve.NewDateTimeFieldMapping()
    dateFieldMapping.Store = false

    // Stored字段映射(用于message_uuid等,不索引但需返回)
    storedFieldMapping := bleve.NewKeywordFieldMapping()
    storedFieldMapping.Store = true   // 仅存储,不参与倒排
    storedFieldMapping.Index = false

    // 文档映射
    messageMapping := bleve.NewDocumentMapping()

    // 添加字段映射
    messageMapping.AddFieldMappingsAt("message_uuid", storedFieldMapping)
    messageMapping.AddFieldMappingsAt("conversation_uuid", storedFieldMapping)
    messageMapping.AddFieldMappingsAt("conversation_title", storedFieldMapping)
    messageMapping.AddFieldMappingsAt("content_text", cjkFieldMapping)
    messageMapping.AddFieldMappingsAt("source_type", keywordFieldMapping)
    messageMapping.AddFieldMappingsAt("role", keywordFieldMapping)
    messageMapping.AddFieldMappingsAt("created_at", dateFieldMapping)

    // 创建索引映射
    indexMapping := bleve.NewIndexMapping()
    indexMapping.DefaultMapping = messageMapping
    indexMapping.DefaultAnalyzer = "cjk"              // 默认使用CJK分析器
    indexMapping.DefaultDateTimeParser = "dateTimeOptional"

    return indexMapping, nil
}
```

---

## 四、CJK分析器

### 4.1 Bleve CJK Analyzer工作原理

**分词策略: 双字符Bigram**

```
原文: "使用Prometheus监控系统"

分词结果:
- 使用
- 用P
- Pr
- ro
- om
- me
- et
- th
- he
- eu
- us
- 监控
- 控系
- 系统
```

**特点:**
- ✅ 无需词典,适应性强
- ✅ 支持未登录词(新词)
- ⚠️ 分词粒度较粗,不如jieba精确
- ⚠️ 索引体积较大

### 4.2 CJK Analyzer配置

Bleve内置的CJK Analyzer包含:
1. **Unicode Tokenizer**: 按字符类型分词
2. **CJK Bigram Filter**: 对CJK字符生成bigram
3. **Lowercase Filter**: 小写转换
4. **Stop Token Filter**: 停用词过滤(可选)

```go
// Bleve内置,无需配置
analyzer := "cjk"
```

### 4.3 搜索效果示例

| 搜索词 | 原文 | 是否匹配 | 说明 |
|-------|------|---------|------|
| "监控系统" | "使用Prometheus监控系统" | ✅ 匹配 | 包含"监控"+"控系"+"系统" |
| "监控" | "使用Prometheus监控系统" | ✅ 匹配 | 包含"监控" |
| "数据库设计" | "数据库架构设计方案" | ✅ 匹配 | 包含"数据"+"据库"+"设计" |
| "监控方案" | "监控系统设计方案" | ✅ 匹配 | 包含"监控"+"方案" |
| "prometheus" | "使用Prometheus监控系统" | ✅ 匹配 | 不区分大小写 |

**召回率 vs 精确率:**
- 召回率: 高 (宽松匹配)
- 精确率: 中 (可能误召回)

**对本项目的影响:**
- ✅ 搜索对话历史,召回率优先
- ✅ 用户可以通过更多关键词缩小范围
- ✅ BM25评分会将最相关的结果排在前面

---

## 五、索引操作

### 5.1 创建索引

```go
func initBleve(indexPath string) (bleve.Index, error) {
    // 检查索引是否已存在
    if _, err := os.Stat(indexPath); err == nil {
        // 打开已有索引
        return bleve.Open(indexPath)
    }

    // 创建索引映射
    mapping, err := createIndexMapping()
    if err != nil {
        return nil, fmt.Errorf("create mapping failed: %w", err)
    }

    // 创建索引(使用BadgerDB作为存储后端)
    index, err := bleve.NewUsing(
        indexPath,
        mapping,
        scorch.Name,    // 使用Scorch索引引擎(默认)
        badger.Name,    // 使用BadgerDB存储(比BoltDB快)
        map[string]interface{}{
            "create_if_missing": true,
            "error_if_exists":   false,
        },
    )

    if err != nil {
        return nil, fmt.Errorf("create index failed: %w", err)
    }

    return index, nil
}
```

### 5.2 索引单条消息

```go
func indexMessage(index bleve.Index, msg *Message, convTitle string) error {
    // 索引user和assistant的文字内容
    if msg.ContentText == "" {
        return nil
    }

    // 构建Bleve文档
    doc := &BleveMessageDocument{
        MessageUUID:       msg.UUID,
        ConversationUUID: msg.ConversationUUID,
        ContentText:       msg.ContentText,
        SourceType:        msg.SourceType,
        Role:              msg.Role,
        CreatedAt:         msg.CreatedAt,
        ConversationTitle: convTitle,
    }

    // 索引文档(使用message.uuid作为文档ID)
    return index.Index(msg.UUID, doc)
}
```

### 5.3 批量索引

```go
func batchIndexMessages(index bleve.Index, messages []*Message) error {
    batch := index.NewBatch()
    batchSize := 0

    for _, msg := range messages {
        // 索引user和assistant的文字内容
        if msg.ContentText == "" {
            continue
        }

        doc := &BleveMessageDocument{
            MessageUUID:      msg.UUID,
            ConversationUUID: msg.ConversationUUID,
            ContentText:      msg.ContentText,
            SourceType:       msg.SourceType,
            Role:             msg.Role,
            CreatedAt:        msg.CreatedAt,
        }

        batch.Index(msg.UUID, doc)
        batchSize++

        // 每1000条提交一次
        if batchSize >= 1000 {
            if err := index.Batch(batch); err != nil {
                return fmt.Errorf("batch index failed: %w", err)
            }
            batch = index.NewBatch()
            batchSize = 0
            log.Printf("Indexed 1000 messages")
        }
    }

    // 提交剩余
    if batchSize > 0 {
        if err := index.Batch(batch); err != nil {
            return fmt.Errorf("batch index failed: %w", err)
        }
        log.Printf("Indexed %d messages", batchSize)
    }

    return nil
}
```

### 5.4 更新索引

```go
func updateMessage(index bleve.Index, msg *Message) error {
    // Bleve的Index()方法会覆盖已有文档
    return indexMessage(index, msg, "")
}
```

### 5.5 删除索引

```go
func deleteMessage(index bleve.Index, messageUUID string) error {
    return index.Delete(messageUUID)
}
```

---

## 六、搜索查询

### 6.1 基础关键词搜索

```go
func searchBasic(index bleve.Index, keyword string) (*bleve.SearchResult, error) {
    // 创建匹配查询
    query := bleve.NewMatchQuery(keyword)
    query.SetField("content_text")
    query.Analyzer = "cjk"

    // 创建搜索请求
    searchRequest := bleve.NewSearchRequest(query)
    searchRequest.Size = 20
    searchRequest.From = 0

    // 按相关性排序(默认)
    searchRequest.SortBy([]string{"-_score"})

    // 执行搜索
    return index.Search(searchRequest)
}
```

### 6.2 复合查询(多条件)

```go
func searchAdvanced(index bleve.Index, req *SearchRequest) (*bleve.SearchResult, error) {
    var mustQueries []query.Query

    // 1. 关键词查询
    if req.Keyword != "" {
        matchQuery := bleve.NewMatchQuery(req.Keyword)
        matchQuery.SetField("content_text")
        matchQuery.Analyzer = "cjk"
        mustQueries = append(mustQueries, matchQuery)
    }

    // 2. 来源过滤
    if len(req.Sources) > 0 {
        var sourceQueries []query.Query
        for _, src := range req.Sources {
            q := bleve.NewTermQuery(src)
            q.SetField("source_type")
            sourceQueries = append(sourceQueries, q)
        }
        mustQueries = append(mustQueries,
            bleve.NewDisjunctionQuery(sourceQueries...)) // OR关系
    }

    // 3. 角色过滤(可选: user|assistant)
    if len(req.Roles) > 0 {
        var roleQueries []query.Query
        for _, role := range req.Roles {
            q := bleve.NewTermQuery(role)
            q.SetField("role")
            roleQueries = append(roleQueries, q)
        }
        mustQueries = append(mustQueries,
            bleve.NewDisjunctionQuery(roleQueries...)) // OR关系
    }

    // 4. 时间范围过滤
    if !req.DateFrom.IsZero() || !req.DateTo.IsZero() {
        dateQuery := bleve.NewDateRangeQuery(req.DateFrom, req.DateTo)
        dateQuery.SetField("created_at")
        mustQueries = append(mustQueries, dateQuery)
    }

    // 组合查询(AND关系)
    finalQuery := bleve.NewConjunctionQuery(mustQueries...)

    // 创建搜索请求
    searchRequest := bleve.NewSearchRequest(finalQuery)
    searchRequest.Size = req.PageSize
    searchRequest.From = (req.Page - 1) * req.PageSize

    // 排序
    if req.SortBy == "time_desc" {
        searchRequest.SortBy([]string{"-created_at"})
    } else if req.SortBy == "time_asc" {
        searchRequest.SortBy([]string{"created_at"})
    } else {
        searchRequest.SortBy([]string{"-_score", "-created_at"})
    }

    // 高亮配置
    searchRequest.Highlight = bleve.NewHighlight()
    searchRequest.Highlight.AddField("content_text")
    searchRequest.Highlight.Style = ansi.Name  // 或html.Name

    // 执行搜索
    return index.Search(searchRequest)
}
```

### 6.3 短语精确匹配

```go
func searchPhrase(index bleve.Index, phrase string) (*bleve.SearchResult, error) {
    // 短语查询(词序必须完全匹配)
    query := bleve.NewMatchPhraseQuery(phrase)
    query.SetField("content_text")
    query.Analyzer = "cjk"

    searchRequest := bleve.NewSearchRequest(query)
    searchRequest.Size = 20

    return index.Search(searchRequest)
}
```

### 6.4 前缀匹配

```go
func searchPrefix(index bleve.Index, prefix string) (*bleve.SearchResult, error) {
    // 前缀查询
    query := bleve.NewPrefixQuery(prefix)
    query.SetField("content_text")

    searchRequest := bleve.NewSearchRequest(query)
    searchRequest.Size = 20

    return index.Search(searchRequest)
}
```

### 6.5 模糊匹配

```go
func searchFuzzy(index bleve.Index, term string, fuzziness int) (*bleve.SearchResult, error) {
    // 模糊查询(允许编辑距离为fuzziness)
    query := bleve.NewFuzzyQuery(term)
    query.SetField("content_text")
    query.Fuzziness = fuzziness  // 1 or 2

    searchRequest := bleve.NewSearchRequest(query)
    searchRequest.Size = 20

    return index.Search(searchRequest)
}
```

### 6.6 标签过滤(tags)

与 `/api/v1/search` 的 `tags` 参数对齐，利用SQLite先筛选对话，再限制Bleve的message_uuid集合：

1) **SQLite筛选conversation_uuid（满足全部标签）**
```sql
SELECT conversation_uuid
FROM conversation_tags
WHERE tag_id IN (?, ?, ...)
GROUP BY conversation_uuid
HAVING COUNT(DISTINCT tag_id) = ?; -- ? 为传入tag数量
```

2) **SQLite获取对应消息uuid集合（只取未隐藏消息）**
```sql
SELECT uuid
FROM messages
WHERE conversation_uuid IN (<上一步结果>)
  AND hidden_at IS NULL;
```

3) **Bleve侧限制docIDs**
```go
docIDQuery := bleve.NewDocIDQuery(messageUUIDs) // messageUUIDs来自上一步
finalQuery := bleve.NewConjunctionQuery(docIDQuery, keywordQuery, filters...) // keywordQuery/filters为其他搜索条件
```

说明：为保持索引精简，conversation_uuid在Bleve中仅作为Stored字段，不索引；标签过滤通过docID白名单实现，不增加索引体积。

---

## 七、高亮显示

### 7.1 高亮配置

```go
// HTML高亮(用于Web)
highlight := bleve.NewHighlightWithStyle(html.Name)
highlight.AddField("content_text")
searchRequest.Highlight = highlight

// ANSI高亮(用于终端)
highlight := bleve.NewHighlightWithStyle(ansi.Name)
highlight.AddField("content_text")
searchRequest.Highlight = highlight
```

### 7.2 自定义高亮标签

```go
// 使用自定义标签
highlight := bleve.NewHighlight()
highlight.AddField("content_text")

// 设置前后缀
fragmenter, _ := simple.NewFragmenter(100, 3) // 片段长度100字符,最多3个片段
fragmenter.Separator = "..."

highlighter := simple.NewHighlighter()
highlighter.Fragmenter = fragmenter
highlighter.Before = "<mark>"
highlighter.After = "</mark>"

highlight.Highlighter = highlighter
searchRequest.Highlight = highlight
```

### 7.3 解析高亮结果

```go
func parseHighlights(result *bleve.SearchResult) []SearchResultItem {
    items := make([]SearchResultItem, 0, len(result.Hits))

    for _, hit := range result.Hits {
        item := SearchResultItem{
            MessageUUID: hit.Fields["message_uuid"].(string),
            Score:       hit.Score,
        }

        // 提取高亮片段
        if fragments, ok := hit.Fragments["content_text"]; ok {
            item.Highlights = fragments
        }

        items = append(items, item)
    }

    return items
}
```

**高亮结果示例:**
```json
{
  "fragments": {
    "content_text": [
      "...使用<mark>Prometheus</mark>进行指标采集...",
      "...配置<mark>Prometheus</mark>的scrape_configs..."
    ]
  }
}
```

---

## 八、索引维护

### 8.1 全量重建索引

**按来源重建（推荐）：**
```go
func rebuildIndexBySource(index bleve.Index, dbConn *sql.DB, sourceType string) error {
    // 1. 删除该来源的旧索引
    query := bleve.NewTermQuery(sourceType)
    query.SetField("source_type")
    searchReq := bleve.NewSearchRequest(query)
    searchReq.Size = 10000  // 分批删除

    for {
        result, err := index.Search(searchReq)
        if err != nil || len(result.Hits) == 0 {
            break
        }

        batch := index.NewBatch()
        for _, hit := range result.Hits {
            batch.Delete(hit.ID)
        }
        index.Batch(batch)
    }

    // 2. 从SQLite查询该来源的所有消息
    rows, err := dbConn.Query(`
        SELECT m.uuid, m.conversation_uuid, m.role, m.content_text,
               m.created_at, c.source_type, c.title
        FROM messages m
        JOIN conversations c ON m.conversation_uuid = c.uuid
        WHERE m.hidden_at IS NULL
          AND c.hidden_at IS NULL
          AND c.source_type = ?
          AND m.content_text != ''
    `, sourceType)
    if err != nil {
        return fmt.Errorf("query messages failed: %w", err)
    }
    defer rows.Close()

    // 3. 批量索引
    batch := index.NewBatch()
    count := 0

    for rows.Next() {
        var msg Message
        var convTitle string
        err := rows.Scan(&msg.UUID, &msg.ConversationUUID,
                        &msg.Role, &msg.ContentText, &msg.CreatedAt,
                        &msg.SourceType, &convTitle)
        if err != nil {
            return fmt.Errorf("scan row failed: %w", err)
        }

        doc := &BleveMessageDocument{
            MessageUUID:       msg.UUID,
            ConversationUUID: msg.ConversationUUID,
            ContentText:       msg.ContentText,
            SourceType:        msg.SourceType,
            Role:              msg.Role,
            CreatedAt:         msg.CreatedAt,
            ConversationTitle: convTitle,
        }

        batch.Index(msg.UUID, doc)
        count++

        if batch.Size() >= 1000 {
            if err := index.Batch(batch); err != nil {
                return fmt.Errorf("batch index failed: %w", err)
            }
            batch = index.NewBatch()
            log.Printf("Indexed %d messages for source %s", count, sourceType)
        }
    }

    // 提交剩余
    if batch.Size() > 0 {
        if err := index.Batch(batch); err != nil {
            return fmt.Errorf("batch index failed: %w", err)
        }
    }

    log.Printf("Rebuild completed for %s: %d messages indexed", sourceType, count)
    return nil
}
```

**全部重建（慎用）：**
```go
func rebuildAllIndex(indexPath string, dbConn *sql.DB) error {
    // 删除整个索引目录并重建
    if err := os.RemoveAll(indexPath); err != nil {
        return fmt.Errorf("remove old index failed: %w", err)
    }

    index, err := initBleve(indexPath)
    if err != nil {
        return fmt.Errorf("init index failed: %w", err)
    }
    defer index.Close()

    // 按来源逐个重建
    sources := []string{"gpt", "claude", "claude_code", "codex", "gemini"}
    for _, source := range sources {
        if err := rebuildIndexBySource(index, dbConn, source); err != nil {
            log.Printf("Warning: rebuild %s failed: %v", source, err)
        }
    }

    return nil
}
```

### 8.2 增量更新索引

```go
// 新增消息后自动更新索引
func onMessageInserted(index bleve.Index, msg *Message) error {
    if msg.ContentText != "" {
        return indexMessage(index, msg, "")
    }
    return nil
}

// 删除消息后自动删除索引
func onMessageDeleted(index bleve.Index, messageUUID string) error {
    return index.Delete(messageUUID)
}

// 更新消息后自动更新索引
func onMessageUpdated(index bleve.Index, msg *Message) error {
    // Bleve的Index()会覆盖已有文档
    return indexMessage(index, msg, "")
}
```

### 8.3 索引优化

```go
// 定期执行优化(合并segment)
func optimizeIndex(index bleve.Index) error {
    // Bleve会自动merge,一般无需手动优化
    // 如果需要强制优化:
    return index.Advanced().(*scorch.Scorch).ForceMerge(1)
}
```

---

## 九、性能优化

### 9.1 索引阶段优化

**批量索引:**
```go
// ✅ 好: 批量提交
batch := index.NewBatch()
for _, msg := range messages {
    batch.Index(msg.UUID, doc)
    if batch.Size() >= 1000 {
        index.Batch(batch)
        batch.Reset()
    }
}

// ❌ 差: 逐条提交
for _, msg := range messages {
    index.Index(msg.UUID, doc)  // 每次都写磁盘
}
```

**并行索引(不推荐):**
- Bleve的写入是串行的
- 并发调用`Index()`不会提升性能
- 建议单线程批量索引

### 9.2 查询阶段优化

**分页参数:**
```go
searchRequest.Size = 20      // 每页20条
searchRequest.From = 0       // 从第0条开始

// ❌ 避免深度分页(From很大)
searchRequest.From = 10000   // 性能差
```

**字段过滤:**
```go
// ✅ 只返回需要的字段
searchRequest.Fields = []string{"message_uuid", "conversation_uuid", "conversation_title"}

// ❌ 返回所有字段
// 默认会返回所有stored字段
```

**高亮限制:**
```go
// 限制高亮片段数量
fragmenter.MaxFragments = 3  // 最多3个片段

// 限制片段长度
fragmenter.Size = 100        // 每个片段100字符
```

### 9.3 存储后端优化

**BadgerDB vs BoltDB:**

| 后端 | 写入性能 | 读取性能 | 索引大小 |
|------|---------|---------|---------|
| BoltDB | 中 | 中 | 较小 |
| BadgerDB | **快** | **快** | 较大 |

**建议:** 使用BadgerDB(默认)

**BadgerDB配置优化:**
```go
indexConfig := map[string]interface{}{
    "create_if_missing": true,
    "ValueLogFileSize":  1 << 28,  // 256MB
    "MaxTableSize":      1 << 26,  // 64MB
    "NumLevelZeroTables": 5,
    "NumLevelZeroTablesStall": 10,
}

index, err := bleve.NewUsing(indexPath, mapping,
    scorch.Name, badger.Name, indexConfig)
```

---

## 十、故障排查

### 10.1 搜索无结果

**排查步骤:**

1. **检查索引是否存在:**
```bash
ls -lh /opt/conversation-manager/data/bleve_index/
```

2. **检查索引文档数:**
```go
count, err := index.DocCount()
log.Printf("Index doc count: %d", count)
```

3. **检查查询语句:**
```go
// 打印查询语句
log.Printf("Query: %+v", query)
```

4. **检查分析器:**
```go
// 测试分析器
tokens := index.Analyzer("cjk").Analyze([]byte("测试文本"))
log.Printf("Tokens: %+v", tokens)
```

### 10.2 搜索很慢

**排查步骤:**

1. **检查索引大小:**
```bash
du -sh /opt/conversation-manager/data/bleve_index/
```

2. **检查查询条件:**
```go
// 避免通配符开头
// ❌ 慢
query := bleve.NewWildcardQuery("*监控")

// ✅ 快
query := bleve.NewWildcardQuery("监控*")
```

3. **检查排序:**
```go
// 按字段排序需要加载所有文档的该字段
// ❌ 慢(如果字段很多)
searchRequest.SortBy([]string{"created_at"})

// ✅ 快(按评分排序)
searchRequest.SortBy([]string{"-_score"})
```

### 10.3 索引占用空间过大

**优化方案:**

1. **减少stored字段:**
```go
// 不需要返回的字段,设置Store=false
fieldMapping.Store = false
```

2. **禁用term vectors(如果不需要高亮):**
```go
fieldMapping.IncludeTermVectors = false
```

---

## 十一、最佳实践

### 11.1 索引设计

1. ✅ **只索引需要搜索的字段** (content_text)
2. ✅ **用Keyword字段做精确过滤** (source_type, role)
3. ✅ **用DateTimeField做时间范围查询** (created_at)
4. ✅ **用Stored字段返回元数据** (message_uuid, conversation_uuid, conversation_title)
5. ❌ **避免索引过大的字段** (>10KB)

### 11.2 查询设计

1. ✅ **使用批量查询代替多次单查询**
2. ✅ **合理设置分页大小** (20-50条)
3. ✅ **避免深度分页** (From<1000)
4. ✅ **优先按评分排序** (_score)
5. ❌ **避免通配符开头** (*keyword)

---

## 附录

### A. 参考资料

- [Bleve官方文档](https://blevesearch.com/)
- [Bleve GitHub](https://github.com/blevesearch/bleve)
- [BadgerDB文档](https://dgraph.io/docs/badger/)
- [Scorch索引引擎](https://github.com/blevesearch/scorch)
