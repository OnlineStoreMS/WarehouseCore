<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search } from '@element-plus/icons-vue'
import { api } from '../../api/wms'

const loading = ref(false)
const list = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')

const productVisible = ref(false)
const productForm = ref<any>({})
const skuVisible = ref(false)
const skuForm = ref<any>({})
const categories = ref<any[]>([])

async function load() {
  loading.value = true
  try {
    const res = await api.listProducts({ page: page.value, pageSize: pageSize.value, keyword: keyword.value })
    list.value = res.list
    total.value = res.total
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadCategories() {
  try {
    const res = await api.listCategories({ page: 1, pageSize: 200 })
    categories.value = res.list
  } catch { /* ignore */ }
}

onMounted(() => {
  load()
  loadCategories()
})

function openCreateProduct() {
  productForm.value = { parentSku: '', name: '', categoryId: undefined, scoreFactor: 1, remark: '', status: 1 }
  productVisible.value = true
}

function openEditProduct(row: any) {
  productForm.value = { ...row }
  productVisible.value = true
}

async function saveProduct() {
  try {
    if (productForm.value.id) {
      await api.updateProduct(productForm.value.id, productForm.value)
    } else {
      await api.createProduct(productForm.value)
    }
    ElMessage.success('已保存')
    productVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function removeProduct(row: any) {
  await ElMessageBox.confirm(`确认删除商品「${row.name}」？`, '提示', { type: 'warning' })
  await api.deleteProduct(row.id)
  ElMessage.success('已删除')
  await load()
}

function openCreateSku(parent: any) {
  skuForm.value = {
    parentId: parent.id,
    skuCode: '',
    pickName: '',
    style1: '',
    style2: '',
    style3: '',
    weightG: 0,
    lastPurchasePrice: 0,
    minPurchasePrice: 0,
    retailPrice: 0,
    upc: '',
    asin: '',
    productType: 'normal',
    status: 'active',
  }
  skuVisible.value = true
}

function openEditSku(sku: any) {
  skuForm.value = { ...sku }
  skuVisible.value = true
}

async function saveSku() {
  try {
    if (skuForm.value.id) {
      await api.updateSku(skuForm.value.id, skuForm.value)
    } else {
      await api.createSku(skuForm.value)
    }
    ElMessage.success('已保存')
    skuVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function removeSku(sku: any) {
  await ElMessageBox.confirm(`确认删除 SKU「${sku.skuCode}」？`, '提示', { type: 'warning' })
  await api.deleteSku(sku.id)
  ElMessage.success('已删除')
  await load()
}

function search() {
  page.value = 1
  load()
}
</script>

<template>
  <div class="page">
    <el-card v-loading="loading">
      <template #header>
        <div class="hdr">
          <span>商品资料</span>
          <el-button type="primary" :icon="Plus" @click="openCreateProduct">新建商品</el-button>
        </div>
      </template>
      <div class="toolbar">
        <el-input v-model="keyword" placeholder="父SKU / 名称" clearable :prefix-icon="Search" style="width: 260px" @change="search" />
        <el-button type="primary" @click="search">查询</el-button>
      </div>
      <el-table :data="list" row-key="id" border stripe>
        <el-table-column type="expand">
          <template #default="{ row }">
            <div class="sku-wrap">
              <div class="sku-hdr">
                <span>SKU 列表</span>
                <el-button size="small" type="primary" link @click="openCreateSku(row)">新增 SKU</el-button>
              </div>
              <el-table :data="row.skus || []" border size="small">
                <el-table-column prop="skuCode" label="SKU编码" width="140" />
                <el-table-column prop="pickName" label="拣货名" min-width="120" />
                <el-table-column prop="style1" label="规格1" width="90" />
                <el-table-column prop="style2" label="规格2" width="90" />
                <el-table-column prop="style3" label="规格3" width="90" />
                <el-table-column prop="weightG" label="重量(g)" width="90" />
                <el-table-column prop="retailPrice" label="零售价" width="90" />
                <el-table-column prop="upc" label="UPC" width="110" />
                <el-table-column prop="asin" label="ASIN" width="110" />
                <el-table-column prop="productType" label="类型" width="90" />
                <el-table-column prop="status" label="状态" width="80" />
                <el-table-column label="操作" width="140" fixed="right">
                  <template #default="{ row: sku }">
                    <el-button link type="primary" @click="openEditSku(sku)">编辑</el-button>
                    <el-button link type="danger" @click="removeSku(sku)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="parentSku" label="父SKU" width="140" />
        <el-table-column prop="name" label="商品名称" min-width="180" />
        <el-table-column prop="categoryId" label="分类ID" width="90" />
        <el-table-column prop="scoreFactor" label="评分系数" width="100" />
        <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip />
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditProduct(row)">编辑</el-button>
            <el-button link type="danger" @click="removeProduct(row)">删除</el-button>
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

    <el-dialog v-model="productVisible" :title="productForm.id ? '编辑商品' : '新建商品'" width="560px">
      <el-form :model="productForm" label-width="100px">
        <el-form-item label="父SKU" required>
          <el-input v-model="productForm.parentSku" />
        </el-form-item>
        <el-form-item label="商品名称" required>
          <el-input v-model="productForm.name" />
        </el-form-item>
        <el-form-item label="分类">
          <el-select v-model="productForm.categoryId" clearable filterable style="width: 100%">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="评分系数">
          <el-input-number v-model="productForm.scoreFactor" :min="0" :step="0.1" style="width: 100%" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="productForm.remark" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="productVisible = false">取消</el-button>
        <el-button type="primary" @click="saveProduct">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="skuVisible" :title="skuForm.id ? '编辑 SKU' : '新建 SKU'" width="640px">
      <el-form :model="skuForm" label-width="110px">
        <el-form-item label="SKU编码" required>
          <el-input v-model="skuForm.skuCode" />
        </el-form-item>
        <el-form-item label="拣货名">
          <el-input v-model="skuForm.pickName" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="规格1"><el-input v-model="skuForm.style1" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="规格2"><el-input v-model="skuForm.style2" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="规格3"><el-input v-model="skuForm.style3" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="重量(g)">
          <el-input-number v-model="skuForm.weightG" :min="0" style="width: 100%" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="最近进价"><el-input-number v-model="skuForm.lastPurchasePrice" :min="0" :precision="2" style="width: 100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="最低进价"><el-input-number v-model="skuForm.minPurchasePrice" :min="0" :precision="2" style="width: 100%" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="零售价"><el-input-number v-model="skuForm.retailPrice" :min="0" :precision="2" style="width: 100%" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="UPC"><el-input v-model="skuForm.upc" /></el-form-item>
        <el-form-item label="ASIN"><el-input v-model="skuForm.asin" /></el-form-item>
        <el-form-item label="产品类型">
          <el-select v-model="skuForm.productType" style="width: 100%">
            <el-option label="普通" value="normal" />
            <el-option label="组合" value="combo" />
            <el-option label="组装" value="assembly" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="skuForm.status" style="width: 100%">
            <el-option label="在售" value="active" />
            <el-option label="停用" value="inactive" />
            <el-option label="清仓" value="clearance" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="skuVisible = false">取消</el-button>
        <el-button type="primary" @click="saveSku">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; }
.toolbar { display: flex; gap: 8px; margin-bottom: 12px; }
.pager { margin-top: 16px; justify-content: flex-end; }
.sku-wrap { padding: 8px 24px 16px; }
.sku-hdr { display: flex; justify-content: space-between; margin-bottom: 8px; font-weight: 500; }
</style>
