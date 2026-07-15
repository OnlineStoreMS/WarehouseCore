<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage, type UploadRequestOptions } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { uploadImage } from '../api/upload'

const props = withDefaults(defineProps<{
  modelValue?: string
  subdir?: string
  tip?: string
  /** 表格内缩略图模式：只显示上传框，不显示 URL 输入 */
  compact?: boolean
  size?: number
}>(), {
  compact: false,
  size: 120,
})
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const uploading = ref(false)
const url = computed({
  get: () => props.modelValue || '',
  set: (v) => emit('update:modelValue', v),
})

const boxSize = computed(() => (props.compact ? Math.min(props.size, 56) : props.size))

async function doUpload(options: UploadRequestOptions) {
  uploading.value = true
  try {
    const file = options.file as File
    const next = await uploadImage(file, props.subdir || 'products')
    url.value = next
    options.onSuccess?.(next as any)
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
    options.onError?.(e as any)
  } finally {
    uploading.value = false
  }
}

function clear(e?: Event) {
  e?.stopPropagation()
  e?.preventDefault()
  url.value = ''
}
</script>

<template>
  <div class="img-field" :class="{ compact }" :style="{ '--img-size': boxSize + 'px' }">
    <el-upload
      class="uploader"
      :show-file-list="false"
      accept="image/*"
      :http-request="doUpload"
      :disabled="uploading"
    >
      <div v-if="url" class="preview">
        <img :src="url" alt="" />
        <button v-if="compact" type="button" class="clear-btn" title="清除" @click="clear">×</button>
      </div>
      <div v-else class="placeholder">
        <el-icon :size="compact ? 16 : 20"><Plus /></el-icon>
        <span v-if="!compact">{{ tip || '上传图片' }}</span>
      </div>
    </el-upload>
    <template v-if="!compact">
      <div class="actions" v-if="url">
        <el-button link type="danger" size="small" @click="clear">清除</el-button>
        <el-input v-model="url" size="small" placeholder="或填入图片 URL" />
      </div>
      <el-input v-else v-model="url" size="small" placeholder="或填入图片 URL" style="margin-top: 6px" />
    </template>
  </div>
</template>

<style scoped>
.img-field {
  width: calc(var(--img-size) + 20px);
  display: inline-flex;
  flex-direction: column;
  align-items: flex-start;
}
.img-field.compact {
  width: var(--img-size);
  vertical-align: middle;
}
.uploader :deep(.el-upload) {
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  cursor: pointer;
  overflow: hidden;
  width: var(--img-size);
  height: var(--img-size);
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
}
.preview, .placeholder {
  position: relative;
  width: var(--img-size);
  height: var(--img-size);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  color: #909399;
  font-size: 12px;
}
.preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.clear-btn {
  position: absolute;
  top: 0;
  right: 0;
  width: 18px;
  height: 18px;
  border: none;
  border-radius: 0 0 0 6px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 14px;
  line-height: 16px;
  cursor: pointer;
  padding: 0;
}
.actions { margin-top: 6px; display: flex; flex-direction: column; gap: 4px; width: 100%; }
</style>
