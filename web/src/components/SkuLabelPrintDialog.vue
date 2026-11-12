<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Printer, RefreshLeft } from '@element-plus/icons-vue'
import {
  LABEL_SIZE_PRESETS,
  loadSavedLabelPrefs,
  printSkuLabels,
  resolvePickName,
  resolvePrintSize,
  saveLabelPrefs,
  skuBarcodeSvg,
  type LabelOrientation,
  type LabelSize,
  type SkuLabelItem,
} from '../utils/skuLabelPrint'

type EditableRow = SkuLabelItem & { _key: string; _origPickName: string }

const props = defineProps<{
  modelValue: boolean
  items: SkuLabelItem[]
}>()

const emit = defineEmits<{
  'update:modelValue': [boolean]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const size = ref<LabelSize>(loadSavedLabelPrefs().size)
const orientation = ref<LabelOrientation>(loadSavedLabelPrefs().orientation)
const presetId = ref('')
const copies = ref(1)
const customW = ref(40)
const customH = ref(50)
const rows = ref<EditableRow[]>([])

const printSize = computed(() => resolvePrintSize(size.value, orientation.value))

const printItems = computed(() =>
  rows.value.map((it) => ({
    skuCode: String(it.skuCode || '').trim(),
    pickName: String(it.pickName || '').trim(),
    copies: Math.min(Math.max(Number(copies.value) || 1, 1), 99),
  })),
)

const totalSheets = computed(() =>
  printItems.value.reduce((n, it) => n + (Math.min(Math.max(Number(it.copies) || 1, 1), 99)), 0),
)

const editedCount = computed(() =>
  rows.value.filter((r) => String(r.pickName || '').trim() !== String(r._origPickName || '').trim()).length,
)

function cloneRows(list: SkuLabelItem[]): EditableRow[] {
  return list.map((it, i) => {
    const pickName = String(it.pickName || '').trim()
    return {
      ...it,
      pickName,
      _origPickName: pickName,
      _key: `${it.skuCode || ''}-${it.id || ''}-${i}`,
    }
  })
}

function matchPreset(s: LabelSize) {
  const hit = LABEL_SIZE_PRESETS.find(
    (p) => p.widthMm === s.widthMm && p.heightMm === s.heightMm,
  )
  presetId.value = hit?.id || 'custom'
  customW.value = s.widthMm
  customH.value = s.heightMm
}

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    const saved = loadSavedLabelPrefs()
    size.value = { ...saved.size }
    orientation.value = saved.orientation
    matchPreset(saved.size)
    rows.value = cloneRows(props.items)
    const firstCopies = props.items[0]?.copies
    copies.value = firstCopies && firstCopies > 0 ? Math.min(firstCopies, 99) : 1
  },
)

function onPresetChange(id: string) {
  if (id === 'custom') {
    size.value = {
      widthMm: Math.max(10, Number(customW.value) || 40),
      heightMm: Math.max(10, Number(customH.value) || 30),
    }
    return
  }
  const p = LABEL_SIZE_PRESETS.find((x) => x.id === id)
  if (!p) return
  size.value = { widthMm: p.widthMm, heightMm: p.heightMm }
  customW.value = p.widthMm
  customH.value = p.heightMm
}

function onCustomChange() {
  presetId.value = 'custom'
  size.value = {
    widthMm: Math.max(10, Math.min(200, Number(customW.value) || 40)),
    heightMm: Math.max(10, Math.min(300, Number(customH.value) || 30)),
  }
}

function resetSpecs() {
  rows.value = rows.value.map((r) => ({ ...r, pickName: r._origPickName }))
  ElMessage.success('已恢复原始配货名称')
}

const previewScale = computed(() => {
  const maxEdge = Math.max(printSize.value.widthMm, printSize.value.heightMm)
  if (maxEdge >= 100) return 2.2
  if (maxEdge >= 70) return 2.8
  return 3.4
})

function doPrint() {
  if (!printItems.value.length) {
    ElMessage.warning('没有可打印的 SKU')
    return
  }
  const emptyPick = printItems.value.filter((it) => !String(it.pickName || '').trim())
  if (emptyPick.length) {
    ElMessage.warning('请填写配货名称后再打印')
    return
  }
  const missing = printItems.value.filter((it) => !String(it.skuCode || '').trim())
  if (missing.length) {
    ElMessage.warning('存在缺少 SKU 编码的记录，请检查后再打')
    return
  }
  saveLabelPrefs({ size: size.value, orientation: orientation.value })
  const ok = printSkuLabels(printItems.value, size.value, orientation.value)
  if (!ok) {
    ElMessage.error('打印失败，请刷新页面后重试')
    return
  }
  ElMessage.success(`已调起打印（${totalSheets.value} 张）`)
  visible.value = false
}
</script>

<template>
  <el-dialog
    v-model="visible"
    title="打印条码"
    width="860px"
    destroy-on-close
    append-to-body
  >
    <div class="toolbar">
      <span class="field-label">标签尺寸</span>
      <el-select v-model="presetId" style="width: 160px" @change="onPresetChange">
        <el-option
          v-for="p in LABEL_SIZE_PRESETS"
          :key="p.id"
          :label="p.label"
          :value="p.id"
        />
        <el-option label="自定义" value="custom" />
      </el-select>
      <template v-if="presetId === 'custom'">
        <el-input-number
          v-model="customW"
          :min="10"
          :max="200"
          :step="1"
          controls-position="right"
          style="width: 110px"
          @change="onCustomChange"
        />
        <span>×</span>
        <el-input-number
          v-model="customH"
          :min="10"
          :max="300"
          :step="1"
          controls-position="right"
          style="width: 110px"
          @change="onCustomChange"
        />
        <span class="unit">mm</span>
      </template>
      <span class="field-label">方向</span>
      <el-radio-group v-model="orientation" size="small">
        <el-radio-button value="portrait">纵向</el-radio-button>
        <el-radio-button value="landscape">横向</el-radio-button>
      </el-radio-group>
      <span class="field-label">每SKU份数</span>
      <el-input-number v-model="copies" :min="1" :max="99" controls-position="right" style="width: 110px" />
      <el-button
        :icon="RefreshLeft"
        :disabled="!editedCount"
        @click="resetSpecs"
      >恢复配货名称</el-button>
      <span class="hint">
        共 {{ rows.length }} 个 SKU · {{ totalSheets }} 张 ·
        {{ printSize.widthMm }}×{{ printSize.heightMm }} mm
        （{{ orientation === 'landscape' ? '横向' : '纵向' }}）
        <template v-if="editedCount"> · 已改 {{ editedCount }} 条配货名称</template>
      </span>
    </div>

    <p class="tip">
      可直接修改下方「配货名称」，打印与预览都会用修改后的文案；不影响商品原始数据。标签从上到下是条形码、SKU 编码、配货名称，配货名称过长会自动缩小字号。
    </p>

    <el-table :data="rows" border size="small" max-height="280" class="edit-table">
      <el-table-column label="SKU编码" prop="skuCode" width="180" show-overflow-tooltip />
      <el-table-column label="配货名称" min-width="260">
        <template #default="{ row }">
          <el-input
            v-model="row.pickName"
            type="textarea"
            :autosize="{ minRows: 1, maxRows: 3 }"
            maxlength="200"
            show-word-limit
            placeholder="标签上显示的配货名称"
          />
        </template>
      </el-table-column>
    </el-table>

    <div class="preview-board">
      <div
        v-for="(it, idx) in printItems.slice(0, 6)"
        :key="`${it.skuCode}-${idx}`"
        class="preview-label"
        :style="{
          width: `${printSize.widthMm * previewScale}px`,
          height: `${printSize.heightMm * previewScale}px`,
        }"
      >
        <div class="preview-inner">
          <div v-if="it.skuCode" class="preview-barcode" v-html="skuBarcodeSvg(it.skuCode)" />
          <div class="preview-code">{{ it.skuCode || '—' }}</div>
          <div class="preview-pick">{{ resolvePickName(it) }}</div>
        </div>
      </div>
      <div v-if="printItems.length > 6" class="more">… 另有 {{ printItems.length - 6 }} 个 SKU</div>
    </div>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :icon="Printer" @click="doPrint">打印</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}
.field-label {
  color: #606266;
  font-size: 13px;
  white-space: nowrap;
}
.unit { color: #909399; font-size: 13px; }
.hint { margin-left: auto; color: #909399; font-size: 12px; }
.tip {
  margin: 0 0 12px;
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}
.edit-table { margin-bottom: 12px; }
.preview-board {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: flex-start;
  min-height: 120px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 8px;
}
.preview-label {
  background: #fff;
  border: 1px dashed #c0c4cc;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.06);
  overflow: hidden;
  transition: width .15s ease, height .15s ease;
}
.preview-inner {
  position: relative;
  width: 100%;
  height: 100%;
  padding: 6% 6% 5%;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}
.preview-barcode {
  flex: 0 0 28%;
  min-height: 28px;
  width: 100%;
}
.preview-barcode :deep(svg) {
  width: 92%;
  height: 100%;
  margin: 0 auto;
  display: block;
  shape-rendering: crispEdges;
}
.preview-code {
  flex: 0 0 auto;
  margin-top: 2px;
  text-align: center;
  font-family: ui-monospace, Menlo, Consolas, monospace;
  font-size: 11px;
  font-weight: 700;
  word-break: break-all;
  line-height: 1.15;
}
.preview-pick {
  flex: 1;
  min-height: 0;
  margin-top: 8px;
  font-weight: 700;
  font-size: 13px;
  line-height: 1.25;
  word-break: break-word;
  white-space: pre-wrap;
  overflow: hidden;
}
.more {
  align-self: center;
  color: #909399;
  font-size: 13px;
}
</style>
