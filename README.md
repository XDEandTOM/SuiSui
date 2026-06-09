# 碎碎

一个轻量级备忘录应用，支持 Markdown、标签、图片上传、消息反应、热力图等功能。

## 技术栈

| 前端 | 后端 |
|------|------|
| Vue 3 + TypeScript | Go |
| Vuetify 4 | SQLite (modernc.org/sqlite) |
| Pinia | 单文件编译 |
| Marked + Highlight.js | RESTful API |
| Vite 6 | 前端静态文件嵌入 |

## 功能

- Markdown 编辑与渲染（粗体、斜体、标题、代码、链接、列表、引用）
- 多图上传 + 轮播查看 + 点击放大
- 消息反应（emojibase 完整 emoji 库，9 分类，游客可用）
- 标签分类与全文搜索
- 置顶排序
- 活动热力图
- 回收站（软删除 + 恢复 + 永久删除）
- 用户注册 / 登录 / 角色权限
- 后台管理（系统设置、用户管理、自定义主题色）
- 暗色 / 亮色主题切换
- 移动端自适应
- 自定义 Favicon 与 App 图标
- 备案号展示（可点击跳转工信部）
- 单二进制部署

## 快速开始

### 开发模式

```bash
# 安装前端依赖
npm install

# 启动 Go 后端（端口 3001）
cd server-go
go run main.go

# 启动 Vite 开发服务器（端口 5173）
cd ..
npx vite --port 5173 --host
```

打开 http://localhost:5173

### 生产构建

```bash
# 构建前端
npm run build

# 编译 Go 后端（前端静态文件自动嵌入）
cd server-go
go build -o suisui .

# 运行
./suisui

# 或指定端口
./suisui -port 8080
```

默认管理员：`admin / admin`

## 项目结构

```
├── index.html
├── vite.config.ts
├── package.json
├── src/
│   ├── main.ts              # 入口
│   ├── App.vue               # 根组件（侧边栏/底栏）
│   ├── stores/
│   │   ├── auth.ts           # 用户认证
│   │   ├── notes.ts          # 备忘录状态
│   │   └── settings.ts       # 系统设置
│   ├── views/
│   │   ├── NotesPage.vue     # 备忘录列表页
│   │   └── AdminPage.vue     # 后台管理页
│   └── components/
│       ├── NoteCard.vue        # 备忘录卡片（含消息反应）
│       ├── Heatmap.vue         # 热力图
│       ├── MarkdownPreview.vue # Markdown 渲染
│       ├── LoginDialog.vue     # 登录弹窗
│       ├── AppLogo.vue         # SVG Logo
│       ├── AppIconPicker.vue   # App 图标选择
│       └── FaviconPicker.vue   # Favicon 选择
├── server-go/
│   ├── main.go              # Go 后端
│   └── dist/                # 构建后的前端
└── README.md
```

## TODO

- [ ] 无限滚动 / 分页加载
- [ ] 自动保存草稿
- [ ] 代码块主题切换
- [x] 笔记导出 / 导入
- [ ] 置顶排序拖动
- [ ] WebSocket 实时更新

## License

MIT

