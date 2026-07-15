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
const visible = ref(false)
const form = ref<any>({ warehouseId: undefined, remark: '' })

const statusMap: Record<string, string> = {
  draft: '草稿',
  counting: '盘点中',
  review: '待过账',
  posted: '已过账',
  cancelled: '已取消',
}

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = res.list
}

async function load() {
  loading.value = true
  try {
    const res = await api.listStocktakes({ page: page.value, pageSize: pageSize.value })
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

function openCreate() {
  form.value = { warehouseId: warehouses.value[0]?.id, remark: '' }
  visible.value = true
}

async function create() {
  try {
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
      <el-table :data="list" border stripe>
        <el-table-column prop="docNo" label="单号" width="160">
          <template #default="{ row }">
            <el-link type="primary" @click="router.push(`/stocktakes/${row.id}`)">{{ row.docNo }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="仓库" width="140">
          <template #default="{ row }">{{ whName(row.warehouseId) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">{{ statusMap[row.status] || row.status }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="创建时间" width="170" />
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
.pager { margin-top: 16px; justify-content: flex-end; }
</style>
