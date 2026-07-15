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
const days = ref(30)

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = res.list
}

async function load() {
  loading.value = true
  try {
    const res = await api.stockSlowMoving({
      page: page.value,
      pageSize: pageSize.value,
      warehouseId: warehouseId.value,
      days: days.value,
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
      <template #header><span>滞销查询</span></template>
      <div class="toolbar">
        <el-select v-model="warehouseId" clearable placeholder="全部仓库" style="width: 200px" @change="search">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <span>闲置天数 ≥</span>
        <el-input-number v-model="days" :min="1" :max="3650" :controls="false" style="width: 100px" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" border stripe>
        <el-table-column prop="warehouseName" label="仓库" width="140" />
        <el-table-column prop="skuCode" label="库存SKU" width="140" />
        <el-table-column prop="productName" label="商品名" min-width="160" show-overflow-tooltip />
        <el-table-column prop="onHand" label="结存" width="100" align="right" />
        <el-table-column prop="lastMoveAt" label="最近移动" width="170" />
        <el-table-column prop="idleDays" label="闲置天数" width="110" align="right" />
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
