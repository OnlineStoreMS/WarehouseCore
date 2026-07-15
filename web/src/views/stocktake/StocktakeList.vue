<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api } from '../../api/wms'

const router = useRouter()
const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const warehouses = ref<any[]>([])
const statusTab = ref('all')
const warehouseId = ref<number | undefined>()
const keyword = ref('')
const visible = ref(false)
const form = ref<any>({ warehouseId: undefined, remark: '' })

const statusMap: Record<string, string> = {
  draft: '草稿',
  counting: '盘点中',
  review: '待过账',
  posted: '已过账',
  cancelled: '已作废',
}

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = res.list
}

async function load() {
  loading.value = true
  try {
    const res = await api.listStocktakes({
      page: page.value,
      pageSize: pageSize.value,
      status: statusTab.value === 'all' ? undefined : statusTab.value,
      warehouseId: warehouseId.value,
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

function openCreate() {
  form.value = { warehouseId: warehouses.value[0]?.id, remark: '' }
  visible.value = true
}

async function create() {
  try {
    if (!form.value.warehouseId) {
      ElMessage.warning('请选择仓库')
      return
    }
    const { data } = await api.createStocktake(form.value)
    ElMessage.success('已创建')
    visible.value = false
    const id = data?.data?.id
    if (id) router.push(`/stocktakes/${id}`)
    else await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '创建失败')
  }
}

function whName(id: number) {
  return warehouses.value.find((w) => w.id === id)?.name || id
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
          <span>仓库盘点单</span>
          <el-button type="primary" :icon="Plus" @click="openCreate">新建盘点</el-button>
        </div>
      </template>

      <el-tabs v-model="statusTab" @tab-change="onTabChange">
        <el-tab-pane label="全部" name="all" />
        <el-tab-pane label="草稿" name="draft" />
        <el-tab-pane label="盘点中" name="counting" />
        <el-tab-pane label="待过账" name="review" />
        <el-tab-pane label="已过账" name="posted" />
        <el-tab-pane label="已作废" name="cancelled" />
      </el-tabs>

      <div class="toolbar">
        <el-select v-model="warehouseId" clearable placeholder="盘点仓库" style="width: 180px">
          <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
        </el-select>
        <el-input v-model="keyword" clearable placeholder="单号" style="width: 200px" @keyup.enter="search" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>

      <el-table :data="list" border stripe>
        <el-table-column prop="docNo" label="盘点单号" width="160">
          <template #default="{ row }">
            <el-link type="primary" @click="router.push(`/stocktakes/${row.id}`)">{{ row.docNo }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="盘点仓库" width="140">
          <template #default="{ row }">{{ whName(row.warehouseId) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">{{ statusMap[row.status] || row.status }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column label="制单时间" width="170">
          <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="router.push(`/stocktakes/${row.id}`)">详情</el-button>
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

    <el-dialog v-model="visible" title="新建盘点单" width="480px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="仓库" required>
          <el-select v-model="form.warehouseId" style="width: 100%">
            <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="create">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; }
.toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
</style>
