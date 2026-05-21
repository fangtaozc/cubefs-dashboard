<!--
 Copyright 2026 The CubeFS Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0
-->

<template>
  <div class="bench-stage-charts">
    <!-- KPI 大数字卡（单 stage 任务直接看完整指标） -->
    <div v-if="stages.length === 1" class="kpi-row">
      <div class="kpi" :class="{ dim: !isMetadataMode && !hasThroughput }">
        <div class="kpi-label">{{ isMetadataMode ? '操作速率' : '吞吐' }}</div>
        <div class="kpi-value">
          {{ isMetadataMode ? formatNum(stages[0].opsPerSec, 0) : formatNum(stages[0].throughputMBs, 1) }}
          <span class="kpi-unit">{{ isMetadataMode ? 'ops/sec' : 'MB/s' }}</span>
        </div>
      </div>
      <div v-if="!isMetadataMode" class="kpi">
        <div class="kpi-label">IOPS</div>
        <div class="kpi-value">{{ formatNum(stages[0].opsPerSec, 0) }}</div>
      </div>
      <div v-if="!isMetadataMode" class="kpi" :class="{ dim: !hasLatency }">
        <div class="kpi-label">P50 延迟</div>
        <div class="kpi-value">
          <template v-if="stages[0].latency">
            {{ formatLatencyParts(stages[0].latency.p50).value }}<span class="kpi-unit">{{ formatLatencyParts(stages[0].latency.p50).unit }}</span>
          </template>
          <template v-else>-</template>
        </div>
      </div>
      <div v-if="!isMetadataMode" class="kpi" :class="{ dim: !hasLatency }">
        <div class="kpi-label">P99 延迟</div>
        <div class="kpi-value">
          <template v-if="stages[0].latency">
            {{ formatLatencyParts(stages[0].latency.p99).value }}<span class="kpi-unit">{{ formatLatencyParts(stages[0].latency.p99).unit }}</span>
          </template>
          <template v-else>-</template>
          <span v-if="hasLatency && latencyTailRatio > 1.5" class="tail-hint">长尾 {{ latencyTailRatio.toFixed(1) }}×</span>
        </div>
      </div>
      <div v-if="isMetadataMode" class="kpi">
        <div class="kpi-label">操作总数</div>
        <div class="kpi-value">{{ formatNum(stages[0].totalOps, 0) }}</div>
      </div>
    </div>

    <!-- 多 stage 对比：metric 切换 + 水平条形图（按选定指标降序） -->
    <div v-else>
      <div class="metric-switch">
        <el-radio-group v-model="primaryMetric" size="mini">
          <el-radio-button v-if="!isMetadataMode" label="throughputMBs">吞吐 MB/s</el-radio-button>
          <el-radio-button label="opsPerSec">{{ isMetadataMode ? '操作速率 ops/sec' : 'IOPS' }}</el-radio-button>
          <el-radio-button v-if="!isMetadataMode" label="p50">P50 延迟</el-radio-button>
          <el-radio-button v-if="!isMetadataMode" label="p99">P99 延迟</el-radio-button>
        </el-radio-group>
        <span class="metric-hint">{{ isLatencyMetric ? '越低越好' : '越高越好' }} · 按数值降序排列</span>
      </div>
      <div ref="barChart" class="bar-chart" :style="{ height: barChartHeight + 'px' }"></div>

      <!-- 延迟分布详图：每个 stage 一条 P50→P95→P99→P999 折线（对数 Y 轴，长尾差距清晰） -->
      <div v-if="!isMetadataMode && hasLatency" class="latency-section">
        <div class="section-label">延迟分布对比（对数 Y 轴 · 折线越陡 = 长尾越严重）</div>
        <div ref="latencyChart" class="latency-chart"></div>
      </div>
    </div>
  </div>
</template>

<script>
import Echarts from 'echarts'

const METRIC_DEF = {
  throughputMBs: { label: '吞吐', unit: 'MB/s', color: '#409eff', digits: 1 },
  opsPerSec:     { label: 'IOPS', unit: 'ops/s', color: '#67c23a', digits: 0 },
  p50:           { label: 'P50',  unit: 'μs',    color: '#91cc75', digits: 0, latency: true },
  p95:           { label: 'P95',  unit: 'μs',    color: '#fac858', digits: 0, latency: true },
  p99:           { label: 'P99',  unit: 'μs',    color: '#ee6666', digits: 0, latency: true },
  p999:          { label: 'P999', unit: 'μs',    color: '#9a60b4', digits: 0, latency: true },
}

export default {
  name: 'BenchStageCharts',
  props: {
    stages: { type: Array, default: () => [] },
    storageType: { type: String, default: '' },
  },
  data() {
    return {
      barInst: null,
      latencyInst: null,
      resizeHandler: null,
      primaryMetric: 'throughputMBs',
    }
  },
  computed: {
    isMetadataMode() {
      if (this.storageType === 'mdtest') return true
      if (!Array.isArray(this.stages) || this.stages.length === 0) return false
      const allMbZero = this.stages.every(s => !(Number(s && s.throughputMBs) > 0))
      const allLatZero = this.stages.every(s => {
        const l = s && s.latency
        if (!l) return true
        return !(Number(l.p50) > 0 || Number(l.p95) > 0 || Number(l.p99) > 0 || Number(l.p999) > 0 || Number(l.mean) > 0)
      })
      const anyOps = this.stages.some(s => Number(s && s.opsPerSec) > 0)
      return allMbZero && allLatZero && anyOps
    },
    hasThroughput() {
      return this.stages.some(s => Number(s && s.throughputMBs) > 0)
    },
    hasLatency() {
      return this.stages.some(s => {
        const l = s && s.latency
        return l && (Number(l.p50) > 0 || Number(l.p95) > 0 || Number(l.p99) > 0)
      })
    },
    latencyTailRatio() {
      const s = this.stages[0]
      if (!s || !s.latency) return 0
      const p50 = Number(s.latency.p50) || 0
      const p99 = Number(s.latency.p99) || 0
      return p50 > 0 ? p99 / p50 : 0
    },
    isLatencyMetric() {
      return !!(METRIC_DEF[this.primaryMetric] && METRIC_DEF[this.primaryMetric].latency)
    },
    barChartHeight() {
      // 每个 stage 38px + 60px header；至少 200px
      return Math.max(200, this.stages.length * 38 + 60)
    },
  },
  watch: {
    stages: { handler() { this.snapMetric(); this.$nextTick(this.render) }, deep: true },
    storageType() { this.snapMetric(); this.$nextTick(this.render) },
    primaryMetric() { this.$nextTick(this.render) },
  },
  mounted() {
    this.snapMetric()
    this.$nextTick(this.render)
    this.resizeHandler = () => {
      if (this.barInst) this.barInst.resize()
      if (this.latencyInst) this.latencyInst.resize()
    }
    window.addEventListener('resize', this.resizeHandler)
  },
  beforeDestroy() {
    if (this.resizeHandler) window.removeEventListener('resize', this.resizeHandler)
    if (this.barInst) { this.barInst.dispose(); this.barInst = null }
    if (this.latencyInst) { this.latencyInst.dispose(); this.latencyInst = null }
  },
  methods: {
    // mdtest 模式下主指标只有 opsPerSec；切到不兼容指标时强制回退。
    snapMetric() {
      if (this.isMetadataMode && this.primaryMetric !== 'opsPerSec') {
        this.primaryMetric = 'opsPerSec'
      } else if (!this.isMetadataMode && this.primaryMetric === 'opsPerSec' && this.hasThroughput) {
        // 第一次 mount 默认到吞吐（性能视角通常先看吞吐）
        this.primaryMetric = 'throughputMBs'
      }
    },
    formatNum(v, digits) {
      const n = Number(v) || 0
      if (n >= 1e6) return (n / 1e6).toFixed(2) + 'M'
      if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K'
      return n.toFixed(digits)
    },
    // Pick a latency unit (s/ms/μs) for a single value — used by the KPI
    // cards where each metric stands alone.
    formatLatencyParts(us) {
      const u = pickLatencyUnit(Number(us) || 0)
      return formatLatencyWithUnit(us, u)
    },
    metricValue(stage, metric) {
      if (!stage) return 0
      if (metric === 'throughputMBs' || metric === 'opsPerSec') return Number(stage[metric]) || 0
      return stage.latency ? Number(stage.latency[metric]) || 0 : 0
    },
    render() {
      if (this.stages.length === 1) return // KPI 卡，不画图
      this.renderBar()
      if (!this.isMetadataMode && this.hasLatency) this.$nextTick(() => this.renderLatency())
    },
    renderBar() {
      const el = this.$refs.barChart
      if (!el) return
      if (!this.barInst) this.barInst = Echarts.init(el)
      const def = METRIC_DEF[this.primaryMetric]
      // 按选定指标降序（延迟指标反向，越低越好）
      const sorted = [...this.stages].sort((a, b) => {
        const va = this.metricValue(a, this.primaryMetric)
        const vb = this.metricValue(b, this.primaryMetric)
        return this.isLatencyMetric ? va - vb : vb - va
      })
      const names = sorted.map(s => s.name || '-')
      const rawValues = sorted.map(s => this.metricValue(s, this.primaryMetric))

      // For latency metrics pick a single unit (μs/ms/s) based on the
      // largest value so every bar in the chart is read in the same
      // currency. Throughput / IOPS keep their static unit.
      let unit = def.unit
      let displayValues = rawValues.map(v => Number(v.toFixed(def.digits)))
      if (def.latency) {
        const u = pickLatencyUnit(Math.max(...rawValues, 1))
        unit = u.unit
        displayValues = rawValues.map(v => Number((v / u.divisor).toFixed(u.digits)))
      }

      this.barInst.setOption({
        tooltip: {
          trigger: 'axis',
          axisPointer: { type: 'shadow' },
          formatter: (params) => {
            const p = params[0]
            const s = sorted[p.dataIndex]
            const lines = [`<b>${s.name}</b>`, `${def.label}: <b>${p.value} ${unit}</b>`]
            if (!this.isMetadataMode && !this.isLatencyMetric) {
              if (s.throughputMBs) lines.push(`吞吐: ${Number(s.throughputMBs).toFixed(1)} MB/s`)
              if (s.opsPerSec) lines.push(`IOPS: ${Math.round(s.opsPerSec)}`)
              if (s.latency && s.latency.p99) {
                const p99 = formatLatencyParts(s.latency.p99)
                lines.push(`P99: ${p99.value} ${p99.unit}`)
              }
            }
            if (s.durationSec) lines.push(`耗时: ${Number(s.durationSec).toFixed(1)} s`)
            return lines.join('<br/>')
          },
        },
        grid: { left: 12, right: 90, top: 16, bottom: 18, containLabel: true },
        xAxis: {
          type: 'value',
          name: unit,
          nameLocation: 'end',
          nameTextStyle: { color: '#909399', fontSize: 11 },
          splitLine: { lineStyle: { type: 'dashed', color: '#e4e7ed' } },
          axisLabel: { fontSize: 10, color: '#909399' },
        },
        yAxis: {
          type: 'category',
          data: names,
          inverse: true, // 最高的排最上
          axisLabel: { fontSize: 11, color: '#303133', formatter: (v) => v.length > 28 ? '…' + v.slice(v.length - 27) : v },
          axisTick: { show: false },
          axisLine: { lineStyle: { color: '#dcdfe6' } },
        },
        series: [{
          type: 'bar',
          data: displayValues.map((v, i) => ({
            value: v,
            itemStyle: { color: i === 0 ? def.color : shadeColor(def.color, 30 + i * 8) },
          })),
          barMaxWidth: 22,
          label: {
            show: true,
            position: 'right',
            formatter: (p) => `${formatLabelNum(p.value)} ${unit}`,
            color: '#303133',
            fontSize: 11,
          },
          // 进度条样式：浅色背景 + 实色填充
          showBackground: true,
          backgroundStyle: { color: '#f5f7fa', borderRadius: 3 },
          itemStyle: { borderRadius: 3 },
        }],
      }, true)
    },
    renderLatency() {
      const el = this.$refs.latencyChart
      if (!el) return
      if (!this.latencyInst) this.latencyInst = Echarts.init(el)
      const fullNames = this.stages.map(s => s.name || '-')
      const series = this.stages.map((s, i) => ({
        name: fullNames[i],
        type: 'line',
        smooth: false,
        symbolSize: 8,
        data: [
          ['P50', Number((s.latency && s.latency.p50) || 0)],
          ['P95', Number((s.latency && s.latency.p95) || 0)],
          ['P99', Number((s.latency && s.latency.p99) || 0)],
          ['P999', Number((s.latency && s.latency.p999) || 0)],
        ],
        emphasis: { focus: 'series' },
      }))
      this.latencyInst.setOption({
        tooltip: {
          trigger: 'axis',
          valueFormatter: (v) => {
            const p = formatLatencyParts(v)
            return `${p.value} ${p.unit}`
          },
        },
        legend: { data: fullNames, top: 0, type: 'scroll', textStyle: { fontSize: 11 } },
        grid: { left: 50, right: 24, top: 36, bottom: 30, containLabel: true },
        xAxis: { type: 'category', data: ['P50', 'P95', 'P99', 'P999'], axisLabel: { fontSize: 11 } },
        yAxis: {
          type: 'log',
          logBase: 10,
          name: '延迟 (log)',
          nameTextStyle: { color: '#909399', fontSize: 11 },
          splitLine: { lineStyle: { type: 'dashed', color: '#e4e7ed' } },
          axisLabel: {
            fontSize: 10,
            // Each log-scale tick formats independently, so a chart that
            // spans 10μs → 5s shows "10μs / 100μs / 1ms / 10ms / 100ms / 1s".
            formatter: (v) => {
              const p = formatLatencyParts(v)
              return `${p.value}${p.unit}`
            },
          },
        },
        series,
      }, true)
    },
  },
}

function formatLabelNum(n) {
  n = Number(n) || 0
  if (n >= 1e6) return (n / 1e6).toFixed(2) + 'M'
  if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K'
  return n >= 100 ? n.toFixed(0) : n.toFixed(1)
}

// pickLatencyUnit chooses between μs / ms / s based on the magnitude of
// the largest value in scope. Latency input from fio is always in μs.
//
//   < 1000 μs       → keep μs (e.g. 182 μs)
//   < 1_000_000 μs  → ms     (e.g. 12.5 ms)
//   ≥ 1_000_000 μs  → s      (e.g. 1.02 s)
//
// Digits scale with the displayed value so the number stays readable
// (3 sig figs at most): 182 μs / 12.5 ms / 1.02 s — never "1022.000ms".
function pickLatencyUnit(maxUs) {
  const n = Math.abs(Number(maxUs) || 0)
  if (n >= 1e6) return { unit: 's', divisor: 1e6 }
  if (n >= 1e3) return { unit: 'ms', divisor: 1e3 }
  return { unit: 'μs', divisor: 1 }
}

// formatLatencyWithUnit converts μs to {value, unit} given a unit picked
// upstream. Used both standalone (one value per call) and inside
// renderBar where every bar must share one unit.
function formatLatencyWithUnit(us, u) {
  const v = (Number(us) || 0) / u.divisor
  let digits
  if (v === 0) digits = 0
  else if (v >= 100) digits = 0
  else if (v >= 10) digits = 1
  else digits = 2
  return { value: v.toFixed(digits), unit: u.unit }
}

// formatLatencyParts is the ECharts-friendly "single value" wrapper:
// callers that only need one number (tooltip, KPI card) don't have to
// pre-pick a unit.
function formatLatencyParts(us) {
  return formatLatencyWithUnit(us, pickLatencyUnit(us))
}

// 简单根据 base 颜色生成略浅的同色系（用于条形图除冠军外的次序色）
function shadeColor(hex, percent) {
  const h = hex.replace('#', '')
  const num = parseInt(h, 16)
  const r = Math.min(255, ((num >> 16) & 0xff) + percent)
  const g = Math.min(255, ((num >> 8) & 0xff) + percent)
  const b = Math.min(255, (num & 0xff) + percent)
  return '#' + ((1 << 24) | (r << 16) | (g << 8) | b).toString(16).slice(1)
}
</script>

<style lang="scss" scoped>
.bench-stage-charts {
  margin-bottom: 16px;
}

/* KPI 大数字卡 */
.kpi-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 10px;
  margin-bottom: 8px;
}
.kpi {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 12px 14px;
  transition: opacity 0.2s;
  &.dim { opacity: 0.5; }
}
.kpi-label {
  font-size: 12px;
  color: #909399;
  margin-bottom: 6px;
  letter-spacing: 0.5px;
}
.kpi-value {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
  line-height: 1.2;
  display: flex;
  align-items: baseline;
  gap: 6px;
  flex-wrap: wrap;
}
.kpi-unit {
  font-size: 12px;
  font-weight: normal;
  color: #909399;
}
.tail-hint {
  font-size: 11px;
  color: #e6a23c;
  background: #fdf6ec;
  padding: 1px 6px;
  border-radius: 3px;
  margin-left: auto;
}

/* 多 stage 对比 */
.metric-switch {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 6px;
}
.metric-hint {
  font-size: 12px;
  color: #909399;
}
.bar-chart {
  width: 100%;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 8px 10px 4px;
}

.latency-section {
  margin-top: 14px;
}
.section-label {
  font-size: 12px;
  color: #606266;
  margin-bottom: 4px;
}
.latency-chart {
  width: 100%;
  height: 220px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 8px;
}
</style>
