<script setup lang="ts">
import { Box, Goods, OfficeBuilding, Switch } from '@element-plus/icons-vue'
</script>

<template>
  <div class="dashboard">
    <el-row :gutter="16">
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <el-icon :size="32" color="#409eff"><Goods /></el-icon>
            <div>
              <div class="stat-title">商品管理</div>
              <div class="stat-desc">商品信息、类别、包装规格与组合/组装</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <el-icon :size="32" color="#67c23a"><OfficeBuilding /></el-icon>
            <div>
              <div class="stat-title">仓库货位</div>
              <div class="stat-desc">仓库档案与库位编码，支持默认仓与分区货架</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <el-icon :size="32" color="#e6a23c"><Box /></el-icon>
            <div>
              <div class="stat-title">库存账</div>
              <div class="stat-desc">结存查询、汇总账、出入流水与滞销分析</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <div class="stat-card">
            <el-icon :size="32" color="#f56c6c"><Switch /></el-icon>
            <div>
              <div class="stat-title">盘点调拨</div>
              <div class="stat-desc">仓库盘点、仓间调拨与其他出入库过账</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.stat-card {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}
.stat-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 8px;
}
.stat-desc {
  font-size: 13px;
  color: #909399;
  line-height: 1.5;
}
</style>
