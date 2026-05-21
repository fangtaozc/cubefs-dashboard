<!--
 Copyright 2026 The CubeFS Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0
-->

<template>
  <div class="shard-grid">
    <div class="head">
      <div class="metric-switch">
        <el-radio-group v-model="metric" size="mini">
          <el-radio-button v-if="!isMetadataMode" label="throughputMBs">吞吐 MB/s</el-radio-button>
          <el-radio-button label="opsPerSec">{{ isMetadataMode ? '操作速率 ops/sec' : 'IOPS' }}</el-radio-button>
          <el-radio-button v-if="!isMetadataMode" label="p99">P99 延迟 μs</el-radio-button>
        </el-radio-group>
      </div>
      <div class="hint">{{ subtitle }}</div>
    </div>
    <div ref="chart" class="heatmap"></div>
  </div>
</template>

<script>
import Echarts from 'echarts'

export default {
  name: 'BenchShardHeatmap',
  props: {
    // Each shard summary: { shardIdx, nodeAddr, status, error, stages: [{name, throughputMBs, opsPerSec, latency:{p50,p95,p99,p999,mean}, totalOps}] }
    shards: { type: Array, default: () => [] },
    // 'mdtest' | 's3' | 'posix' | '' — drives metric availability.
    // When 'mdtest', only opsPerSec is meaningful (throughput / latency not reported).
    storageType: { type: String, default: '' },
  },
  data() {
    return {
      inst: null,
      resizeHandler: null,
      metric: 'throughputMBs',
    }
  },
  computed: {
    // Metadata benchmarks (mdtest) only carry opsPerSec / totalOps signal.
    // Falls back to a data-shape heuristic when storageType prop is absent
    // (legacy task records without rule metadata wired through).
    isMetadataMode() {
      if (this.storageType === 'mdtest') return true
      if (!Array.isArray(this.shards) || this.shards.length === 0) return false
      let sawStage = false
      for (const sh of this.shards) {
        for (const st of (sh && sh.stages) || []) {
          sawStage = true
          if (Number(st && st.throughputMBs) > 0) return false
          const l = st && st.latency
          if (l && (Number(l.p50) > 0 || Number(l.p95) > 0 || Number(l.p99) > 0 || Number(l.p999) > 0 || Number(l.mean) > 0)) {
            return false
          }
        }
      }
      // All stages had zero throughput AND zero latency — treat as metadata.
      return sawStage
    },
    stageNames() {
      const set = new Set()
      for (const s of this.shards) {
        for (const st of (s.stages || [])) set.add(st.name || '-')
      }
      return Array.from(set)
    },
    subtitle() {
      const total = this.shards.length
      const withData = this.shards.filter(s => s.stages && s.stages.length).length
      return `${withData}/${total} shard 上报了 stage 数据`
    },
  },
  watch: {
    shards: { handler() { this.$nextTick(this.render) }, deep: true },
    metric() { this.$nextTick(this.render) },
    storageType() { this.$nextTick(this.render) },
    // When entering metadata mode the previously selected throughput / p99
    // metric would render an empty heatmap; force-switch to opsPerSec.
    isMetadataMode: {
      immediate: true,
      handler(v) {
        if (v && this.metric !== 'opsPerSec') this.metric = 'opsPerSec'
      },
    },
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
    cellValue(stage, metric) {
      if (!stage) return null
      if (metric === 'throughputMBs' || metric === 'opsPerSec') {
        const v = stage[metric]
        return v == null ? null : Number(v)
      }
      if (metric === 'p99') {
        return stage.latency && stage.latency.p99 != null ? Number(stage.latency.p99) : null
      }
      return null
    },
    render() {
      if (!this.$refs.chart) return
      if (!this.inst) this.inst = Echarts.init(this.$refs.chart)
      const shards = this.shards
      const stageNames = this.stageNames
      const data = []
      let maxVal = 0
      for (let yi = 0; yi < shards.length; yi++) {
        const sh = shards[yi]
        for (let xi = 0; xi < stageNames.length; xi++) {
          const stage = (sh.stages || []).find(st => (st.name || '-') === stageNames[xi])
          const v = this.cellValue(stage, this.metric)
          if (v != null && v > maxVal) maxVal = v
          data.push([xi, yi, v == null ? '-' : Number(v.toFixed(this.metric === 'opsPerSec' ? 0 : 2))])
        }
      }
      const yLabels = shards.map(s => `#${s.shardIdx} ${s.nodeAddr ? '@' + s.nodeAddr.split(':')[0] : ''}`)
      const isLatency = this.metric === 'p99'
      this.inst.setOption({
        tooltip: {
          position: 'top',
          formatter: (p) => {
            const sh = shards[p.value[1]]
            const stage = stageNames[p.value[0]]
            const val = p.value[2]
            const unit = this.metric === 'throughputMBs'
              ? 'MB/s'
              : (this.metric === 'opsPerSec' ? (this.isMetadataMode ? 'ops/sec' : 'ops/s') : 'μs')
            const nodeLine = sh ? `节点: ${sh.nodeAddr || '-'}<br/>状态: ${sh.status || '-'}` : ''
            return `Shard #${sh && sh.shardIdx} · Stage <b>${stage}</b><br/>${nodeLine}<br/><b>${val}</b> ${unit}`
          },
        },
        grid: { left: 110, right: 20, top: 30, bottom: 70, containLabel: true },
        xAxis: {
          type: 'category', data: stageNames.map(n => n && n.length > 14 ? '…' + n.slice(n.length - 13) : n), splitArea: { show: true },
          axisLabel: { rotate: stageNames.length > 3 ? 45 : 0, fontSize: 11, interval: 0 },
        },
        yAxis: { type: 'category', data: yLabels, splitArea: { show: true } },
        visualMap: {
          min: 0,
          max: maxVal > 0 ? maxVal : 1,
          calculable: false,
          orient: 'horizontal',
          left: 'center', bottom: 8,
          // 延迟数据：值越低越好 → 反转色阶（低=绿，高=红）
          inRange: isLatency
            ? { color: ['#67c23a', '#fac858', '#ee6666'] }
            : { color: ['#eef3f7', '#79bbff', '#409eff', '#1d70c4'] },
          text: isLatency ? ['高 (差)', '低 (好)'] : ['高 (好)', '低'],
        },
        series: [{
          type: 'heatmap',
          data,
          label: { show: stageNames.length * shards.length <= 36, fontSize: 10 },
          emphasis: { itemStyle: { shadowBlur: 8, shadowColor: 'rgba(0,0,0,0.5)' } },
        }],
      }, true)
    },
  },
}
</script>

<style lang="scss" scoped>
.shard-grid {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 10px 12px;
  margin-bottom: 14px;
}
.head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.hint {
  font-size: 12px;
  color: #909399;
}
.heatmap {
  width: 100%;
  height: 320px;
}
</style>
