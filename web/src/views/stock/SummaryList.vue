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
const warehouseId = ref<number | undefined>()
const skuCode = ref('')

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = res.list
}

async function load() {
  loading.value = true
  try {
    const res = await api.stockSummary({
      page: page.value,
      pageSize: pageSize.value,
      warehouseId: warehouseId.value,
      skuCode: skuCode.value,
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
</script>

<template>
  <div class="page">
    <el-card v-loading="loading">
      <template #header><span>库存汇总账</span></template>
      <div class="toolbar">
        <el-select v-model="warehouseId" clearable placeholder="全部仓库" style="width: 200px" @change="search">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <el-input v-model="skuCode" placeholder="SKU编码" clearable style="width: 160px" @change="search" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" border stripe>
        <el-table-column prop="warehouseName" label="仓库名称" width="140" />
        <el-table-column prop="skuCode" label="库存SKU" width="140" />
        <el-table-column label="配货/商品名" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ row.pickName || row.productName || '-' }}</template>
        </el-table-column>
        <el-table-column prop="opening" label="期初数量" width="110" align="right" />
        <el-table-column prop="inbound" label="入库数量" width="110" align="right" />
        <el-table-column prop="outbound" label="出库数量" width="110" align="right" />
        <el-table-column prop="closing" label="期末数量" width="110" align="right" />
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
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
</style>
