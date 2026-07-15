<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  HomeFilled, Goods, Box, OfficeBuilding, DataBoard,
  Document, Switch, Download, Upload, Printer, Link,
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const collapsed = defineModel<boolean>('collapsed', { default: false })

const activeMenu = computed(() => route.path)
const logoText = computed(() => (collapsed.value ? 'WC' : '仓储中心'))

function navigate(path: string) {
  router.push(path)
}

watch(() => route.path, () => {})
</script>

<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="logo">{{ logoText }}</div>
    <el-menu
      :default-active="activeMenu"
      :collapse="collapsed"
      :default-openeds="['product', 'stock', 'wh', 'stk', 'xfer', 'io', 'integ']"
      background-color="#001529"
      text-color="#ffffffa6"
      active-text-color="#fff"
    >
      <el-menu-item index="/dashboard" @click="navigate('/dashboard')">
        <el-icon><HomeFilled /></el-icon><span>工作台</span>
      </el-menu-item>

      <el-sub-menu index="product">
        <template #title><el-icon><Goods /></el-icon><span>商品管理</span></template>
        <el-menu-item index="/products" @click="navigate('/products')">商品信息</el-menu-item>
        <el-menu-item index="/categories" @click="navigate('/categories')">商品类别</el-menu-item>
        <el-menu-item index="/pack-specs" @click="navigate('/pack-specs')">包装规格</el-menu-item>
        <el-menu-item index="/boms" @click="navigate('/boms')">组合/组装品</el-menu-item>
        <el-menu-item index="/barcode" @click="navigate('/barcode')">
          <el-icon><Printer /></el-icon><span>条码打印</span>
        </el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="stock">
        <template #title><el-icon><DataBoard /></el-icon><span>库存情况</span></template>
        <el-menu-item index="/stock/balances" @click="navigate('/stock/balances')">库存查询</el-menu-item>
        <el-menu-item index="/stock/summary" @click="navigate('/stock/summary')">库存汇总账</el-menu-item>
        <el-menu-item index="/stock/movements" @click="navigate('/stock/movements')">库存明细表</el-menu-item>
        <el-menu-item index="/stock/slow-moving" @click="navigate('/stock/slow-moving')">滞销查询</el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="wh">
        <template #title><el-icon><OfficeBuilding /></el-icon><span>仓库货位</span></template>
        <el-menu-item index="/warehouses" @click="navigate('/warehouses')">仓库设置</el-menu-item>
        <el-menu-item index="/locations" @click="navigate('/locations')">库位管理</el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="stk">
        <template #title><el-icon><Document /></el-icon><span>仓库盘点</span></template>
        <el-menu-item index="/stocktakes" @click="navigate('/stocktakes')">仓库盘点单</el-menu-item>
        <el-menu-item index="/stocktake-details" @click="navigate('/stocktake-details')">盘点明细表</el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="xfer">
        <template #title><el-icon><Switch /></el-icon><span>仓库调拨</span></template>
        <el-menu-item index="/transfers" @click="navigate('/transfers')">仓库调拨单</el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="io">
        <template #title><el-icon><Box /></el-icon><span>其他出入库</span></template>
        <el-menu-item index="/other-inbounds" @click="navigate('/other-inbounds')">
          <el-icon><Download /></el-icon><span>其他入库单</span>
        </el-menu-item>
        <el-menu-item index="/other-outbounds" @click="navigate('/other-outbounds')">
          <el-icon><Upload /></el-icon><span>其他出库单</span>
        </el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="integ">
        <template #title><el-icon><Link /></el-icon><span>对接预留</span></template>
        <el-menu-item index="/pim-mappings" @click="navigate('/pim-mappings')">PIM 映射</el-menu-item>
      </el-sub-menu>
    </el-menu>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  background: #001529;
  transition: width 0.2s;
  flex-shrink: 0;
  overflow-y: auto;
}
.sidebar.collapsed { width: 64px; }
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 600;
  font-size: 16px;
  border-bottom: 1px solid #ffffff14;
}
.sidebar :deep(.el-menu) { border-right: none; }
</style>
