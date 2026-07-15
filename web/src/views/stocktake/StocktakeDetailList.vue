<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../../api/wms'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
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
      keyword: keyword.value || undefined,
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
        <el-select v-model="warehouseId" clearable placeholder="全部仓库" style="width: 180px">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <el-input
          v-model="keyword"
          clearable
          placeholder="单号/SKU"
          style="width: 220px"
          @keyup.enter="search"
        />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" border stripe>
        <el-table-column prop="docNo" label="盘点单号" width="150" />
        <el-table-column prop="warehouseName" label="仓库" width="140" />
        <el-table-column prop="skuCode" label="库存SKU" width="140" />
        <el-table-column prop="pickName" label="配货名称" min-width="140" show-overflow-tooltip />
        <el-table-column prop="locationCode" label="库位" width="120" />
        <el-table-column prop="bookQty" label="账存" width="100" align="right" />
        <el-table-column prop="countQty" label="实盘" width="100" align="right" />
        <el-table-column prop="diffQty" label="差额" width="100" align="right" />
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
