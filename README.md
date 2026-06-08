# MengJi

一个轻量级备忘录应用，支持 Markdown、标签、图片上传、热力图、暗色模式等特性。

## 技术栈

| 前端 | 后端 |
|------|------|
| Vue 3 + TypeScript | Go |
| Vuetify 4 | SQLite (modernc.org/sqlite) |
| Pinia | 单文件编译（前端嵌入后端） |
| Marked + Highlight.js | RESTful API |
| Vite 6 | |

## 功能

- 备忘录 Markdown 编辑与渲染（语法高亮、代码复制）
- 多图上传 + Carousel 轮播 + 点击放大
- 标签分类与搜索
- 置顶与删除
- 活动热力图（按日统计）
- 用户注册/登录
- 后台管理（用户管理、系统设置、备案号、自定义图标）
- 暗色/亮色主题切换
- 移动端适配
- 单二进制部署，无需 Node.js 运行环境

## 快速开始

### 开发模式

`ash
# 1. 安装前端依赖
npm install

# 2. 启动 Go 后端（端口 3001）
cd server-go
go run main.go

# 3. 启动 Vite 开发服务器（端口 5173，自动代理 /api 到后端）
cd ..
npx vite --port 5173 --host
`

打开 http://localhost:5173

### 生产构建

`ash
# 构建前端
npm run build

# 编译 Go 后端（前端静态文件自动嵌入）
cd server-go
go build -o mengji .

# 运行
./mengji
# 或指定端口
./mengji -port 8080
`

默认管理员：dmin / admin

## 项目结构

`
├── src/
│   ├── components/     # 组件
│   │   ├── NoteCard.vue       # 备忘录卡片
│   │   ├── MarkdownPreview.vue # Markdown 渲染
│   │   ├── Heatmap.vue        # 活动热力图
│   │   ├── LoginDialog.vue    # 登录/注册弹窗
│   │   ├── AppIconPicker.vue  # 工具栏图标选择
│   │   ├── AvatarPicker.vue   # 头像选择
│   │   └── FaviconPicker.vue  # 网站图标选择
│   ├── views/
│   │   ├── NotesPage.vue      # 主页面（内容区 + 侧栏）
│   │   └── AdminPage.vue      # 后台管理
│   ├── stores/
│   │   ├── auth.ts            # 用户认证状态
│   │   └── notes.ts           # 备忘录数据状态
│   ├── plugins/vuetify.ts     # Vuetify 配置
│   └── main.ts
├── server-go/
│   └── main.go                # Go 后端（API + 静态文件服务）
├── vite.config.ts
└── package.json
`

## API 概览

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/auth/login | 登录 |
| POST | /api/auth/register | 注册 |
| GET | /api/auth/verify | 验证登录状态 |
| GET | /api/notes | 获取备忘录列表 |
| POST | /api/notes | 创建备忘录 |
| PUT | /api/notes/:id | 更新备忘录 |
| DELETE | /api/notes/:id | 删除备忘录 |
| PATCH | /api/notes/:id/pin | 切换置顶 |
| POST | /api/notes/upload | 上传图片 |
| GET | /api/settings | 获取系统设置 |
| POST | /api/settings | 保存系统设置 |
| GET | /api/admin/stats | 管理员统计 |
| GET | /api/admin/users | 用户列表 |

## 配置

系统设置通过后台管理界面配置：

- **网站标题** — 浏览器标签页标题
- **备案号** — 页面底部显示的 ICP 备案号
- **允许新用户注册** — 开关控制
- **工具栏图标** — 自定义应用图标
- **网站图标 (Favicon)** — 自定义浏览器标签图标

## License

MIT
