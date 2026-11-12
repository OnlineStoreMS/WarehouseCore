<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Printer, Search } from '@element-plus/icons-vue'
import { api } from '../../api/wms'
import SkuLabelPrintDialog from '../../components/SkuLabelPrintDialog.vue'
import type { SkuLabelItem } from '../../utils/skuLabelPrint'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(50)
const keyword = ref('')
const selected = ref<any[]>([])
const printVisible = ref(false)
const printItems = ref<SkuLabelItem[]>([])

function toItem(row: any): SkuLabelItem {
  return {
    id: Number(row.id) || undefined,
    skuCode: String(row.skuCode || '').trim(),
    pickName: String(row.pickName || '').trim(),
  }
}

async function load() {
  loading.value = true
  try {
    const res = await api.listSkus({ page: page.value, pageSize: pageSize.value, keyword: keyword.value })
    list.value = res.list
    total.value = res.total
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)

function onSelectionChange(rows: any[]) {
  selected.value = rows
}

function search() {
  page.value = 1
  load()
}

function openPrint(rows: any[]) {
  if (!rows.length) {
    ElMessage.warning('请先勾选要打印的 SKU')
    return
  }
  printItems.value = rows.map(toItem)
  printVisible.value = true
}
</script>

<template>
  <div class="page">
    <el-card v-loading="loading">
      <template #header>
        <div class="hdr">
          <span>条码打印</span>
          <el-button type="primary" :icon="Printer" :disabled="!selected.length" @click="openPrint(selected)">打印</el-button>
        </div>
      </template>
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="SKU编码 / 拣货名" clearable :prefix-icon="Search" style="width: 260px" @change="search" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" border stripe @selection-change="onSelectionChange">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="skuCode" label="SKU编码" width="160" />
        <el-table-column prop="pickName" label="拣货名" min-width="160" />
        <el-table-column prop="style1" label="规格1" width="100" />
        <el-table-column prop="status" label="状态" width="90" />
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

    <SkuLabelPrintDialog v-model="printVisible" :items="printItems" />
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; }
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
</style>
