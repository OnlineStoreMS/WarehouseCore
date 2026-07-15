<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api } from '../../api/wms'
import SkuSearchSelect from '../../components/SkuSearchSelect.vue'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const warehouses = ref<any[]>([])
const statusTab = ref('all')
const fromWarehouseId = ref<number | undefined>()
const toWarehouseId = ref<number | undefined>()
const keyword = ref('')
const visible = ref(false)
const form = ref<any>({})
const detailVisible = ref(false)
const detail = ref<any>(null)

const statusMap: Record<string, string> = {
  draft: '草稿',
  in_transit: '在途',
  received: '已收货',
  cancelled: '已取消',
}

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = res.list
}

async function load() {
  loading.value = true
  try {
    const res = await api.listTransfers({
      page: page.value,
      pageSize: pageSize.value,
      status: statusTab.value === 'all' ? undefined : statusTab.value,
      fromWarehouseId: fromWarehouseId.value,
      toWarehouseId: toWarehouseId.value,
      keyword: keyword.value || undefined,
    })
    list.value = res.list
    total.value = res.total
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

function search() {
  page.value = 1
  load()
}

function onTabChange() {
  page.value = 1
  load()
}

function sumQty(row: any) {
  return (row.items || []).reduce((s: number, i: any) => s + (Number(i.qty) || 0), 0)
}

function whName(id: number) {
  return warehouses.value.find((w) => w.id === id)?.name || id
}

function openCreate() {
  form.value = {
    fromWarehouseId: warehouses.value[0]?.id,
    toWarehouseId: warehouses.value[1]?.id || warehouses.value[0]?.id,
    remark: '',
    items: [{ invSkuId: undefined, qty: 1 }],
  }
  visible.value = true
}

function addItem() {
  form.value.items.push({ invSkuId: undefined, qty: 1 })
}

function removeItem(idx: number) {
  form.value.items.splice(idx, 1)
}

async function create() {
  try {
    const body = {
      ...form.value,
      items: (form.value.items || []).filter((i: any) => i.invSkuId),
    }
    if (!body.fromWarehouseId || !body.toWarehouseId) {
      ElMessage.warning('请选择调出/调入仓')
      return
    }
    if (!body.items.length) {
      ElMessage.warning('请添加明细')
      return
    }
    await api.createTransfer(body)
    ElMessage.success('已创建')
    visible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '创建失败')
  }
}

async function showDetail(row: any) {
  try {
    detail.value = await api.getTransfer(row.id)
    detailVisible.value = true
  } catch (e) {
    ElMessage.error((e as Error).message || '加载明细失败')
  }
}

async function ship(row: any) {
  await ElMessageBox.confirm('确认发货？', '提示')
  await api.shipTransfer(row.id)
  ElMessage.success('已发货')
  await load()
}

async function receive(row: any) {
  await ElMessageBox.confirm('确认收货？', '提示')
  await api.receiveTransfer(row.id)
  ElMessage.success('已收货')
  await load()
}

async function cancel(row: any) {
  await ElMessageBox.confirm('确认作废？', '提示', { type: 'warning' })
  await api.cancelTransfer(row.id)
  ElMessage.success('已作废')
  await load()
}

function fmtTime(v: string) {
  if (!v) return '-'
  return String(v).replace('T', ' ').slice(0, 19)
}
</script>

<template>
  <div class="page">
    <el-card v-loading="loading">
      <template #header>
        <div class="hdr">
          <span>仓库调拨单</span>
          <el-button type="primary" :icon="Plus" @click="openCreate">新建调拨</el-button>
        </div>
      </template>

      <el-tabs v-model="statusTab" @tab-change="onTabChange">
        <el-tab-pane label="全部" name="all" />
        <el-tab-pane label="草稿" name="draft" />
        <el-tab-pane label="在途" name="in_transit" />
        <el-tab-pane label="已收货" name="received" />
        <el-tab-pane label="已取消" name="cancelled" />
      </el-tabs>

      <div class="toolbar">
        <el-select v-model="fromWarehouseId" clearable placeholder="出库仓" style="width: 160px">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <el-select v-model="toWarehouseId" clearable placeholder="入库仓" style="width: 160px">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <el-input v-model="keyword" clearable placeholder="单号" style="width: 180px" @keyup.enter="search" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>

      <el-table :data="list" border stripe>
        <el-table-column prop="docNo" label="调拨单号" width="160" />
        <el-table-column label="调出仓库" width="130">
          <template #default="{ row }">{{ whName(row.fromWarehouseId) }}</template>
        </el-table-column>
        <el-table-column label="调入仓库" width="130">
          <template #default="{ row }">{{ whName(row.toWarehouseId) }}</template>
        </el-table-column>
        <el-table-column label="总数量" width="100" align="right">
          <template #default="{ row }">{{ sumQty(row) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">{{ statusMap[row.status] || row.status }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column label="制单时间" width="170">
          <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">明细</el-button>
            <el-button v-if="row.status === 'draft'" link type="primary" @click="ship(row)">发货</el-button>
            <el-button v-if="row.status === 'in_transit'" link type="success" @click="receive(row)">收货</el-button>
            <el-button v-if="row.status === 'draft'" link type="danger" @click="cancel(row)">作废</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        class="pager"
        layout="total, prev, pager, next"
        :total="total"
        v-model:current-page="page"
        :page-size="pageSize"
        @current-change="load"
      />
    </el-card>

    <el-dialog v-model="visible" title="新建调拨单" width="720px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="调出仓" required>
          <el-select v-model="form.fromWarehouseId" style="width: 100%">
            <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="调入仓" required>
          <el-select v-model="form.toWarehouseId" style="width: 100%">
            <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
        <el-form-item label="明细">
          <el-table :data="form.items" border size="small" style="width: 100%">
            <el-table-column label="库存SKU" min-width="220">
              <template #default="{ row }">
                <SkuSearchSelect v-model="row.invSkuId" />
              </template>
            </el-table-column>
            <el-table-column label="数量" width="130">
              <template #default="{ row }">
                <el-input-number v-model="row.qty" :min="0.0001" :controls="false" style="width: 110px" />
              </template>
            </el-table-column>
            <el-table-column label="" width="70">
              <template #default="{ $index }">
                <el-button link type="danger" @click="removeItem($index)">移除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-button type="primary" link style="margin-top: 8px" @click="addItem">+ 添加</el-button>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="create">创建</el-button>
      </template>
    </el-dialog>

    <el-drawer v-model="detailVisible" title="调拨明细" size="520px">
      <template v-if="detail">
        <el-descriptions :column="1" border class="mb">
          <el-descriptions-item label="调拨单号">{{ detail.docNo }}</el-descriptions-item>
          <el-descriptions-item label="调出仓库">{{ whName(detail.fromWarehouseId) }}</el-descriptions-item>
          <el-descriptions-item label="调入仓库">{{ whName(detail.toWarehouseId) }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ statusMap[detail.status] || detail.status }}</el-descriptions-item>
          <el-descriptions-item label="备注">{{ detail.remark || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-table :data="detail.items || []" border stripe size="small">
          <el-table-column prop="skuCode" label="库存SKU" min-width="120" />
          <el-table-column prop="pickName" label="配货名称" min-width="120" show-overflow-tooltip />
          <el-table-column prop="qty" label="数量" width="90" align="right" />
        </el-table>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; }
.toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
.mb { margin-bottom: 16px; }
</style>
