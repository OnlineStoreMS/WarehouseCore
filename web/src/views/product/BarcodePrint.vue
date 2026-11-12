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

function specOf(row: any): string {
  const pick = String(row.pickName || '').trim()
  if (pick) return pick
  const styles = [row.style1, row.style2, row.style3]
    .map((s) => String(s || '').trim())
    .filter(Boolean)
  if (styles.length) return styles.join(' / ')
  return String(row.productName || '').trim()
}

function toItem(row: any): SkuLabelItem {
  return {
    id: Number(row.id) || undefined,
    skuCode: String(row.skuCode || '').trim(),
    spec: specOf(row),
    productName: String(row.productName || '').trim(),
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
          <div>
            <span>条码打印</span>
            <p class="desc">勾选库存 SKU 后打印商品规格标签。标签含 SKU 条形码、编码和规格名称，版式与售后中心一致。</p>
          </div>
          <el-button type="primary" :icon="Printer" :disabled="!selected.length" @click="openPrint(selected)">
            打印规格标签{{ selected.length ? `（${selected.length}）` : '' }}
          </el-button>
        </div>
      </template>
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="SKU编码 / 配货名称 / 商品名称" clearable :prefix-icon="Search" style="width: 280px" @change="search" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" border stripe @selection-change="onSelectionChange">
        <el-table-column type="selection" width="48" />
        <el-table-column prop="skuCode" label="库存SKU" width="160" show-overflow-tooltip />
        <el-table-column prop="productName" label="商品名称" min-width="140" show-overflow-tooltip />
        <el-table-column prop="pickName" label="配货名称" min-width="160" show-overflow-tooltip />
        <el-table-column prop="style1" label="款式1" width="100" show-overflow-tooltip />
        <el-table-column prop="style2" label="款式2" width="100" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="90" />
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" :icon="Printer" @click="openPrint([row])">打标签</el-button>
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

    <SkuLabelPrintDialog v-model="printVisible" :items="printItems" />
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; }
.desc { margin: 6px 0 0; color: #909399; font-size: 12px; font-weight: 400; }
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
</style>
