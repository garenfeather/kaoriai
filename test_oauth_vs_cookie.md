# 测试：contribution.usercontent.google.com 认证方式

## 测试 1：使用 Cookie（已知可行）
```bash
curl -v 'https://contribution.usercontent.google.com/download?c=CgxiYXJk...' \
  -H 'cookie: SID=g.a000...; __Secure-1PSID=...'
```
✅ 预期：200 OK，返回文件内容

## 测试 2：使用 OAuth Token
```bash
# 首先获取 OAuth Token
# 1. 访问 https://console.cloud.google.com/apis/credentials
# 2. 创建 OAuth 2.0 Client ID
# 3. 使用 OAuth Playground 获取 token: https://developers.google.com/oauthplayground/

curl -v 'https://contribution.usercontent.google.com/download?c=CgxiYXJk...' \
  -H 'Authorization: Bearer ya29.a0AfB_xxx'
```

如果返回：
- ✅ 200 OK → 支持 OAuth，可以用标准框架
- ❌ 401/403 → 不支持 OAuth，只能用 Cookie

## 测试 3：检查 URL 参数中的 authuser
```
https://contribution.usercontent.google.com/download?...&authuser=3
```
`authuser=3` 表示这是基于浏览器 Session 的多账号切换，暗示是 Cookie-based

---

## 结论

根据你提供的 fetch 示例，**很可能是 Cookie-based**，因为：
1. ✅ 使用了 `cookie` header
2. ✅ 有 `authuser` 参数
3. ✅ 没有看到 `Authorization: Bearer` header
4. ✅ 有大量浏览器特征 headers（sec-ch-ua-*, x-client-data）

如果是这样，标准 OAuth 可能不适用。
