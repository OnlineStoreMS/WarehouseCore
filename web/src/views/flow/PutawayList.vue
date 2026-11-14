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
const statusTab = ref('open')
const warehouseId = ref<number | undefined>()
const keyword = ref('')
const visible = ref(false)
const current = ref<any>(null)
const form = ref({ locationCode: '', qty: 1 })

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = (res.list || []).filter((w: any) => w.status !== 0)
}

async function load() {
  loading.value = true
  try {
    const res = await api.listPutaways({
      page: page.value,
      pageSize: pageSize.value,
      status: statusTab.value,
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

function openPlace(row: any) {
  current.value = row
  form.value = { locationCode: row.suggestCode || '', qty: Math.max(row.qty - row.doneQty, 1) }
  visible.value = true
}

async function place() {
  if (!current.value) return
  if (!form.value.locationCode.trim()) {
    ElMessage.warning('请填写库位')
    return
  }
  try {
    await api.placePutaway(current.value.id, {
      locationCode: form.value.locationCode.trim(),
      barcode: current.value.skuCode,
      qty: form.value.qty,
    })
    ElMessage.success('已上架')
    visible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '上架失败')
  }
}
</script>

<template>
  <div class="page">
    <el-card v-loading="loading">
      <template #header><span>上架任务</span></template>
      <el-tabs v-model="statusTab" @tab-change="search">
        <el-tab-pane label="待上架" name="open" />
        <el-tab-pane label="已上架" name="done" />
        <el-tab-pane label="全部" name="all" />
      </el-tabs>
      <div class="toolbar">
        <el-select v-model="warehouseId" clearable filterable placeholder="仓库" style="width: 180px" @change="search">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <el-input v-model="keyword" clearable placeholder="收货单号 / SKU / 建议库位" style="width: 240px" @keyup.enter="search" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" border stripe>
        <el-table-column prop="docNo" label="收货单号" width="170" />
        <el-table-column prop="warehouseName" label="仓库" width="140" />
        <el-table-column prop="skuCode" label="库存SKU" width="140" />
        <el-table-column prop="suggestCode" label="建议库位" width="140" />
        <el-table-column prop="qty" label="待上数量" width="100" align="right" />
        <el-table-column prop="doneQty" label="已上数量" width="100" align="right" />
        <el-table-column prop="batchNo" label="批次" width="110" />
        <el-table-column prop="statusLabel" label="状态" width="90" />
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status !== 'done'" link type="primary" @click="openPlace(row)">上架</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination class="pager" background layout="total, prev, pager, next" :total="total" :page-size="pageSize" :current-page="page" @current-change="(p: number) => { page = p; load() }" />
    </el-card>

    <el-dialog v-model="visible" title="确认上架" width="420px">
      <el-form label-width="80px">
        <el-form-item label="库存SKU"><span>{{ current?.skuCode }}</span></el-form-item>
        <el-form-item label="库位">
          <el-input v-model="form.locationCode" placeholder="库区-货架-层号-格位" />
        </el-form-item>
        <el-form-item label="数量">
          <el-input-number v-model="form.qty" :min="1" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="place">确认上架</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
</style>
