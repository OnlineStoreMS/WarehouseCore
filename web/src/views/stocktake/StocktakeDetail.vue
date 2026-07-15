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
const warehouses = ref<any[]>([])
const locationCode = ref('')

const id = computed(() => Number(route.params.id))

const statusMap: Record<string, string> = {
  draft: '草稿',
  counting: '盘点中',
  review: '待过账',
  posted: '已过账',
  cancelled: '已作废',
}

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = res.list
}

function whName(warehouseId: number) {
  return warehouses.value.find((w) => w.id === warehouseId)?.name || warehouseId
}

async function resolveLocationCode(warehouseId: number, locationId: number) {
  if (!locationId) {
    locationCode.value = ''
    return
  }
  try {
    const res = await api.listLocations({ warehouseId, page: 1, pageSize: 200 })
    locationCode.value = res.list?.find((l: any) => l.id === locationId)?.code || String(locationId)
  } catch {
    locationCode.value = String(locationId)
  }
}

async function load() {
  loading.value = true
  try {
    detail.value = await api.getStocktake(id.value)
    items.value = (detail.value.items || []).map((i: any) => ({ ...i }))
    await resolveLocationCode(detail.value.warehouseId, detail.value.locationId)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadWarehouses()
  await load()
})
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
  await ElMessageBox.confirm('确认作废该盘点单？', '提示', { type: 'warning' })
  await api.cancelStocktake(id.value)
  ElMessage.success('已作废')
  await load()
}

const canEditCount = computed(() => ['draft', 'counting'].includes(detail.value?.status))

function diffOf(row: any) {
  const book = Number(row.bookQty) || 0
  const count = Number(row.countQty) || 0
  return count - book
}
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
            <el-button v-if="detail.status !== 'posted' && detail.status !== 'cancelled'" type="danger" @click="cancel">作废</el-button>
          </div>
        </div>
      </template>
      <el-descriptions :column="3" border class="mb">
        <el-descriptions-item label="盘点仓库">{{ whName(detail.warehouseId) }}</el-descriptions-item>
        <el-descriptions-item label="库位">{{ locationCode || (detail.locationId ? detail.locationId : '全仓') }}</el-descriptions-item>
        <el-descriptions-item label="备注">{{ detail.remark || '-' }}</el-descriptions-item>
      </el-descriptions>
      <el-table :data="items" border stripe>
        <el-table-column prop="skuCode" label="库存SKU" width="140" />
        <el-table-column prop="pickName" label="配货名称" min-width="140" show-overflow-tooltip />
        <el-table-column prop="locationCode" label="库位" width="120" />
        <el-table-column prop="bookQty" label="账存数量" width="110" align="right" />
        <el-table-column label="实盘数量" width="140">
          <template #default="{ row }">
            <el-input-number
              v-if="canEditCount"
              v-model="row.countQty"
              :min="0"
              :precision="4"
              :controls="false"
              style="width: 120px"
            />
            <span v-else>{{ row.countQty }}</span>
          </template>
        </el-table-column>
        <el-table-column label="差额" width="100" align="right">
          <template #default="{ row }">
            {{ canEditCount ? diffOf(row) : row.diffQty }}
          </template>
        </el-table-column>
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
