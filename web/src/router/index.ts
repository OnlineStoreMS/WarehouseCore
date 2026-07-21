import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import { getToken, redirectToPortal, ensureSession, clearToken } from '../utils/auth'

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
      path: '/m/photo-upload',
      name: 'MobilePhotoUpload',
      component: () => import('../views/MobilePhotoUpload.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '工作台' } },

        { path: 'products', name: 'Products', component: () => import('../views/product/ProductList.vue'), meta: { title: '商品信息', section: '商品', group: '商品管理' } },
        { path: 'categories', name: 'Categories', component: () => import('../views/product/CategoryList.vue'), meta: { title: '商品类别', section: '商品', group: '商品管理' } },
        { path: 'pack-specs', name: 'PackSpecs', component: () => import('../views/product/PackSpecList.vue'), meta: { title: '包装规格', section: '商品', group: '商品管理' } },

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

        { path: 'barcode', name: 'Barcode', component: () => import('../views/product/BarcodePrint.vue'), meta: { title: '条码打印', section: '商品' } },
        { path: 'barcode/skus', redirect: '/barcode' },
        { path: 'barcode-print', redirect: '/barcode' },

        { path: 'tools/sku-cost', name: 'SkuCost', component: () => import('../views/product/SkuCostSettings.vue'), meta: { title: '商品费用设置', section: '商品', group: '其它' } },
        { path: 'tools/weight-check', name: 'WeightCheck', component: () => import('../views/product/WeightCheck.vue'), meta: { title: '商品重量检测', section: '商品', group: '其它' } },
        { path: 'tools/profit-calc', name: 'ProfitCalc', component: () => import('../views/product/ProfitCalc.vue'), meta: { title: '商品利润试算', section: '商品', group: '其它' } },

        { path: 'boms', redirect: '/details/combo' },

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
        { path: 'other-inbounds', name: 'OtherInbounds', component: () => import('../views/io/OtherInboundList.vue'), meta: { title: '其它入库单' } },
        { path: 'other-outbounds', name: 'OtherOutbounds', component: () => import('../views/io/OtherOutboundList.vue'), meta: { title: '其它出库单' } },
        { path: 'pim-mappings', name: 'PimMappings', component: () => import('../views/integ/PimMappingList.vue'), meta: { title: '商品库映射' } },
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
