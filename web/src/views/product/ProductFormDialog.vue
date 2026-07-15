<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Delete } from '@element-plus/icons-vue'
import { api } from '../../api/wms'

const props = defineProps<{
  modelValue: boolean
  productId?: number | null
  defaultProductType?: string
  categories: any[]
  warehouses: any[]
  packSpecs: any[]
  presetCategoryId?: number
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'saved'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const saving = ref(false)
const activeTab = ref('base')
const form = ref<any>({})
const skus = ref<any[]>([])

const title = computed(() => {
  if (form.value.id) return '编辑商品'
  const t = props.defaultProductType || 'normal'
  if (t === 'combo') return '新增组合品'
  if (t === 'assembly') return '新增组装品(加工产品)'
  return '新增普通商品'
})

function emptySku(productType = 'normal') {
  return {
    id: undefined,
    skuCode: '',
    pickName: '',
    style1: '',
    style2: '',
    style3: '',
    status: 'active',
    lastPurchasePrice: 0,
    minPurchasePrice: 0,
    retailPrice: 0,
    weightG: 0,
    upc: '',
    asin: '',
    supplierItemNo: '',
    description: '',
    pic: '',
    productType,
  }
}

function resetCreate() {
  const today = new Date().toISOString().slice(0, 10)
  const pt = props.defaultProductType || 'normal'
  form.value = {
    parentSku: '',
    name: '',
    categoryId: props.presetCategoryId || undefined,
    packSpecId: undefined,
    scoreFactor: 1,
    remark: '',
    status: 1,
    developedAt: today,
    defaultWarehouseId: props.warehouses.find((w) => w.isDefault)?.id || undefined,
    pic: '',
    defaultProductType: pt,
  }
  skus.value = [emptySku(pt)]
  activeTab.value = 'base'
}

async function loadEdit(id: number) {
  const detail = await api.getProduct(id)
  form.value = {
    ...detail,
    developedAt: detail.developedAt ? String(detail.developedAt).slice(0, 10) : '',
    defaultProductType: detail.skus?.[0]?.productType || 'normal',
  }
  skus.value = (detail.skus || []).map((s: any) => ({ ...s }))
  if (!skus.value.length) skus.value = [emptySku(form.value.defaultProductType)]
  activeTab.value = 'base'
}

watch(
  () => props.modelValue,
  async (open) => {
    if (!open) return
    try {
      if (props.productId) {
        await loadEdit(props.productId)
      } else {
        resetCreate()
      }
    } catch (e) {
      ElMessage.error((e as Error).message || '加载失败')
      visible.value = false
    }
  },
)

function addSkuRow() {
  skus.value.push(emptySku(form.value.defaultProductType || 'normal'))
}

function removeSkuRow(idx: number) {
  if (skus.value.length <= 1) {
    ElMessage.warning('至少保留一条库存SKU')
    return
  }
  skus.value.splice(idx, 1)
}

function fillSkuFromParent() {
  if (!form.value.parentSku) return
  const first = skus.value[0]
  if (first && !first.skuCode) first.skuCode = form.value.parentSku
  if (first && !first.pickName) first.pickName = form.value.name || ''
}

async function save() {
  if (!form.value.parentSku?.trim()) {
    ElMessage.warning('请填写父SKU/主SKU')
    activeTab.value = 'base'
    return
  }
  if (!form.value.name?.trim()) {
    ElMessage.warning('请填写商品名称')
    activeTab.value = 'base'
    return
  }
  const validSkus = skus.value.filter((s) => String(s.skuCode || '').trim())
  if (!validSkus.length) {
    ElMessage.warning('请至少填写一条库存SKU')
    activeTab.value = 'base'
    return
  }
  const codes = new Set<string>()
  for (const s of validSkus) {
    const c = String(s.skuCode).trim()
    if (codes.has(c)) {
      ElMessage.warning(`库存SKU重复：${c}`)
      return
    }
    codes.add(c)
  }

  const body = {
    parentSku: form.value.parentSku.trim(),
    name: form.value.name.trim(),
    categoryId: form.value.categoryId || 0,
    packSpecId: form.value.packSpecId || 0,
    developedAt: form.value.developedAt || '',
    defaultWarehouseId: form.value.defaultWarehouseId || 0,
    scoreFactor: form.value.scoreFactor ?? 1,
    remark: form.value.remark || '',
    pic: form.value.pic || '',
    albumPics: form.value.albumPics || '',
    status: form.value.status ?? 1,
    defaultProductType: form.value.defaultProductType || 'normal',
    skus: validSkus.map((s) => ({
      id: s.id || undefined,
      skuCode: String(s.skuCode).trim(),
      pickName: s.pickName || '',
      style1: s.style1 || '',
      style2: s.style2 || '',
      style3: s.style3 || '',
      status: s.status || 'active',
      productType: s.productType || form.value.defaultProductType || 'normal',
      lastPurchasePrice: Number(s.lastPurchasePrice) || 0,
      minPurchasePrice: Number(s.minPurchasePrice) || 0,
      retailPrice: Number(s.retailPrice) || 0,
      weightG: Number(s.weightG) || 0,
      upc: s.upc || '',
      asin: s.asin || '',
      supplierItemNo: s.supplierItemNo || '',
      description: s.description || '',
      pic: s.pic || '',
    })),
  }

  saving.value = true
  try {
    if (form.value.id) {
      await api.updateProductWithSkus(form.value.id, body)
    } else {
      await api.createProductWithSkus(body)
    }
    ElMessage.success('已保存')
    visible.value = false
    emit('saved')
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="title"
    width="1100px"
    top="4vh"
    destroy-on-close
    class="product-form-dlg"
    :close-on-click-modal="false"
  >
    <el-tabs v-model="activeTab">
      <el-tab-pane label="基本信息" name="base">
        <div class="section-title">基础信息</div>
        <el-form :model="form" label-width="120px" class="base-form">
          <el-row :gutter="16">
            <el-col :span="8">
              <el-form-item label="父SKU/主SKU" required>
                <el-input v-model="form.parentSku" :disabled="!!form.id" @blur="fillSkuFromParent" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="商品类别">
                <el-select v-model="form.categoryId" clearable filterable style="width: 100%">
                  <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="开发日期">
                <el-date-picker v-model="form.developedAt" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="默认发货仓库">
                <el-select v-model="form.defaultWarehouseId" clearable filterable style="width: 100%">
                  <el-option v-for="w in warehouses" :key="w.id" :label="w.name" :value="w.id" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="评分系数">
                <el-input-number v-model="form.scoreFactor" :min="0" :step="0.1" style="width: 100%" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="状态">
                <el-select v-model="form.status" style="width: 100%">
                  <el-option :value="1" label="启用" />
                  <el-option :value="0" label="停用" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="16">
              <el-form-item label="商品名称" required>
                <el-input v-model="form.name" @blur="fillSkuFromParent" />
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item label="产品类型">
                <el-select v-model="form.defaultProductType" style="width: 100%">
                  <el-option label="普通" value="normal" />
                  <el-option label="组合品" value="combo" />
                  <el-option label="组装品" value="assembly" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="24">
              <el-form-item label="备注">
                <el-input v-model="form.remark" type="textarea" :rows="2" />
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>

        <div class="section-title sku-title">
          <span>库存SKU明细</span>
          <el-button type="primary" link :icon="Plus" @click="addSkuRow">添加</el-button>
        </div>
        <el-table :data="skus" border size="small" class="sku-grid">
          <el-table-column label="库存SKU" min-width="130" fixed>
            <template #default="{ row }">
              <el-input v-model="row.skuCode" size="small" />
            </template>
          </el-table-column>
          <el-table-column label="配货名称" min-width="120">
            <template #default="{ row }"><el-input v-model="row.pickName" size="small" /></template>
          </el-table-column>
          <el-table-column label="款式1" width="90">
            <template #default="{ row }"><el-input v-model="row.style1" size="small" /></template>
          </el-table-column>
          <el-table-column label="款式2" width="90">
            <template #default="{ row }"><el-input v-model="row.style2" size="small" /></template>
          </el-table-column>
          <el-table-column label="款式3" width="90">
            <template #default="{ row }"><el-input v-model="row.style3" size="small" /></template>
          </el-table-column>
          <el-table-column label="商品状态" width="100">
            <template #default="{ row }">
              <el-select v-model="row.status" size="small" style="width: 100%">
                <el-option label="在售" value="active" />
                <el-option label="停用" value="inactive" />
                <el-option label="清仓" value="clearance" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="上次采购价" width="110">
            <template #default="{ row }">
              <el-input-number v-model="row.lastPurchasePrice" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="零售价" width="100">
            <template #default="{ row }">
              <el-input-number v-model="row.retailPrice" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="重量(g)" width="90">
            <template #default="{ row }">
              <el-input-number v-model="row.weightG" :min="0" :controls="false" size="small" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="UPC码" width="110">
            <template #default="{ row }"><el-input v-model="row.upc" size="small" /></template>
          </el-table-column>
          <el-table-column label="ASIN码" width="110">
            <template #default="{ row }"><el-input v-model="row.asin" size="small" /></template>
          </el-table-column>
          <el-table-column label="供应商货号" width="120">
            <template #default="{ row }"><el-input v-model="row.supplierItemNo" size="small" /></template>
          </el-table-column>
          <el-table-column label="备注" min-width="100">
            <template #default="{ row }"><el-input v-model="row.description" size="small" /></template>
          </el-table-column>
          <el-table-column label="操作" width="70" fixed="right">
            <template #default="{ $index }">
              <el-button link type="danger" :icon="Delete" @click="removeSkuRow($index)" />
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="采购信息" name="purchase">
        <el-alert
          type="info"
          :closable="false"
          show-icon
          title="采购价、最低采购价按库存SKU维护，请在「基本信息 → 库存SKU明细」中填写。"
          style="margin-bottom: 12px"
        />
        <el-table :data="skus" border size="small">
          <el-table-column prop="skuCode" label="库存SKU" width="140" />
          <el-table-column label="上次采购价 / 成本" width="150">
            <template #default="{ row }">
              <el-input-number v-model="row.lastPurchasePrice" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="最低采购价" width="140">
            <template #default="{ row }">
              <el-input-number v-model="row.minPurchasePrice" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="供应商货号" min-width="140">
            <template #default="{ row }"><el-input v-model="row.supplierItemNo" size="small" /></template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="包装信息" name="pack">
        <el-form :model="form" label-width="120px" style="max-width: 520px">
          <el-form-item label="外包装规格">
            <el-select v-model="form.packSpecId" clearable filterable style="width: 100%">
              <el-option v-for="p in packSpecs" :key="p.id" :label="`${p.name}（成本 ${p.cost} / ${p.weightG}g）`" :value="p.id" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <div class="hint">包装规格主档在「商品管理 → 包装规格」维护；可绑定库存SKU与数量范围。</div>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="销售信息" name="sale">
        <el-form :model="form" label-width="120px" style="max-width: 640px; margin-bottom: 12px">
          <el-form-item label="商品图片URL">
            <el-input v-model="form.pic" placeholder="网络图片URL" />
          </el-form-item>
        </el-form>
        <el-table :data="skus" border size="small">
          <el-table-column prop="skuCode" label="库存SKU" width="140" />
          <el-table-column label="配货名称" min-width="120">
            <template #default="{ row }"><el-input v-model="row.pickName" size="small" /></template>
          </el-table-column>
          <el-table-column label="零售价" width="120">
            <template #default="{ row }">
              <el-input-number v-model="row.retailPrice" :min="0" :precision="2" :controls="false" size="small" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="重量(g)" width="110">
            <template #default="{ row }">
              <el-input-number v-model="row.weightG" :min="0" :controls="false" size="small" style="width: 100%" />
            </template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-select v-model="row.status" size="small" style="width: 100%">
                <el-option label="在售" value="active" />
                <el-option label="停用" value="inactive" />
                <el-option label="清仓" value="clearance" />
              </el-select>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.section-title {
  font-weight: 600;
  margin: 4px 0 12px;
  padding-left: 8px;
  border-left: 3px solid var(--el-color-primary);
}
.sku-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 16px;
}
.base-form { margin-bottom: 4px; }
.sku-grid :deep(.el-input-number) { width: 100%; }
.hint { color: #909399; font-size: 12px; line-height: 1.5; }
</style>
