<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { api } from '../../api/wms'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const warehouses = ref<any[]>([])
const visible = ref(false)
const form = ref<any>({})

const statusMap: Record<string, string> = {
  draft: '草稿',
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
    const res = await api.listOtherOut({ page: page.value, pageSize: pageSize.value })
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
  form.value = {
    warehouseId: warehouses.value[0]?.id,
    reason: 'adjust',
    remark: '',
    items: [{ invSkuId: undefined, qty: 1 }],
  }
  visible.value = true
}

function addItem() {
  form.value.items.push({ invSkuId: undefined, qty: 1 })
}

function removeItem(idx: number) {
  form.value.items.splice(idx, 1)
}

async function create() {
  try {
    await api.createOtherOut(form.value)
    ElMessage.success('已创建')
    visible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '创建失败')
  }
}

async function post(row: any) {
  await ElMessageBox.confirm('确认过账出库？', '提示')
  await api.postOtherOut(row.id)
  ElMessage.success('已过账')
  await load()
}

async function cancel(row: any) {
  await ElMessageBox.confirm('确认取消？', '提示', { type: 'warning' })
  await api.cancelOtherOut(row.id)
  ElMessage.success('已取消')
  await load()
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
          <span>其他出库单</span>
          <el-button type="primary" :icon="Plus" @click="openCreate">新建出库</el-button>
        </div>
      </template>
      <el-table :data="list" border stripe>
        <el-table-column prop="docNo" label="单号" width="160" />
        <el-table-column label="仓库" width="140">
          <template #default="{ row }">{{ whName(row.warehouseId) }}</template>
        </el-table-column>
        <el-table-column prop="reason" label="原因" width="110" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">{{ statusMap[row.status] || row.status }}</template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
        <el-table-column prop="createdAt" label="创建时间" width="170" />
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button v-if="row.status === 'draft'" link type="primary" @click="post(row)">过账</el-button>
            <el-button v-if="row.status === 'draft'" link type="danger" @click="cancel(row)">取消</el-button>
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

    <el-dialog v-model="visible" title="新建其他出库" width="640px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="仓库" required>
          <el-select v-model="form.warehouseId" style="width: 100%">
            <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="原因">
          <el-select v-model="form.reason" style="width: 100%">
            <el-option label="报损" value="damage" />
            <el-option label="样品" value="sample" />
            <el-option label="领用" value="usage" />
            <el-option label="调整" value="adjust" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" /></el-form-item>
        <el-form-item label="明细">
          <div class="items">
            <div v-for="(item, idx) in form.items" :key="idx" class="item-row">
              <el-input-number v-model="item.invSkuId" :min="1" placeholder="SKU ID" controls-position="right" />
              <el-input-number v-model="item.qty" :min="0.0001" placeholder="数量" controls-position="right" />
              <el-button link type="danger" @click="removeItem(idx)">移除</el-button>
            </div>
            <el-button type="primary" link @click="addItem">+ 添加明细</el-button>
          </div>
        </el-form-item>
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
.items { width: 100%; display: flex; flex-direction: column; gap: 8px; }
.item-row { display: flex; gap: 8px; align-items: center; }
</style>
