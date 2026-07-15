<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage, type UploadRequestOptions } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { uploadImage } from '../api/upload'

const props = defineProps<{
  modelValue?: string
  subdir?: string
  tip?: string
}>()
const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()

const uploading = ref(false)
const url = computed({
  get: () => props.modelValue || '',
  set: (v) => emit('update:modelValue', v),
})

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

function clear() {
  url.value = ''
}
</script>

<template>
  <div class="img-field">
    <el-upload
      class="uploader"
      :show-file-list="false"
      accept="image/*"
      :http-request="doUpload"
      :disabled="uploading"
    >
      <div v-if="url" class="preview">
        <img :src="url" alt="" />
      </div>
      <div v-else class="placeholder">
        <el-icon><Plus /></el-icon>
        <span>{{ tip || '上传图片' }}</span>
      </div>
    </el-upload>
    <div class="actions" v-if="url">
      <el-button link type="danger" size="small" @click="clear">清除</el-button>
      <el-input v-model="url" size="small" placeholder="或填入图片 URL" />
    </div>
    <el-input v-else v-model="url" size="small" placeholder="或填入图片 URL" style="margin-top: 6px" />
  </div>
</template>

<style scoped>
.img-field { width: 140px; }
.uploader :deep(.el-upload) {
  border: 1px dashed var(--el-border-color);
  border-radius: 6px;
  cursor: pointer;
  overflow: hidden;
  width: 120px;
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.preview, .placeholder {
  width: 120px;
  height: 120px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  color: #909399;
  font-size: 12px;
}
.preview img { width: 100%; height: 100%; object-fit: cover; }
.actions { margin-top: 6px; display: flex; flex-direction: column; gap: 4px; }
</style>
