<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../../api/wms'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const stocktakeId = ref<number | undefined>()
const warehouseId = ref<number | undefined>()
const warehouses = ref<any[]>([])

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = res.list
}

async function load() {
  loading.value = true
  try {
    const res = await api.listStocktakeDetails({
      page: page.value,
      pageSize: pageSize.value,
      stocktakeId: stocktakeId.value,
      warehouseId: warehouseId.value,
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
      <template #header><span>盘点明细表</span></template>
      <div class="toolbar">
        <el-select v-model="warehouseId" clearable placeholder="全部仓库" style="width: 180px" @change="search">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <el-input-number v-model="stocktakeId" :min="1" placeholder="盘点单ID" controls-position="right" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" border stripe>
        <el-table-column prop="orderId" label="盘点单ID" width="110" />
        <el-table-column prop="docNo" label="单号" width="150" />
        <el-table-column prop="warehouseId" label="仓库ID" width="100" />
        <el-table-column prop="locationId" label="库位ID" width="100" />
        <el-table-column prop="invSkuId" label="SKU ID" width="100" />
        <el-table-column prop="skuCode" label="SKU编码" width="140" />
        <el-table-column prop="bookQty" label="账面" width="100" align="right" />
        <el-table-column prop="countQty" label="实盘" width="100" align="right" />
        <el-table-column prop="diffQty" label="差异" width="100" align="right" />
        <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
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
.toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
</style>
