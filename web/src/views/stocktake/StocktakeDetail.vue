<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api } from '../../api/wms'
import SkuSearchSelect from '../../components/SkuSearchSelect.vue'
import LocationSelect from '../../components/LocationSelect.vue'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const detail = ref<any>(null)
const items = ref<any[]>([])
const warehouses = ref<any[]>([])
const locationCode = ref('')
const savingHeader = ref(false)

const addSkuId = ref<number | null>(null)
const addLocationId = ref<number | null>(null)
const addCountQty = ref<number | undefined>(undefined)
const adding = ref(false)

const id = computed(() => Number(route.params.id))

const statusMap: Record<string, string> = {
  draft: '未审核',
  counting: '盘点中',
  review: '已审核/已盘点',
  posted: '已完结',
  cancelled: '已作废',
}

const canEditCount = computed(() => ['draft', 'counting'].includes(detail.value?.status))
const canDelete = computed(() => detail.value && detail.value.status !== 'posted')
const canEditHeader = computed(() => ['draft', 'counting'].includes(detail.value?.status))

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

async function saveHeader() {
  if (!detail.value) return
  savingHeader.value = true
  try {
    await api.updateStocktake(id.value, {
      checkerName: detail.value.checkerName || '',
      remark: detail.value.remark || '',
    })
    ElMessage.success('已保存')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    savingHeader.value = false
  }
}

async function addItem() {
  if (!addSkuId.value) {
    ElMessage.warning('请选择库存SKU')
    return
  }
  adding.value = true
  try {
    const body: any = {
      items: [{
        invSkuId: addSkuId.value,
        locationId: addLocationId.value || 0,
        remark: '',
      }],
    }
    if (addCountQty.value != null && !Number.isNaN(Number(addCountQty.value))) {
      body.items[0].countQty = Number(addCountQty.value)
    }
    await api.addStocktakeItems(id.value, body)
    ElMessage.success('已添加商品')
    addSkuId.value = null
    addCountQty.value = undefined
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '添加失败')
  } finally {
    adding.value = false
  }
}

async function removeItem(row: any) {
  await ElMessageBox.confirm(`删除商品 ${row.skuCode}？`, '提示', { type: 'warning' })
  await api.deleteStocktakeItem(id.value, row.id)
  ElMessage.success('已删除')
  await load()
}

async function start() {
  await api.startStocktake(id.value)
  ElMessage.success('已开始盘点')
  await load()
}

async function submitCount() {
  if (!items.value.length) {
    ElMessage.warning('请先添加盘点商品')
    return
  }
  await api.countStocktake(id.value, {
    items: items.value.map((i) => ({ id: i.id, countQty: Number(i.countQty) || 0, remark: i.remark || '' })),
  })
  ElMessage.success('已保存盘点数量')
  await load()
}

async function post() {
  if (!items.value.length) {
    ElMessage.warning('盘点明细为空，无法过账')
    return
  }
  await ElMessageBox.confirm('确认仓库审核过账？将按盘点差异调整库存。', '仓库审核', { type: 'warning' })
  if (canEditCount.value) {
    await api.countStocktake(id.value, {
      items: items.value.map((i) => ({ id: i.id, countQty: Number(i.countQty) || 0, remark: i.remark || '' })),
    })
  }
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

async function remove() {
  await ElMessageBox.confirm('确认删除该盘点单？删除后不可恢复。', '删除确认', { type: 'warning' })
  await api.deleteStocktake(id.value)
  ElMessage.success('已删除')
  router.push('/stocktakes')
}

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
            <el-button v-if="canEditHeader" :loading="savingHeader" @click="saveHeader">保存</el-button>
            <el-button v-if="detail.status === 'draft'" type="primary" @click="start">开始盘点</el-button>
            <el-button v-if="canEditCount" type="success" @click="submitCount">保存盘点数量</el-button>
            <el-button
              v-if="detail.status === 'draft' || detail.status === 'counting' || detail.status === 'review'"
              type="warning"
              @click="post"
            >
              保存并仓库审核
            </el-button>
            <el-button v-if="detail.status !== 'posted' && detail.status !== 'cancelled'" type="danger" @click="cancel">作废</el-button>
            <el-button v-if="canDelete" type="danger" plain @click="remove">删除</el-button>
          </div>
        </div>
      </template>

      <el-alert
        class="mb"
        type="info"
        :closable="false"
        show-icon
        title="① 编辑盘点单（仅未审核/盘点中可编辑）→ ② 仓库审核后库存数量就会变化 → ③ 完结"
      />

      <el-descriptions :column="3" border class="mb">
        <el-descriptions-item label="盘点仓库">{{ detail.warehouseName || whName(detail.warehouseId) }}</el-descriptions-item>
        <el-descriptions-item label="库位范围">{{ locationCode || (detail.locationId ? detail.locationId : '不限') }}</el-descriptions-item>
        <el-descriptions-item label="盘点人">
          <el-input v-if="canEditHeader" v-model="detail.checkerName" size="small" style="max-width: 180px" />
          <span v-else>{{ detail.checkerName || '-' }}</span>
        </el-descriptions-item>
        <el-descriptions-item label="备注" :span="3">
          <el-input v-if="canEditHeader" v-model="detail.remark" type="textarea" :rows="2" />
          <span v-else>{{ detail.remark || '-' }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <div v-if="canEditCount" class="add-bar mb">
        <span class="add-label">添加商品</span>
        <SkuSearchSelect v-model="addSkuId" placeholder="库存SKU" style="width: 260px" />
        <LocationSelect v-model="addLocationId" :warehouse-id="detail.warehouseId" placeholder="库位(可空=默认)" style="width: 180px" />
        <el-input-number
          v-model="addCountQty"
          :min="0"
          :controls="false"
          placeholder="实盘数量"
          style="width: 120px"
        />
        <el-button type="primary" :icon="Plus" :loading="adding" @click="addItem">添加商品</el-button>
      </div>

      <el-table :data="items" border stripe>
        <el-table-column prop="skuCode" label="库存SKU" width="140" />
        <el-table-column prop="pickName" label="配货名称" min-width="140" show-overflow-tooltip />
        <el-table-column prop="locationCode" label="库位" width="100" />
        <el-table-column prop="bookQty" label="账存数量" width="100" align="right" />
        <el-table-column label="账存金额" width="100" align="right">
          <template #default="{ row }">{{ Number(row.bookAmount || 0).toFixed(2) }}</template>
        </el-table-column>
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
        <el-table-column label="差额数量" width="100" align="right">
          <template #default="{ row }">
            {{ canEditCount ? diffOf(row) : row.diffQty }}
          </template>
        </el-table-column>
        <el-table-column label="差额金额" width="100" align="right">
          <template #default="{ row }">
            {{ ((canEditCount ? diffOf(row) : Number(row.diffQty) || 0) * (Number(row.unitCost) || 0)).toFixed(2) }}
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="140">
          <template #default="{ row }">
            <el-input v-if="canEditCount" v-model="row.remark" size="small" />
            <span v-else>{{ row.remark || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column v-if="canEditCount" label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" @click="removeItem(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!items.length" class="empty-hint">暂无商品，请点击「添加商品」录入盘点明细（对齐普源）</div>
    </el-card>
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 8px; }
.title { font-weight: 600; margin-left: 4px; }
.actions { display: flex; gap: 8px; flex-wrap: wrap; }
.mb { margin-bottom: 16px; }
.add-bar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.add-label { font-weight: 500; color: #606266; }
.empty-hint { margin-top: 16px; color: #909399; text-align: center; }
</style>
