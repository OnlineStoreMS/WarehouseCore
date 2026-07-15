<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../../api/wms'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const detail = ref<any>(null)
const items = ref<any[]>([])

const id = computed(() => Number(route.params.id))

const statusMap: Record<string, string> = {
  draft: '草稿',
  counting: '盘点中',
  review: '待过账',
  posted: '已过账',
  cancelled: '已取消',
}

async function load() {
  loading.value = true
  try {
    detail.value = await api.getStocktake(id.value)
    items.value = (detail.value.items || []).map((i: any) => ({ ...i }))
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
watch(id, load)

async function start() {
  await api.startStocktake(id.value)
  ElMessage.success('已开始盘点')
  await load()
}

async function submitCount() {
  await api.countStocktake(id.value, {
    items: items.value.map((i) => ({ id: i.id, countQty: Number(i.countQty) || 0, remark: i.remark || '' })),
  })
  ElMessage.success('已提交盘点数量')
  await load()
}

async function post() {
  await ElMessageBox.confirm('确认过账？将按盘点差异调整库存。', '提示', { type: 'warning' })
  await api.postStocktake(id.value)
  ElMessage.success('已过账')
  await load()
}

async function cancel() {
  await ElMessageBox.confirm('确认取消该盘点单？', '提示', { type: 'warning' })
  await api.cancelStocktake(id.value)
  ElMessage.success('已取消')
  await load()
}

const canEditCount = computed(() => ['draft', 'counting'].includes(detail.value?.status))
</script>

<template>
  <div class="page" v-loading="loading">
    <el-card v-if="detail">
      <template #header>
        <div class="hdr">
          <div>
            <el-button link type="primary" @click="router.push('/stocktakes')">← 返回</el-button>
            <span class="title">盘点单 {{ detail.docNo }}</span>
            <el-tag size="small" style="margin-left: 8px">{{ statusMap[detail.status] || detail.status }}</el-tag>
          </div>
          <div class="actions">
            <el-button v-if="detail.status === 'draft'" type="primary" @click="start">开始盘点</el-button>
            <el-button v-if="canEditCount" type="success" @click="submitCount">提交盘点</el-button>
            <el-button v-if="detail.status === 'counting' || detail.status === 'review'" type="warning" @click="post">过账</el-button>
            <el-button v-if="detail.status !== 'posted' && detail.status !== 'cancelled'" type="danger" @click="cancel">取消</el-button>
          </div>
        </div>
      </template>
      <el-descriptions :column="3" border class="mb">
        <el-descriptions-item label="仓库ID">{{ detail.warehouseId }}</el-descriptions-item>
        <el-descriptions-item label="库位ID">{{ detail.locationId || '全仓' }}</el-descriptions-item>
        <el-descriptions-item label="备注">{{ detail.remark || '-' }}</el-descriptions-item>
      </el-descriptions>
      <el-table :data="items" border stripe>
        <el-table-column prop="invSkuId" label="SKU ID" width="100" />
        <el-table-column prop="locationId" label="库位ID" width="100" />
        <el-table-column prop="bookQty" label="账面数" width="110" align="right" />
        <el-table-column label="实盘数" width="140">
          <template #default="{ row }">
            <el-input-number
              v-if="canEditCount"
              v-model="row.countQty"
              :min="0"
              :precision="4"
              controls-position="right"
              style="width: 120px"
            />
            <span v-else>{{ row.countQty }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="diffQty" label="差异" width="100" align="right" />
        <el-table-column prop="remark" label="备注" min-width="140">
          <template #default="{ row }">
            <el-input v-if="canEditCount" v-model="row.remark" size="small" />
            <span v-else>{{ row.remark || '-' }}</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; }
.title { font-weight: 600; margin-left: 4px; }
.actions { display: flex; gap: 8px; }
.mb { margin-bottom: 16px; }
</style>
