# Fix TODO

## Resolved

- [#4] Heatmap.vue - store.notes empty crash
- [#5] Heatmap.vue - colors follow theme color (already using var(--v-theme-primary))
- [#6] server-go/main.go - Password hash with per-user salt (16 bytes random salt, SHA-256(salt+pwd))
- [#7] server-go/main.go - Auth tokens (random 32-byte token on login, stored in auth_tokens table)
- [#8] server-go/main.go - File upload size limit (10MB via MaxBytesReader + ParseMultipartForm)

## Remaining

### Medium Priority


### Low Priority


## Newly Found

### Critical (Security)


### Medium

- [#H] `src/stores/notes.ts` - addToken() reads from localStorage directly instead of getAuthToken()- [#12] NotesPage.vue - Removed localStorage filter, added backend trash/restore/hard-delete endpoints
- [#13] EmojiPicker.vue - Deleted unused duplicate file
- [#14] MarkdownPreview.vue - data-code uses encodeURIComponent/decodeURIComponent
- [#15] MarkdownPreview.vue - SVG child click fixed with pointer-events:none
- [#16] NotesPage.vue - zoomedUpload teleport cleanup on beforeUnmount
- [#17] AdminPage.vue - User list pagination (backend + frontend)
- [#A] `server-go/main.go` - Auth PATCH handlers (avatar, nickname, theme, app-icon) token verification added
- [#B] `server-go/main.go` - Auth password change token verification added
- [#C] `server-go/main.go` - Settings POST now uses token verification instead of `?admin=` query param
- [#D] `server-go/main.go` - Avatar upload token verification added
- [#E] `server-go/main.go` + `auth.ts` - Register now returns token, frontend saves it
- [#F] `src/stores/auth.ts` - updateAvatar/Nickname/ThemeColor/AppIcon now pass token via addToken()
- [#G] `src/stores/settings.ts` - save() now compatible with backend token verification (fixed by #C)
- [#H] `src/stores/notes.ts` + `settings.ts` - addToken() now uses getAuthToken() with fallback to localStorage

