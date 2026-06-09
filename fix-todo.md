# 🛠 碎碎 SuiSui — 修复清单

> 根据代码审查整理，按优先级排列。

---

## ✅ 已修复

| # | 问题 | 严重度 | 状态 |
|---|------|--------|------|
| 1 | 改密码时 `generateSalt()` 调了两次，加密 salt ≠ 存储 salt，改密码即锁号 | 🔴 紧急 | ✅ 已修复 |
| 2 | 回收站 `GET /notes/trash` 无 token 校验，任何人可枚举 username 查看别人回收站 | 🔴 安全 | ✅ 已修复 |
|   | 连带：前端 `restoreNote` / `deleteForever` 从未传 token，后端校验一直静默失败 | 🟡 功能 | ✅ 已修复 |

---

## 🔴 高优先级

### 3. 文件上传无扩展名白名单 — 存储型 XSS

**位置**: `server-go/main.go:651-663`

**问题**: `handleUploads` 只拦了 `..` 和 `/`，不校验扩展名。用户上传 `.html` / `.svg` 后通过 `http.ServeFile` 直接服务，浏览器解析为 HTML，可执行任意 JS。

**建议**: 添加扩展名白名单（`.png .jpg .gif .webp .ico`），设置 `Content-Disposition: attachment` 或 Content-Type 安全检查。

### 4. 上传 IO 错误忽略 → 可能 panic + 临时文件泄露

**位置**: `server-go/main.go:352-354, 413-415`

**问题**:
- `os.Create` 返回值用 `_` 忽略，失败时 `dst` 为 `nil`，下一行 `io.Copy(dst, file)` **panic**，整个服务崩溃
- `io.Copy` / `dst.Close` 错误均忽略，用户静默拿到失败结果
- `ParseMultipartForm` 后未调 `r.MultipartForm.RemoveAll()`，临时文件写入磁盘后永不清理，长时间运行可耗尽磁盘

**建议**: 检查所有 IO 错误并返回 500；在成功和错误路径都 `defer r.MultipartForm.RemoveAll()`。

### 5. 数据库查询可能 panic + 数据静默丢失

**位置**: `server-go/main.go:373, 501, 560, 623` 等处

**问题**:
- `rows, _ := db.Query(...)` + `defer rows.Close()` — 查询失败时 `rows` 为 `nil`，`Close()` 调用 **panic**（2 处：line 570 `getReactions`，line 619 `handleAdmin`）
- `rows.Scan()` 返回值未检查（4 处），某行解析失败时数据静默跳过
- `rows.Next()` 循环后缺 `rows.Err()` 检查，迭代中途错误丢失

**建议**: 检查 `db.Query` 错误，只在成功后 defer close；检查 `rows.Scan` 返回值；循环后调 `rows.Err()`。

### 6. 登录成功未重置限流计数

**位置**: `server-go/main.go:35-55`

**问题**: 同一 IP 输错 5 次后被限流，此时成功登录也不清除计数。用户正常操作几分钟后又被限流。

**建议**: 登录成功后在 `handleAuth` 中调用 `loginMu.Lock()` 并从 `loginAttempts` 中删除该 IP。

---

## 🟡 中优先级

### 7. SQL 执行错误全局忽略（30+ 处）

**位置**: `server-go/main.go` 各处

**问题**: 大量 `db.Exec` / `db.QueryRow` 的返回值被 `_` 忽略或完全不检查。写操作（INSERT/UPDATE/DELETE）失败时前端收到 200 OK 但数据未写入，用户蒙在鼓里。

**建议**: 写操作至少返回错误给前端；读操作至少 `log.Printf` 记录。

### 8. JSON 编解码错误忽略（17 处）

**位置**: `server-go/main.go` 各处

**问题**: `json.NewDecoder(r.Body).Decode(&body)` 返回值被忽略，空 body 时 body 为 zero-value，后续逻辑使用空串作为查询条件。`json.Marshal` 错误也用 `_` 忽略。

**建议**: 请求体解码失败时返回 400；服务端 `json.Encode` 错误可保留忽略但建议记录日志。

### 9. 前端 `addToken` 重复定义三次

**位置**: `src/stores/auth.ts:11`, `src/stores/notes.ts:7`, `src/stores/settings.ts:7`

**问题**: 三个 store 文件各自实现了一模一样的 `addToken(url)` 函数。修改逻辑要改三处，易产生遗漏。

**建议**: 抽取到 `src/utils/api.ts` 统一导出。

### 10. 笔记列表 N+1 查询 reactions

**位置**: `server-go/main.go:378`

**问题**: 遍历笔记列表时每条笔记调一次 `getReactions(id)` → 单独 SQL 查询。100 条笔记 = 1 次主查询 + 100 次子查询。

**建议**: 先查出所有笔记 ID，用 `SELECT id, emoji, username FROM reactions WHERE id IN (...)` 批量查询后在内存中组装。

### 11. 前端死代码与未使用声明

**位置**:
- `src/views/AdminPage.vue:15-17` — `snackbar` / `snackMsg` 声明了但从未赋值
- `src/components/AdminProfile.vue:7` — `defineEmits<{ back: [] }>()` 未使用
- `src/components/AdminSystem.vue:6` — `defineEmits<{ back: [] }>()` 未使用

**建议**: 移除死代码。

### 12. `playwright` 在 dependencies 中但从未使用

**位置**: `package.json`

**问题**: `playwright` 是 E2E 测试工具，放在了 `dependencies`（生产依赖）而非 `devDependencies`，且项目中没有任何测试文件。

**建议**: `npm uninstall playwright`，如有需要作为 devDependency 重新安装。

### 13. Go 单文件达到 683 行

**位置**: `server-go/main.go`

**问题**: 虽然单文件是设计选择，但 683 行已接近"大泥球"边界。新功能开发者需要在一个文件里搜索定位。

**建议**: 按功能拆分为同包多文件：`main.go`(入口+路由) + `auth.go` + `notes.go` + `admin.go` + `db.go`。

---

## 🔵 低优先级

### 14. `handleStatic` 回退 index.html 失败返回空 200

**位置**: `server-go/main.go:678-680`

**问题**: 如果 `dist/index.html` 也丢失，`data` 为 nil，写入 0 字节，但状态码是默认 200。

**建议**: 检查第二次 `ReadFile` 的错误，返回 500。

### 15. Token 通过 URL query 传递

**位置**: 全局（前端 `addToken` 函数 + 后端 `verifyToken` 逻辑）

**问题**: `?token=xxx` 出现在浏览器历史、服务器访问日志、Referer header 中，增加泄露风险。

**建议**: 统一走 `Authorization: Bearer` header，URL query 仅作 fallback。

### 16. 笔记全量加载无分页 + 缺索引

**位置**: `server-go/main.go:361`（查询），SQLite 表结构

**问题**: `GET /api/notes` 返回全部笔记，无 limit/offset。`username` 和 `created_at` 列无索引。

**建议**: 加分页参数；为 `notes(username)` 和 `notes(created_at)` 创建索引。

### 17. 端口硬编码

**位置**: `server-go/main.go:65`, `vite.config.ts`

**问题**: 后端默认 3001，前端 Vite 代理目标硬编码 `localhost:3001`。换端口要改两处。

**建议**: 改用环境变量 `PORT` / `VITE_API_PROXY`。

### 18. Vuetify dark 主题未定义

**位置**: `src/plugins/vuetify.ts`

**问题**: `defaultTheme: "system"` 但只定义了 `light` 主题。用户系统暗色模式时 Vuetify 回退到默认暗色，可能与自定义背景色不协调。

**建议**: 显式定义 `dark` 主题，或改为 `defaultTheme: "light"`。

### 19. `vue-tsc` tsconfig 弃用警告

**位置**: `tsconfig.json:7`

**问题**: `baseUrl` 将在 TypeScript 7.0 中失效，当前版本已报 deprecation。

**建议**: 迁移到 `paths` 的完整相对路径写法，或添加 `"ignoreDeprecations": "6.0"`。

### 20. 未版本化 DB 迁移

**位置**: `server-go/main.go:73-99` (`initDB`)

**问题**: `CREATE TABLE IF NOT EXISTS` + `ALTER TABLE` 的运行时迁移策略。长期迭代后新增列只能在启动时追加，已部署实例的 schema 演进难以追踪。

**建议**: 记录 schema version 到数据库中，或引入轻量迁移工具。

### 21. 零测试覆盖

**位置**: 整个项目

**问题**: 后无 Go test，前无 Vitest / Playwright。变更无安全网。

**建议**: 至少为后端 handler 加集成测试，为 store 加单元测试。

---

## 速查总表

| # | 问题 | 文件 | 严重度 | 目标 |
|---|------|------|--------|------|
| 1 | ✅ 改密码 salt 重复生成 | `main.go:329` | 🔴 紧急 | 已修复 |
| 2 | ✅ 回收站缺 token 校验 | `main.go:486`, `NotesPage.vue` | 🔴 安全 | 已修复 |
| 3 | ✅ 文件上传无扩展名白名单 | 🔴 安全 | 已修复 |
| 4 | ✅ 上传 IO 错误忽略 + 缺 RemoveAll | 🔴 正确性 | 已修复 |
| 5 | ✅ DB 查询 nil rows panic + Scan 错误忽略 | 🔴 正确性 | 已修复 |
| 6 | ✅ 登录限流成功不重置 | 🟡 功能 | 已修复 |
| 7 | ✅ SQL 执行错误全局忽略 | 🟡 正确性 | 已修复 |
| 8 | ✅ JSON 编解码错误忽略 | 🟡 正确性 | 已修复 |
| 9 | ✅ `addToken` 重复三次 | 🟡 可维护性 | 已修复 |
| 10 | ✅ N+1 查询 reactions | 🟡 性能 | 已修复 |
| 11 | ✅ 前端死代码 | 🟡 可维护性 | 已修复 |
| 12 | ✅ `playwright` 错放依赖 | 🟡 项目 | 已修复 |
| 13 | ✅ Go 单文件拆分 | 🟡 可维护性 | 已修复 |
| 14 | ✅ 回退 index.html 失败返回空 200 | 🔵 | 已修复 |
| 15 | ✅ Token 走 URL query | 🔵 安全 | 已修复 |
| 16 | ✅ 全量加载无分页 + 缺索引 | 🔵 性能 | 已修复 |
| 17 | ✅ 端口硬编码 | 🔵 | 已修复 |
| 18 | ✅ Vuetify 缺 dark 主题 | 🔵 | 已修复 |
| 19 | ✅ tsconfig 弃用警告 | 🔵 | 已修复 |
| 20 | ✅ 无版本化 DB 迁移 | 🔵 | 已修复 |
| 21 | ✅ 零测试覆盖 | 🔵 | 已修复 |
