<!--
 Copyright 2026 The CubeFS Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0
-->

<template>
  <div class="trend-chart">
    <div ref="chart" class="canvas"></div>
  </div>
</template>

<script>
import Echarts from 'echarts'

// Y-axis unit per metric.
const METRIC_UNIT = {
  throughputMBs: 'MB/s',
  opsPerSec: 'ops/s',
  p50: 'μs',
  p95: 'μs',
  p99: 'μs',
  p999: 'μs',
  mean: 'μs',
}

const METRIC_LABEL = {
  throughputMBs: '吞吐',
  opsPerSec: 'IOPS',
  p50: 'P50',
  p95: 'P95',
  p99: 'P99',
  p999: 'P999',
  mean: '平均',
}

// Metadata benchmarks (mdtest) only report opsPerSec; the y-axis label should
// reflect that and the legacy "IOPS" label is misleading there.
const METRIC_LABEL_MDTEST = Object.freeze({
  ...METRIC_LABEL,
  opsPerSec: '操作速率',
})
const METRIC_UNIT_MDTEST = Object.freeze({
  ...METRIC_UNIT,
  opsPerSec: 'ops/sec',
})

export default {
  name: 'BenchTrendChart',
  props: {
    // [{ taskID, createdAt, stages: [{name, throughputMBs, opsPerSec, latency:{p50,...}}] }] — already aggregated per task by parent
    points: { type: Array, default: () => [] },
    metric: { type: String, default: 'throughputMBs' },
    // 'mdtest' | 's3' | 'posix' | '' — when 'mdtest' the y-axis label and
    // tooltip units switch to ops/sec semantics.
    storageType: { type: String, default: '' },
  },
  data() {
    return { inst: null, resizeHandler: null }
  },
  watch: {
    points: { handler() { this.$nextTick(this.render) }, deep: true },
    metric() { this.$nextTick(this.render) },
    storageType() { this.$nextTick(this.render) },
  },
  mounted() {
    this.$nextTick(this.render)
    this.resizeHandler = () => this.inst && this.inst.resize()
    window.addEventListener('resize', this.resizeHandler)
  },
  beforeDestroy() {
    if (this.resizeHandler) window.removeEventListener('resize', this.resizeHandler)
    if (this.inst) { this.inst.dispose(); this.inst = null }
  },
  methods: {
    valueAt(stage, metric) {
      if (!stage) return null
      if (metric === 'throughputMBs' || metric === 'opsPerSec') {
        return stage[metric] == null ? null : Number(stage[metric])
      }
      if (stage.latency && stage.latency[metric] != null) {
        return Number(stage.latency[metric])
      }
      return null
    },
    buildSeries() {
      // Collect all unique stage names across all task points.
      const stageNames = []
      const seen = new Set()
      for (const p of this.points) {
        for (const st of (p.stages || [])) {
          const n = st.name || '-'
          if (!seen.has(n)) { seen.add(n); stageNames.push(n) }
        }
      }
      // One series per stage; x-axis is task timestamps shared across series.
      const sorted = [...this.points].sort((a, b) => (a.createdAt || 0) - (b.createdAt || 0))
      const xs = sorted.map(p => formatTs(p.createdAt))
      const series = stageNames.map(name => ({
        name,
        type: 'line',
        smooth: true,
        showSymbol: true,
        symbolSize: 6,
        connectNulls: true,
        emphasis: { focus: 'series' },
        data: sorted.map(p => {
          const st = (p.stages || []).find(x => (x.name || '-') === name)
          const v = this.valueAt(st, this.metric)
          return v == null ? null : round(v, this.metric === 'opsPerSec' ? 0 : 2)
        }),
      }))
      return { xs, series, stageNames, sorted }
    },
    render() {
      if (!this.$refs.chart) return
      if (!this.inst) this.inst = Echarts.init(this.$refs.chart)
      const { xs, series, stageNames, sorted } = this.buildSeries()
      const isMdtest = this.storageType === 'mdtest'
      const unitMap = isMdtest ? METRIC_UNIT_MDTEST : METRIC_UNIT
      const labelMap = isMdtest ? METRIC_LABEL_MDTEST : METRIC_LABEL
      const unit = unitMap[this.metric] || ''
      const label = labelMap[this.metric] || this.metric
      this.inst.setOption({
        tooltip: {
          trigger: 'axis',
          formatter: (params) => {
            const idx = params[0] && params[0].dataIndex
            const task = sorted[idx]
            const head = `<b>${xs[idx]}</b><br/>task: ${task ? truncate(task.taskID, 36) : '-'}`
            const lines = [head]
            for (const p of params) {
              lines.push(`${p.marker} ${p.seriesName}: <b>${p.value == null ? '-' : p.value}</b> ${unit}`)
            }
            return lines.join('<br/>')
          },
        },
        legend: {
          data: stageNames,
          type: 'scroll',
          top: 0,
          textStyle: { fontSize: 11 },
        },
        grid: { left: 50, right: 30, top: 40, bottom: 70, containLabel: true },
        xAxis: {
          type: 'category',
          data: xs,
          axisLabel: { rotate: xs.length > 6 ? 35 : 0, fontSize: 11 },
        },
        yAxis: {
          type: 'value',
          name: `${label} (${unit})`,
          splitLine: { lineStyle: { type: 'dashed' } },
        },
        dataZoom: xs.length > 10
          ? [{ type: 'inside', start: 0, end: 100 }, { type: 'slider', height: 18, bottom: 32 }]
          : [],
        series,
        graphic: stageNames.length === 0 ? [{
          type: 'text', left: 'center', top: 'middle',
          style: { text: '无历史数据', fontSize: 13, fill: '#909399' },
        }] : [],
      }, true)
    },
  },
}

function round(v, digits) {
  const f = Math.pow(10, digits)
  return Math.round(Number(v) * f) / f
}

function formatTs(ts) {
  if (!ts) return '-'
  const d = new Date(typeof ts === 'number' ? ts : Date.parse(ts))
  if (isNaN(d.getTime())) return '-'
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function truncate(s, n) {
  if (!s) return '-'
  return s.length <= n ? s : s.slice(0, n) + '...'
}
</script>

<style lang="scss" scoped>
.trend-chart {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 10px 12px;
}
.canvas {
  width: 100%;
  height: 360px;
}
</style>
