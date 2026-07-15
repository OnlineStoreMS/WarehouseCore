<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import { api } from '../../api/wms'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const warehouses = ref<any[]>([])
const warehouseId = ref<number | undefined>()
const keyword = ref('')
const skuCode = ref('')

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = res.list
}

async function load() {
  loading.value = true
  try {
    const res = await api.stockBalances({
      page: page.value,
      pageSize: pageSize.value,
      warehouseId: warehouseId.value,
      keyword: keyword.value,
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
      <template #header><span>库存查询</span></template>
      <div class="toolbar">
        <el-select v-model="warehouseId" clearable placeholder="全部仓库" style="width: 200px" @change="search">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <el-input v-model="skuCode" placeholder="SKU编码" clearable style="width: 160px" @change="search" />
        <el-input v-model="keyword" placeholder="关键词" clearable :prefix-icon="Search" style="width: 200px" @change="search" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" border stripe>
        <el-table-column prop="warehouseName" label="仓库" width="140" />
        <el-table-column prop="locationCode" label="库位" width="120" />
        <el-table-column prop="skuCode" label="SKU编码" width="140" />
        <el-table-column prop="pickName" label="拣货名" min-width="140" />
        <el-table-column prop="productName" label="商品名" min-width="140" show-overflow-tooltip />
        <el-table-column prop="onHand" label="结存" width="100" align="right" />
        <el-table-column prop="lastCost" label="最近成本" width="110" align="right" />
        <el-table-column prop="retailPrice" label="零售价" width="100" align="right" />
        <el-table-column prop="updatedAt" label="更新时间" width="170" />
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
.toolbar { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
</style>
