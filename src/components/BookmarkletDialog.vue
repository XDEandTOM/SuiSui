<script setup lang="ts">
const visible = defineModel<boolean>("modelValue", { required: true })
const bookmarkletCode = `javascript:(function(){
  const t=document.title, u=location.href, s=getSelection()?.toString()||'';
  let body = t + '\\n' + u;
  if(s) body += '\\n\\n> ' + s.replace(/\\n/g,'\\n> ');
  window.open('${location.origin}/?clip=' + encodeURIComponent(body), '_blank');
})()`
</script>

<template>
  <v-dialog :model-value="visible" @update:model-value="v => visible = v" max-width="420">
    <v-card class="rounded-xl pa-4">
      <div class="d-flex align-center mb-3">
        <span class="text-subtitle-2 font-weight-medium">🔖 网页剪藏</span>
        <v-spacer />
        <v-btn icon="mdi-close" size="x-small" variant="text" @click="visible = false" />
      </div>
      <div class="mb-3 text-caption text-medium-emphasis" style="line-height:1.6">
        把下面的链接拖到浏览器书签栏，在任何网页上点击它，页面标题+链接+选中文字会自动保存到碎片笔记。
      </div>
      <a :href="bookmarkletCode" class="bookmarklet-box">
        <span class="bookmarklet-text">📥 拖到书签栏</span>
      </a>
      <div class="text-caption text-medium-emphasis mt-2" style="line-height:1.5">
        提示：书签栏没显示的话，按 <kbd class="hint-kbd">Ctrl+Shift+B</kbd> 打开书签栏，然后把上面这个链接拖进去。
      </div>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.bookmarklet-box {
  display: block; padding: 12px 16px; border-radius: 10px;
  background: rgba(var(--v-theme-primary), 0.08);
  border: 2px dashed rgba(var(--v-theme-primary), 0.25);
  text-align: center; cursor: pointer; transition: all 0.15s;
}
.bookmarklet-box:hover { background: rgba(var(--v-theme-primary), 0.12); border-color: rgba(var(--v-theme-primary), 0.4); }
.bookmarklet-text { font-size: 0.9rem; font-weight: 600; color: rgb(var(--v-theme-primary)); }
.hint-kbd {
  display: inline-block; padding: 1px 5px; border-radius: 3px;
  border: 1px solid rgba(var(--v-theme-on-surface), 0.15);
  font-size: 0.75rem; font-family: inherit;
}
</style>
