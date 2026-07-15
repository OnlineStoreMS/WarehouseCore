import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import { getToken, redirectToPortal, ensureSession, clearToken } from '../utils/auth'

const Placeholder = () => import('../views/common/PlaceholderPage.vue')

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/auth/callback',
      name: 'AuthCallback',
      component: () => import('../views/AuthCallback.vue'),
      meta: { public: true },
    },
    {
      path: '/auth/logout',
      name: 'AuthLogout',
      component: () => import('../views/AuthLogout.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '工作台' } },

        // 商品管理
        { path: 'products', name: 'Products', component: () => import('../views/product/ProductList.vue'), meta: { title: '商品信息', section: '商品', group: '商品管理' } },
        { path: 'categories', name: 'Categories', component: () => import('../views/product/CategoryList.vue'), meta: { title: '商品类别', section: '商品', group: '商品管理' } },
        { path: 'pack-specs', name: 'PackSpecs', component: () => import('../views/product/PackSpecList.vue'), meta: { title: '包装规格', section: '商品', group: '商品管理' } },

        // 商品明细
        {
          path: 'details/assembly',
          name: 'AssemblyDetails',
          component: () => import('../views/product/BomTypeDetailList.vue'),
          props: { bomType: 'assembly' },
          meta: { title: '组装品明细', section: '商品', group: '商品明细' },
        },
        {
          path: 'details/combo',
          name: 'ComboDetails',
          component: () => import('../views/product/BomTypeDetailList.vue'),
          props: { bomType: 'combo' },
          meta: { title: '组合品明细', section: '商品', group: '商品明细' },
        },
        {
          path: 'details/store-skus',
          name: 'StoreSkuDetails',
          component: () => import('../views/product/StoreSkuDetailList.vue'),
          meta: { title: '店铺SKU明细', section: '商品', group: '商品明细' },
        },

        // 条码打印
        { path: 'barcode', redirect: '/barcode/skus' },
        { path: 'barcode/skus', name: 'BarcodeSkus', component: () => import('../views/product/BarcodePrint.vue'), meta: { title: '库存SKU', section: '商品', group: '条码打印' } },
        {
          path: 'barcode/overseas',
          name: 'BarcodeOverseas',
          component: Placeholder,
          meta: { title: '海外仓SKU', section: '商品', group: '条码打印', placeholderTip: '海外仓条码打印将按普源 overseas SKU 打印能力补齐。' },
        },

        // 图片
        {
          path: 'images/space',
          name: 'ImageSpace',
          component: Placeholder,
          meta: { title: '图片空间', section: '商品', group: '图片', placeholderTip: '图片空间用于统一管理商品主图/SKU 图，后续对接对象存储浏览与清理。' },
        },

        // 其它
        {
          path: 'tools/sku-cost',
          name: 'SkuCost',
          component: Placeholder,
          meta: { title: '商品费用设置', section: '商品', group: '其它', placeholderTip: '按库存SKU维护头程/包装/其它费用项，对齐普源商品费用设置。' },
        },
        {
          path: 'tools/weight-check',
          name: 'WeightCheck',
          component: Placeholder,
          meta: { title: '商品重量检测', section: '商品', group: '其它', placeholderTip: '比对申报重量与实测重量差异，对齐普源商品重量检测。' },
        },
        {
          path: 'tools/profit-calc',
          name: 'ProfitCalc',
          component: Placeholder,
          meta: { title: '商品利润试算', section: '商品', group: '其它', placeholderTip: '按售价、采购成本与费用项试算毛利，对齐普源商品利润试算。' },
        },

        // 兼容旧入口
        { path: 'boms', redirect: '/details/combo' },
        { path: 'barcode-print', redirect: '/barcode/skus' },

        { path: 'warehouses', name: 'Warehouses', component: () => import('../views/warehouse/WarehouseList.vue'), meta: { title: '仓库设置' } },
        { path: 'locations', name: 'Locations', component: () => import('../views/warehouse/LocationList.vue'), meta: { title: '库位管理' } },
        { path: 'stock/balances', name: 'StockBalances', component: () => import('../views/stock/BalanceList.vue'), meta: { title: '库存查询' } },
        { path: 'stock/summary', name: 'StockSummary', component: () => import('../views/stock/SummaryList.vue'), meta: { title: '库存汇总账' } },
        { path: 'stock/movements', name: 'StockMovements', component: () => import('../views/stock/MovementList.vue'), meta: { title: '库存明细表' } },
        { path: 'stock/slow-moving', name: 'SlowMoving', component: () => import('../views/stock/SlowMovingList.vue'), meta: { title: '滞销查询' } },
        { path: 'stocktakes', name: 'Stocktakes', component: () => import('../views/stocktake/StocktakeList.vue'), meta: { title: '仓库盘点单' } },
        { path: 'stocktakes/:id', name: 'StocktakeDetail', component: () => import('../views/stocktake/StocktakeDetail.vue'), meta: { title: '盘点单详情' } },
        { path: 'stocktake-details', name: 'StocktakeDetails', component: () => import('../views/stocktake/StocktakeDetailList.vue'), meta: { title: '盘点明细表' } },
        { path: 'transfers', name: 'Transfers', component: () => import('../views/transfer/TransferList.vue'), meta: { title: '仓库调拨单' } },
        { path: 'other-inbounds', name: 'OtherInbounds', component: () => import('../views/io/OtherInboundList.vue'), meta: { title: '其他入库单' } },
        { path: 'other-outbounds', name: 'OtherOutbounds', component: () => import('../views/io/OtherOutboundList.vue'), meta: { title: '其他出库单' } },
        { path: 'pim-mappings', name: 'PimMappings', component: () => import('../views/integ/PimMappingList.vue'), meta: { title: 'PIM 映射' } },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  if (!getToken()) {
    redirectToPortal()
    return false
  }
  const ok = await ensureSession()
  if (!ok) {
    clearToken()
    redirectToPortal()
    return false
  }
  return true
})

export default router
