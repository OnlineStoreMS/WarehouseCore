<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../../api/wms'

const props = defineProps<{ stage?: string }>()
const outbound = props.stage === 'outbound'
const title = outbound ? '标准出库' : '拣货单'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const warehouses = ref<any[]>([])
const statusTab = ref(outbound ? 'packed' : 'all')
const warehouseId = ref<number | undefined>()
const keyword = ref('')
const detailVisible = ref(false)
const detail = ref<any>(null)
const mailNo = ref('')

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = (res.list || []).filter((w: any) => w.status !== 0)
}

async function load() {
  loading.value = true
  try {
    const res = await api.listPicks({
      page: page.value,
      pageSize: pageSize.value,
      status: statusTab.value === 'all' ? undefined : statusTab.value,
      stage: outbound ? 'outbound' : undefined,
      warehouseId: warehouseId.value,
      keyword: keyword.value || undefined,
    })
    list.value = res.list || []
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

async function showDetail(row: any) {
  try {
    detail.value = await api.getPick(row.id)
    mailNo.value = detail.value?.mailNo || ''
    detailVisible.value = true
  } catch (e) {
    ElMessage.error((e as Error).message || '加载明细失败')
  }
}

async function recheck() {
  if (!detail.value) return
  try {
    await api.recheckPick(detail.value.id)
    ElMessage.success('已重新审单')
    detailVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '审单失败')
  }
}

async function handover() {
  if (!detail.value) return
  if (!mailNo.value.trim()) {
    ElMessage.warning('请填写运单号')
    return
  }
  try {
    await api.handoverPick(detail.value.id, { mailNo: mailNo.value.trim() })
    ElMessage.success('已交接出库')
    detailVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '交接失败')
  }
}
</script>

<template>
  <div class="page">
    <el-card v-loading="loading">
      <template #header><span>{{ title }}</span></template>
      <el-tabs v-if="outbound" v-model="statusTab" @tab-change="search">
        <el-tab-pane label="待交接" name="packed" />
        <el-tab-pane label="已出库" name="done" />
        <el-tab-pane label="全部" name="all" />
      </el-tabs>
      <el-tabs v-else v-model="statusTab" @tab-change="search">
        <el-tab-pane label="全部" name="all" />
        <el-tab-pane label="待审核" name="blocked" />
        <el-tab-pane label="待拣货" name="picking" />
        <el-tab-pane label="复核中" name="reviewing" />
        <el-tab-pane label="待发货" name="packed" />
        <el-tab-pane label="已出库" name="done" />
      </el-tabs>
      <div class="toolbar">
        <el-select v-model="warehouseId" clearable filterable placeholder="仓库" style="width: 180px" @change="search">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <el-input v-model="keyword" clearable placeholder="拣货单号 / 自营单号 / 买家 / 运单号" style="width: 280px" @keyup.enter="search" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" border stripe>
        <el-table-column prop="docNo" label="拣货单号" width="170" />
        <el-table-column prop="orderNo" label="自营单号" width="150" />
        <el-table-column prop="buyer" label="买家" width="120" show-overflow-tooltip />
        <el-table-column prop="warehouseName" label="仓库" width="140" />
        <el-table-column prop="statusLabel" label="状态" width="100" />
        <el-table-column prop="qty" label="应拣" width="80" align="right" />
        <el-table-column prop="pickedQty" label="已拣" width="80" align="right" />
        <el-table-column prop="mailNo" label="运单号" width="150" show-overflow-tooltip />
        <el-table-column prop="blockReason" label="说明" min-width="160" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="创建时间" width="170" />
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">明细</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination class="pager" background layout="total, prev, pager, next" :total="total" :page-size="pageSize" :current-page="page" @current-change="(p: number) => { page = p; load() }" />
    </el-card>

    <el-drawer v-model="detailVisible" :title="title + '明细'" size="680px">
      <template v-if="detail">
        <el-descriptions :column="2" border class="mb">
          <el-descriptions-item label="拣货单号">{{ detail.docNo }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ detail.statusLabel }}</el-descriptions-item>
          <el-descriptions-item label="自营单号">{{ detail.orderNo }}</el-descriptions-item>
          <el-descriptions-item label="买家">{{ detail.buyer || '—' }}</el-descriptions-item>
          <el-descriptions-item label="地址" :span="2">{{ detail.address || '—' }}</el-descriptions-item>
          <el-descriptions-item label="说明" :span="2">{{ detail.blockReason || '—' }}</el-descriptions-item>
        </el-descriptions>
        <el-table :data="detail.lines || []" border stripe size="small" class="mb">
          <el-table-column prop="skuCode" label="库存SKU" width="140" />
          <el-table-column prop="productName" label="商品" min-width="140" />
          <el-table-column prop="suggestCode" label="库位" width="140" />
          <el-table-column prop="qty" label="应拣" width="80" align="right" />
          <el-table-column prop="pickedQty" label="已拣" width="80" align="right" />
          <el-table-column prop="checkedQty" label="已复核" width="80" align="right" />
        </el-table>
        <div v-if="detail.status === 'blocked'" class="actions">
          <el-button type="primary" @click="recheck">重新审单</el-button>
        </div>
        <div v-if="detail.status === 'packed'" class="actions">
          <el-input v-model="mailNo" placeholder="运单号" style="width: 240px" />
          <el-button type="primary" @click="handover">交接出库</el-button>
        </div>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
.mb { margin-bottom: 16px; }
.actions { display: flex; gap: 8px; align-items: center; }
</style>
