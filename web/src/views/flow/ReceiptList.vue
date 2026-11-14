<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../../api/wms'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const warehouses = ref<any[]>([])
const statusTab = ref('all')
const warehouseId = ref<number | undefined>()
const keyword = ref('')
const detailVisible = ref(false)
const detail = ref<any>(null)

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = (res.list || []).filter((w: any) => w.status !== 0)
}

async function load() {
  loading.value = true
  try {
    const res = await api.listReceipts({
      page: page.value,
      pageSize: pageSize.value,
      status: statusTab.value === 'all' ? undefined : statusTab.value,
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
    detail.value = await api.getReceipt(row.id)
    detailVisible.value = true
  } catch (e) {
    ElMessage.error((e as Error).message || '加载明细失败')
  }
}
</script>

<template>
  <div class="page">
    <el-card v-loading="loading">
      <template #header><span>标准入库</span></template>
      <el-tabs v-model="statusTab" @tab-change="search">
        <el-tab-pane label="全部" name="all" />
        <el-tab-pane label="待收货" name="open" />
        <el-tab-pane label="待上架" name="putaway" />
        <el-tab-pane label="已上架" name="done" />
      </el-tabs>
      <div class="toolbar">
        <el-select v-model="warehouseId" clearable filterable placeholder="仓库" style="width: 180px" @change="search">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <el-input v-model="keyword" clearable placeholder="收货单号 / 采购单号 / 物流单号" style="width: 260px" @keyup.enter="search" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" border stripe>
        <el-table-column prop="docNo" label="收货单号" width="170" />
        <el-table-column prop="poNo" label="采购单号" width="150" />
        <el-table-column prop="supplierName" label="供应商" min-width="140" show-overflow-tooltip />
        <el-table-column prop="logisticsNo" label="物流单号" width="160" show-overflow-tooltip />
        <el-table-column prop="warehouseName" label="仓库" width="140" />
        <el-table-column prop="statusLabel" label="状态" width="100" />
        <el-table-column prop="expectedQty" label="应收" width="80" align="right" />
        <el-table-column prop="receivedQty" label="实收" width="80" align="right" />
        <el-table-column prop="createdAt" label="创建时间" width="170" />
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">明细</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination class="pager" background layout="total, prev, pager, next" :total="total" :page-size="pageSize" :current-page="page" @current-change="(p: number) => { page = p; load() }" />
    </el-card>

    <el-drawer v-model="detailVisible" title="收货明细" size="640px">
      <template v-if="detail">
        <el-descriptions :column="2" border class="mb">
          <el-descriptions-item label="收货单号">{{ detail.docNo }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ detail.statusLabel }}</el-descriptions-item>
          <el-descriptions-item label="采购单号">{{ detail.poNo || '—' }}</el-descriptions-item>
          <el-descriptions-item label="物流单号">{{ detail.logisticsNo || '—' }}</el-descriptions-item>
          <el-descriptions-item label="供应商">{{ detail.supplierName || '—' }}</el-descriptions-item>
          <el-descriptions-item label="仓库">{{ detail.warehouseName || '—' }}</el-descriptions-item>
        </el-descriptions>
        <el-table :data="detail.lines || []" border stripe size="small">
          <el-table-column prop="skuCode" label="库存SKU" width="140" />
          <el-table-column prop="productName" label="商品" min-width="140" />
          <el-table-column prop="expectedQty" label="应收" width="80" align="right" />
          <el-table-column prop="receivedQty" label="实收" width="80" align="right" />
          <el-table-column prop="batchNo" label="批次" width="100" />
          <el-table-column prop="expiry" label="效期" width="110" />
        </el-table>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
.mb { margin-bottom: 16px; }
</style>
