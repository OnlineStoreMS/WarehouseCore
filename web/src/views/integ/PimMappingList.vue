<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search, Connection } from '@element-plus/icons-vue'
import { api } from '../../api/wms'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')

const bindVisible = ref(false)
const binding = ref(false)

// left: product library
const pimKeyword = ref('')
const pimLoading = ref(false)
const pimProducts = ref<any[]>([])
const pimTotal = ref(0)
const pimPage = ref(1)
const pimPageSize = ref(10)
const selectedProduct = ref<any | null>(null)
const pimSkus = ref<any[]>([])
const pimSkusLoading = ref(false)
const selectedPimSku = ref<any | null>(null)

// right: warehouse skus
const whKeyword = ref('')
const whLoading = ref(false)
const whSkus = ref<any[]>([])
const whTotal = ref(0)
const whPage = ref(1)
const whPageSize = ref(10)
const selectedWhSku = ref<any | null>(null)

const canBind = computed(() => !!selectedPimSku.value?.id && !!selectedWhSku.value?.id)

function fmtTime(v?: string) {
  if (!v) return '-'
  return String(v).replace('T', ' ').slice(0, 19)
}

function specsText(specs?: Record<string, string>) {
  if (!specs || typeof specs !== 'object') return ''
  return Object.values(specs).filter(Boolean).join(' / ')
}

async function load() {
  loading.value = true
  try {
    const res = await api.listPimMappings({
      page: page.value,
      pageSize: pageSize.value,
      keyword: keyword.value.trim() || undefined,
    })
    list.value = res.list || []
    total.value = res.total || 0
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)

function search() {
  page.value = 1
  load()
}

async function loadPimProducts() {
  pimLoading.value = true
  try {
    const res = await api.listPimProducts({
      page: pimPage.value,
      pageSize: pimPageSize.value,
      keyword: pimKeyword.value.trim() || undefined,
    })
    pimProducts.value = res.list || []
    pimTotal.value = res.total || 0
  } catch (e) {
    pimProducts.value = []
    pimTotal.value = 0
    ElMessage.error((e as Error).message || '加载商品库失败')
  } finally {
    pimLoading.value = false
  }
}

async function loadWhSkus() {
  whLoading.value = true
  try {
    const res = await api.listSkus({
      page: whPage.value,
      pageSize: whPageSize.value,
      keyword: whKeyword.value.trim() || undefined,
    })
    whSkus.value = res.list || []
    whTotal.value = res.total || 0
  } catch (e) {
    whSkus.value = []
    whTotal.value = 0
    ElMessage.error((e as Error).message || '加载仓库SKU失败')
  } finally {
    whLoading.value = false
  }
}

async function onSelectProduct(row: any) {
  if (!row?.id) return
  selectedProduct.value = row
  selectedPimSku.value = null
  pimSkus.value = []
  pimSkusLoading.value = true
  try {
    const data = await api.getPimProductSkus(row.id)
    pimSkus.value = data?.skus || []
    if (pimSkus.value.length === 1) {
      selectedPimSku.value = pimSkus.value[0]
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '加载商品库SKU失败')
  } finally {
    pimSkusLoading.value = false
  }
}

function openBind() {
  bindVisible.value = true
  pimKeyword.value = ''
  whKeyword.value = ''
  pimPage.value = 1
  whPage.value = 1
  selectedProduct.value = null
  selectedPimSku.value = null
  selectedWhSku.value = null
  pimSkus.value = []
  loadPimProducts()
  loadWhSkus()
}

watch(bindVisible, (open) => {
  if (!open) {
    selectedProduct.value = null
    selectedPimSku.value = null
    selectedWhSku.value = null
  }
})

async function doBind() {
  if (!canBind.value) {
    ElMessage.warning('请左右各选一个 SKU')
    return
  }
  binding.value = true
  try {
    await api.upsertPimMapping({
      invSkuId: selectedWhSku.value.id,
      pimSkuId: selectedPimSku.value.id,
      pimSkuCode: selectedPimSku.value.skuCode || '',
      remark: selectedProduct.value
        ? `商品库：${selectedProduct.value.name || selectedProduct.value.id}`
        : '',
    })
    ElMessage.success('已绑定')
    bindVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '绑定失败')
  } finally {
    binding.value = false
  }
}

async function remove(row: any) {
  await ElMessageBox.confirm(
    `确认解除「${row.pimSkuCode || row.pimSkuId}」与「${row.invSkuCode || row.invSkuId}」的映射？`,
    '提示',
    { type: 'warning' },
  )
  await api.deletePimMapping(row.id)
  ElMessage.success('已解除')
  await load()
}

function pimRowClass({ row }: { row: any }) {
  return selectedProduct.value?.id === row.id ? 'is-selected' : ''
}
function whRowClass({ row }: { row: any }) {
  return selectedWhSku.value?.id === row.id ? 'is-selected' : ''
}
function pimSkuRowClass({ row }: { row: any }) {
  return selectedPimSku.value?.id === row.id ? 'is-selected' : ''
}
</script>

<template>
  <div class="page">
    <el-card v-loading="loading">
      <template #header>
        <div class="hdr">
          <div>
            <div class="title">商品库映射</div>
            <div class="sub">将 ProductCore 商品库 SKU 与仓储库存 SKU 一一绑定</div>
          </div>
          <el-button type="primary" :icon="Plus" @click="openBind">新建关联</el-button>
        </div>
      </template>

      <div class="toolbar">
        <el-input
          v-model="keyword"
          clearable
          placeholder="商品库SKU / 仓库SKU / 备注"
          :prefix-icon="Search"
          style="width: 280px"
          @keyup.enter="search"
          @clear="search"
        />
        <el-button type="primary" @click="search">查询</el-button>
      </div>

      <el-table :data="list" border stripe>
        <el-table-column label="商品库 SKU" min-width="180">
          <template #default="{ row }">
            <div class="cell-main">{{ row.pimSkuCode || '-' }}</div>
            <div class="cell-sub">ID {{ row.pimSkuId }}</div>
          </template>
        </el-table-column>
        <el-table-column label="仓库库存 SKU" min-width="200">
          <template #default="{ row }">
            <div class="cell-main">{{ row.invSkuCode || `ID ${row.invSkuId}` }}</div>
            <div class="cell-sub">{{ row.pickName || row.productName || '-' }}</div>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">{{ fmtTime(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" @click="remove(row)">解除</el-button>
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

    <el-dialog
      v-model="bindVisible"
      title="新建商品库关联"
      width="1120px"
      top="4vh"
      destroy-on-close
      :close-on-click-modal="false"
    >
      <div class="bind-layout">
        <!-- 左：商品库 -->
        <div class="panel">
          <div class="panel-hd">商品库（ProductCore）</div>
          <div class="panel-toolbar">
            <el-input
              v-model="pimKeyword"
              clearable
              size="small"
              placeholder="搜索商品名称/货号"
              :prefix-icon="Search"
              @keyup.enter="() => { pimPage = 1; loadPimProducts() }"
              @clear="() => { pimPage = 1; loadPimProducts() }"
            />
            <el-button size="small" type="primary" @click="() => { pimPage = 1; loadPimProducts() }">查询</el-button>
          </div>
          <el-table
            v-loading="pimLoading"
            :data="pimProducts"
            border
            size="small"
            height="240"
            highlight-current-row
            :row-class-name="pimRowClass"
            @row-click="onSelectProduct"
          >
            <el-table-column prop="name" label="商品" min-width="140" show-overflow-tooltip />
            <el-table-column prop="productSn" label="货号" width="110" show-overflow-tooltip />
            <el-table-column prop="skuCount" label="SKU数" width="70" align="center" />
          </el-table>
          <el-pagination
            class="mini-pager"
            small
            layout="total, prev, pager, next"
            :total="pimTotal"
            v-model:current-page="pimPage"
            :page-size="pimPageSize"
            @current-change="loadPimProducts"
          />

          <div class="sku-block">
            <div class="sku-title">
              商品库 SKU
              <span v-if="selectedProduct" class="muted">（{{ selectedProduct.name }}）</span>
              <span v-else class="muted">— 请先点选上方商品</span>
            </div>
            <el-table
              v-loading="pimSkusLoading"
              :data="pimSkus"
              border
              size="small"
              height="200"
              highlight-current-row
              :row-class-name="pimSkuRowClass"
              @row-click="(row: any) => { selectedPimSku = row }"
            >
              <el-table-column width="42" align="center">
                <template #default="{ row }">
                  <el-radio :model-value="selectedPimSku?.id" :value="row.id" @change="selectedPimSku = row" />
                </template>
              </el-table-column>
              <el-table-column prop="skuCode" label="SKU编码" min-width="120" show-overflow-tooltip />
              <el-table-column label="规格" min-width="120" show-overflow-tooltip>
                <template #default="{ row }">{{ specsText(row.specs) || '-' }}</template>
              </el-table-column>
            </el-table>
          </div>
        </div>

        <!-- 中：绑定 -->
        <div class="mid">
          <div class="mid-card">
            <div class="mid-label">商品库 SKU</div>
            <div class="mid-value">{{ selectedPimSku?.skuCode || '未选择' }}</div>
            <el-icon class="mid-icon" :size="28"><Connection /></el-icon>
            <div class="mid-label">仓库 SKU</div>
            <div class="mid-value">{{ selectedWhSku?.skuCode || '未选择' }}</div>
            <el-button
              type="primary"
              class="bind-btn"
              :disabled="!canBind"
              :loading="binding"
              @click="doBind"
            >
              绑定
            </el-button>
          </div>
        </div>

        <!-- 右：仓库 SKU -->
        <div class="panel">
          <div class="panel-hd">仓库商品（库存 SKU）</div>
          <div class="panel-toolbar">
            <el-input
              v-model="whKeyword"
              clearable
              size="small"
              placeholder="搜索库存SKU/配货名"
              :prefix-icon="Search"
              @keyup.enter="() => { whPage = 1; loadWhSkus() }"
              @clear="() => { whPage = 1; loadWhSkus() }"
            />
            <el-button size="small" type="primary" @click="() => { whPage = 1; loadWhSkus() }">查询</el-button>
          </div>
          <el-table
            v-loading="whLoading"
            :data="whSkus"
            border
            size="small"
            height="460"
            highlight-current-row
            :row-class-name="whRowClass"
            @row-click="(row: any) => { selectedWhSku = row }"
          >
            <el-table-column width="42" align="center">
              <template #default="{ row }">
                <el-radio :model-value="selectedWhSku?.id" :value="row.id" @change="selectedWhSku = row" />
              </template>
            </el-table-column>
            <el-table-column prop="skuCode" label="库存SKU" width="130" show-overflow-tooltip />
            <el-table-column prop="pickName" label="配货名称" min-width="120" show-overflow-tooltip />
            <el-table-column prop="productName" label="商品名称" min-width="120" show-overflow-tooltip />
          </el-table>
          <el-pagination
            class="mini-pager"
            small
            layout="total, prev, pager, next"
            :total="whTotal"
            v-model:current-page="whPage"
            :page-size="whPageSize"
            @current-change="loadWhSkus"
          />
        </div>
      </div>

      <template #footer>
        <el-button @click="bindVisible = false">关闭</el-button>
        <el-button type="primary" :disabled="!canBind" :loading="binding" @click="doBind">确认绑定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.title { font-weight: 600; font-size: 16px; }
.sub { margin-top: 4px; font-size: 12px; color: #909399; }
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
.cell-main { font-weight: 500; }
.cell-sub { font-size: 12px; color: #909399; margin-top: 2px; }

.bind-layout {
  display: grid;
  grid-template-columns: 1fr 160px 1fr;
  gap: 12px;
  min-height: 520px;
}
.panel {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 12px;
  background: #fafbfc;
  min-width: 0;
}
.panel-hd {
  font-weight: 600;
  margin-bottom: 10px;
  color: #303133;
}
.panel-toolbar {
  display: flex;
  gap: 6px;
  margin-bottom: 8px;
}
.mini-pager {
  margin-top: 8px;
  justify-content: flex-end;
}
.sku-block { margin-top: 12px; }
.sku-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 6px;
}
.muted { color: #909399; font-weight: 400; }

.mid {
  display: flex;
  align-items: center;
  justify-content: center;
}
.mid-card {
  width: 100%;
  text-align: center;
  padding: 16px 10px;
  border: 1px dashed var(--el-border-color);
  border-radius: 10px;
  background: #fff;
}
.mid-label { font-size: 12px; color: #909399; }
.mid-value {
  font-size: 13px;
  font-weight: 600;
  margin: 4px 0 10px;
  word-break: break-all;
  color: #303133;
  min-height: 20px;
}
.mid-icon { color: var(--el-color-primary); margin: 4px 0 10px; }
.bind-btn { width: 100%; margin-top: 4px; }

:deep(.is-selected) > td {
  background: var(--el-color-primary-light-9) !important;
}
</style>
