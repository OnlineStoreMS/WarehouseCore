<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api } from '../../api/wms'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const warehouses = ref<any[]>([])
const warehouseId = ref<number | undefined>()
const visible = ref(false)
const form = ref<any>({})

async function loadWarehouses() {
  const res = await api.listWarehouses({ page: 1, pageSize: 200 })
  warehouses.value = res.list
  if (!warehouseId.value && res.list.length) {
    warehouseId.value = res.list[0].id
  }
}

async function load() {
  if (!warehouseId.value) {
    list.value = []
    total.value = 0
    return
  }
  loading.value = true
  try {
    const res = await api.listLocations({
      page: page.value,
      pageSize: pageSize.value,
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

watch(warehouseId, () => {
  page.value = 1
  load()
})

function openCreate() {
  form.value = {
    warehouseId: warehouseId.value,
    code: '',
    zone: '',
    aisle: '',
    shelf: '',
    bin: '',
    status: 1,
  }
  visible.value = true
}

function openEdit(row: any) {
  form.value = { ...row }
  visible.value = true
}

async function save() {
  try {
    if (form.value.id) {
      await api.updateLocation(form.value.id, form.value)
    } else {
      await api.createLocation(form.value)
    }
    ElMessage.success('已保存')
    visible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function remove(row: any) {
  await ElMessageBox.confirm(`确认删除库位「${row.code}」？`, '提示', { type: 'warning' })
  await api.deleteLocation(row.id)
  ElMessage.success('已删除')
  await load()
}
</script>

<template>
  <div class="page">
    <el-card v-loading="loading">
      <template #header>
        <div class="hdr">
          <span>库位管理</span>
          <el-button type="primary" :icon="Plus" :disabled="!warehouseId" @click="openCreate">新建库位</el-button>
        </div>
      </template>
      <div class="toolbar">
        <span>仓库</span>
        <el-select v-model="warehouseId" placeholder="选择仓库" style="width: 220px" filterable>
          <el-option v-for="w in warehouses" :key="w.id" :label="`${w.code} - ${w.name}`" :value="w.id" />
        </el-select>
      </div>
      <el-table :data="list" border stripe>
        <el-table-column prop="code" label="库位编码" width="140" />
        <el-table-column prop="zone" label="库区" width="100" />
        <el-table-column prop="aisle" label="巷道" width="100" />
        <el-table-column prop="shelf" label="货架" width="100" />
        <el-table-column prop="bin" label="格位" width="100" />
        <el-table-column label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="remove(row)">删除</el-button>
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

    <el-dialog v-model="visible" :title="form.id ? '编辑库位' : '新建库位'" width="520px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="仓库">
          <el-select v-model="form.warehouseId" style="width: 100%">
            <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="编码" required><el-input v-model="form.code" /></el-form-item>
        <el-form-item label="库区"><el-input v-model="form.zone" /></el-form-item>
        <el-form-item label="巷道"><el-input v-model="form.aisle" /></el-form-item>
        <el-form-item label="货架"><el-input v-model="form.shelf" /></el-form-item>
        <el-form-item label="格位"><el-input v-model="form.bin" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; }
.toolbar { display: flex; gap: 8px; align-items: center; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
</style>
