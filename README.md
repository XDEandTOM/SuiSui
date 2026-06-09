<div align="center">

# ✨ 碎碎 SuiSui

**轻量级备忘录 SPA — 写你所想，存你所记**

<p align="center">
  <img src="https://img.shields.io/badge/Vue_3-4FC08D?style=for-the-badge&logo=vuedotjs&logoColor=white" alt="Vue 3"/>
  <img src="https://img.shields.io/badge/TypeScript-3178C6?style=for-the-badge&logo=typescript&logoColor=white" alt="TypeScript"/>
  <img src="https://img.shields.io/badge/Vuetify_4-1867C0?style=for-the-badge&logo=vuetify&logoColor=white" alt="Vuetify 4"/>
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go"/>
  <img src="https://img.shields.io/badge/SQLite-003B57?style=for-the-badge&logo=sqlite&logoColor=white" alt="SQLite"/>
</p>

<p align="center">
  <a href="https://github.com/Linraintong/SuiSui/releases"><img src="https://img.shields.io/github/v/tag/Linraintong/SuiSui?style=flat-square" alt="Release"/></a>
  <a href="https://github.com/Linraintong/SuiSui/blob/main/LICENSE"><img src="https://img.shields.io/github/license/Linraintong/SuiSui?style=flat-square" alt="License"/></a>
  <a href="https://github.com/Linraintong/SuiSui/commits/main"><img src="https://img.shields.io/github/last-commit/Linraintong/SuiSui?style=flat-square" alt="Last Commit"/></a>
  <a href="https://github.com/Linraintong/SuiSui"><img src="https://img.shields.io/github/repo-size/Linraintong/SuiSui?style=flat-square" alt="Repo Size"/></a>
</p>

</div>

---

## 📖 简介

**碎碎 (SuiSui)** 是一个**单二进制部署**的轻量级备忘录应用。支持 Markdown 编辑、图片上传、Emoji 反应、标签分类、活动热力图、多用户权限和后台管理。前端 Vue 3 + 后端 Go 编译为**一个可执行文件**，开箱即用。

> 「碎碎」—— 捕捉每一丝灵感碎片。

---

## ✨ Features

### 📝 笔记核心

| 功能 | 说明 |
|------|------|
| Markdown 编辑器 | 工具栏快捷插入 · 粗体 / 斜体 / 标题 / 代码 / 链接 / 列表 / 引用 |
| 多图上传 | 拖拽上传 · 自动轮播 · 点击放大预览 |
| 标签系统 | 标签筛选 · 一键过滤 · 按频次排序 |
| 全文搜索 | 实时搜索笔记内容和标签 |
| 置顶排序 | 重要笔记置顶 · 按时间倒序 |

### 🎨 用户体验

| 功能 | 说明 |
|------|------|
| Emoji 反应 | 丰富的 emoji 库 · 游客也可参与 |
| 活动热力图 | 月度日历 · 按笔记数量着色 |
| 暗色模式 | 一键切换 · 跟随系统 |
| 自定义主题色 | 每位用户独立设置主色 |
| 响应式适配 | 桌面侧边栏 · 移动端底部导航 |

### 🔐 系统管理

| 功能 | 说明 |
|------|------|
| 用户系统 | 注册 / 登录 · 角色权限（普通用户 / 管理员） |
| 回收站 | 软删除 · 恢复 · 永久清空 |
| 后台管理 | 系统设置（标题 / 备案 / 注册开关）· 用户管理（分页/删除） |
| 数据导入导出 | JSON 格式批量导入/导出备忘录 |

### 🚀 部署特色

| 特性 | 说明 |
|------|------|
| 单文件部署 | Go 二进制嵌入前端静态资源，**一个文件运行全部** |
| SQLite 存储 | 无需数据库服务器，文件即数据库 |
| 零外部依赖 | 除浏览器外无需安装任何运行时 |

---

## 🚀 Quick Start

### 开发模式

```bash
# 终端 1：启动后端
cd server-go && go run main.go

# 终端 2：启动前端开发服务器
npx vite --port 5173 --host
```

打开 **http://localhost:5173** — 默认管理员：`admin / admin`

### 生产构建

```bash
# 构建前端 → server-go/dist/
npm run build

# 编译为单二进制文件
cd server-go && go build -o suisui .

# 运行！
./suisui              # 默认端口 3001
./suisui -port 8080   # 自定义端口
PORT=8080 ./suisui    # 环境变量（优先级高于 -port）
```

---

## 🏗️ 项目架构

```
suisui/
├── src/                         # 🎨 前端 (Vue 3 + Vuetify 4)
│   ├── main.ts                  #   入口
│   ├── App.vue                  #   根组件 (侧边栏/底部栏 + 页面路由)
│   ├── stores/                  #   Pinia 状态管理
│   │   ├── auth.ts              #     认证 / 用户信息
│   │   ├── notes.ts             #     笔记 CRUD / Emoji 反应
│   │   └── settings.ts          #     站点配置
│   ├── views/                   #   页面
│   │   ├── NotesPage.vue        #     主页面 (编辑器 + 笔记列表)
│   │   └── AdminPage.vue        #     后台管理
│   ├── components/              #   组件 (13 个)
│   │   ├── NoteCard.vue         #     笔记卡片
│   │   ├── MarkdownPreview.vue  #     Markdown 渲染
│   │   ├── Heatmap.vue          #     活动热力图
│   │   └── ...
│   ├── utils/
│   │   └── api.ts               #   共享 API 工具函数
│   └── plugins/
│       └── vuetify.ts           #   Vuetify 主题配置
│
├── server-go/                   # 🖥️ 后端 (Go)
│   ├── main.go                  #   入口 + 路由 + 静态文件服务
│   ├── db.go                    #   数据库初始化 + 工具函数
│   ├── auth.go                  #   认证 handler
│   ├── notes.go                 #   笔记 + 回收站 handler
│   ├── admin.go                 #   设置 + 管理 handler
│   ├── main_test.go             #   测试 (7 个用例)
│   ├── dist/                    #   构建后的前端 (go:embed)
│   └── uploads/                 #   用户上传文件
│
├── vite.config.ts               # Vite 配置
├── tsconfig.json                # TypeScript 配置
└── fix-todo.md                  # 修复清单
```

### 🔄 数据流

```
用户操作 → Vue 组件 → Pinia Store → authFetch(Bearer) → Go handler → SQLite
                                                                    ↓
                                              JSON Response ← 查询 / 写入
                                                                    ↓
                                              Pinia Store 更新 → Vue 响应式渲染
```

---

## 🛠️ 技术栈

| 前端 | 后端 |
|------|------|
| Vue 3 + TypeScript | Go |
| Vuetify 4 | SQLite (modernc.org/sqlite) |
| Pinia 状态管理 | RESTful API (net/http) |
| Marked + Highlight.js | 自实现 SHA-256 密码哈希 |
| Vite 6 | Token 鉴权 + IP 限流 |
| emojibase-data | 版本化 DB 迁移 |

---

## 📸 截图

> *（欢迎贡献截图！PR 区）*

| 笔记列表 | 后台管理 |
|----------|----------|
| ![Notes](https://via.placeholder.com/400x300?text=Notes+Page) | ![Admin](https://via.placeholder.com/400x300?text=Admin+Page) |

---

## 📋 API 一览

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 登录 |
| POST | `/api/auth/register` | 注册 |
| GET | `/api/auth/verify` | Token 验证 |
| PATCH | `/api/auth/avatar` | 更新头像 |
| PATCH | `/api/auth/nickname` | 更新昵称 |
| PATCH | `/api/auth/theme` | 更新主题色 |
| PATCH | `/api/auth/password` | 修改密码 |

### 笔记

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/notes?limit=&offset=` | 获取笔记列表（支持分页） |
| POST | `/api/notes` | 创建笔记 |
| PUT | `/api/notes/:id` | 更新笔记 |
| DELETE | `/api/notes/:id` | 软删除至回收站 |
| PATCH | `/api/notes/:id/pin` | 切换置顶 |
| POST/DELETE | `/api/notes/:id/react` | 添加/移除 Emoji 反应 |

### 回收站

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/notes/trash` | 查看回收站 |
| PATCH | `/api/notes/:id/restore` | 恢复笔记 |
| DELETE | `/api/notes/:id/hard-delete` | 永久删除 |

### 管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/api/settings` | 读取/更新站点设置 |
| GET | `/api/admin/stats` | 统计数据 |
| GET | `/api/admin/users` | 用户列表（分页） |
| DELETE | `/api/admin/users/:id` | 删除用户 |

---

## 🔜 Roadmap

- [x] 笔记导入 / 导出
- [x] 分页加载
- [x] 回收站
- [x] Emoji 反应
- [ ] 无限滚动
- [ ] 自动保存草稿
- [ ] 代码块主题切换
- [ ] 拖拽排序
- [ ] WebSocket 实时更新

---

## 🤝 贡献

欢迎提交 Issue 和 PR！请确保：

1. 代码通过 `go vet` 和 `vue-tsc` 检查
2. 后端变更有对应测试
3. 提交信息清晰描述改动

```bash
# 运行测试
cd server-go && go test ./...

# 类型检查
npx vue-tsc --noEmit
```

---

## 📄 许可

[MIT License](LICENSE)

<div align="center">

---

**碎碎** — *Capture every spark of inspiration.*

</div>
