<!--
 Copyright 2026 The CubeFS Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0
-->

<template>
  <el-drawer
    :visible.sync="innerVisible"
    direction="rtl"
    size="960px"
    append-to-body
    :title="title"
    @open="onOpen"
  >
    <div class="trend-content">
      <div class="header">
        <div class="meta">
          <span><b>规则：</b>{{ rule && rule.name || '-' }}</span>
          <span class="rid">{{ rule && rule.id }}</span>
          <span class="hint">最近 {{ tasks.length }} 次成功运行（限制 {{ limit }} 条）</span>
        </div>
        <div class="actions">
          <el-radio-group v-model="metric" size="small">
            <el-radio-button v-if="!isMetadataMode" label="throughputMBs">吞吐</el-radio-button>
            <el-radio-button label="opsPerSec">{{ isMetadataMode ? '操作速率' : 'IOPS' }}</el-radio-button>
            <el-radio-button v-if="!isMetadataMode" label="p50">P50</el-radio-button>
            <el-radio-button v-if="!isMetadataMode" label="p99">P99</el-radio-button>
            <el-radio-button v-if="!isMetadataMode" label="p999">P999</el-radio-button>
          </el-radio-group>
          <el-button size="small" plain :loading="loading" icon="el-icon-refresh" @click="loadData">刷新</el-button>
        </div>
      </div>

      <div v-if="!loading && points.length === 0" class="empty-tip">
        该规则尚无成功完成的任务。运行规则、等任务终态为 succeeded 后再来查看趋势。
      </div>

      <BenchTrendChart v-else :points="points" :metric="metric" :storage-type="ruleStorageType" />

      <div v-if="warn" class="warn">{{ warn }}</div>

      <div class="legend-help">
        <h4>聚合规则</h4>
        <ul>
          <li><b>单 shard 任务</b>：直接使用 task.benchResult.stages 数据</li>
          <li><b>fan-out 任务</b>：按 stage 名聚合所有 shard：吞吐 / IOPS = 求和，延迟（P50/P95/P99/P999/mean）= 取所有 shard 的最大值（保守估计长尾）</li>
          <li>缺数据的点（旧记录、运行失败）会以折线断点形式跳过</li>
        </ul>
      </div>
    </div>
  </el-drawer>
</template>

<script>
import { getBenchTaskList } from '@/api/cfs/cluster'
import BenchTrendChart from './BenchTrendChart.vue'

const DEFAULT_LIMIT = 30

export default {
  name: 'BenchTrendsDrawer',
  components: { BenchTrendChart },
  props: {
    visible: { type: Boolean, default: false },
    clusterName: { type: String, required: true },
    rule: { type: Object, default: null },
  },
  data() {
    return {
      loading: false,
      tasks: [],
      metric: 'throughputMBs',
      limit: DEFAULT_LIMIT,
      warn: '',
    }
  },
  computed: {
    innerVisible: {
      get() { return this.visible },
      set(val) { this.$emit('update:visible', val) },
    },
    title() {
      return `历史趋势 - ${this.rule && this.rule.name || (this.rule && this.rule.id) || ''}`
    },
    ruleStorageType() {
      return (this.rule && this.rule.storageType) || ''
    },
    // mdtest rules only carry opsPerSec signal; hide throughput / latency
    // metric switches and default to opsPerSec.
    isMetadataMode() {
      return this.ruleStorageType === 'mdtest'
    },
    // Transform each task record into a single { taskID, createdAt, stages[] }
    // point for the chart. Drops tasks that produced no stage data so the
    // line doesn't carry confusing zero points.
    points() {
      const out = []
      for (const t of this.tasks) {
        const agg = aggregateTask(t)
        if (agg && agg.stages && agg.stages.length > 0) out.push(agg)
      }
      return out
    },
  },
  watch: {
    // If switching to a metadata rule with an incompatible metric selected,
    // snap back to opsPerSec so the chart isn't blank.
    isMetadataMode: {
      immediate: true,
      handler(v) {
        if (v && this.metric !== 'opsPerSec') this.metric = 'opsPerSec'
      },
    },
  },
  methods: {
    async onOpen() {
      this.metric = this.isMetadataMode ? 'opsPerSec' : 'throughputMBs'
      this.warn = ''
      await this.loadData()
    },
    async loadData() {
      if (!this.rule || !this.rule.id) return
      this.loading = true
      try {
        const { data } = await getBenchTaskList({
          cluster_name: this.clusterName,
          ruleID: this.rule.id,
          status: 'succeeded',
        })
        const raw = Array.isArray(data) ? data : (data?.data || [])
        // Drop child shard records (ParentTaskID != ''); only keep the parent
        // (fan-out) or stand-alone single-shard tasks. Parent records have
        // shards[] populated; single-shard records have benchResult.stages.
        const filtered = raw.filter(t => !t.parentTaskID)
        // Sort by createdAt asc and cap to most recent N.
        filtered.sort((a, b) => (a.createdAt || 0) - (b.createdAt || 0))
        this.tasks = filtered.slice(Math.max(0, filtered.length - this.limit))
        const skipped = this.tasks.filter(t => !aggregateTask(t) || !aggregateTask(t).stages.length).length
        this.warn = skipped > 0
          ? `${skipped} / ${this.tasks.length} 个任务无 stage 数据（升级前的旧记录），已从折线中跳过。`
          : ''
      } catch (e) {
        this.$message.error('加载历史失败：' + (e.message || e))
        this.tasks = []
      } finally {
        this.loading = false
      }
    },
  },
}

// aggregateTask reduces one bench task record to a single { taskID, createdAt, stages[] }
// snapshot. Three cases:
//   1. Single shard: task.benchResult.stages directly.
//   2. Fan-out:      task.shards[*].stages summed (throughput/IOPS) or maxed (latency).
//   3. No data:      returns null.
function aggregateTask(t) {
  if (!t) return null
  const base = { taskID: t.taskID, createdAt: t.createdAt, stages: [] }

  // Case 1: single shard
  if (t.benchResult && Array.isArray(t.benchResult.stages) && t.benchResult.stages.length > 0) {
    base.stages = t.benchResult.stages.map(s => ({
      name: s.name,
      throughputMBs: s.throughputMBs,
      opsPerSec: s.opsPerSec,
      latency: s.latency ? { ...s.latency } : null,
    }))
    return base
  }

  // Case 2: fan-out parent
  if (Array.isArray(t.shards) && t.shards.length > 0) {
    const byName = new Map() // stageName -> { count, throughputMBs(sum), opsPerSec(sum), latency(max) }
    let anyStage = false
    for (const sh of t.shards) {
      if (!sh || !Array.isArray(sh.stages)) continue
      for (const st of sh.stages) {
        anyStage = true
        const name = st.name || '-'
        const acc = byName.get(name) || {
          name,
          throughputMBs: 0,
          opsPerSec: 0,
          latency: { p50: 0, p95: 0, p99: 0, p999: 0, mean: 0 },
          _haveLat: false,
        }
        if (st.throughputMBs != null) acc.throughputMBs += Number(st.throughputMBs)
        if (st.opsPerSec != null) acc.opsPerSec += Number(st.opsPerSec)
        if (st.latency) {
          acc._haveLat = true
          acc.latency.p50  = Math.max(acc.latency.p50,  Number(st.latency.p50  || 0))
          acc.latency.p95  = Math.max(acc.latency.p95,  Number(st.latency.p95  || 0))
          acc.latency.p99  = Math.max(acc.latency.p99,  Number(st.latency.p99  || 0))
          acc.latency.p999 = Math.max(acc.latency.p999, Number(st.latency.p999 || 0))
          acc.latency.mean = Math.max(acc.latency.mean, Number(st.latency.mean || 0))
        }
        byName.set(name, acc)
      }
    }
    if (!anyStage) return base
    base.stages = Array.from(byName.values()).map(s => ({
      name: s.name,
      throughputMBs: s.throughputMBs,
      opsPerSec: s.opsPerSec,
      latency: s._haveLat ? s.latency : null,
    }))
    return base
  }
  return base
}
</script>

<style lang="scss" scoped>
.trend-content {
  padding: 0 18px 18px;
}
.header {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.meta {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #303133;
  font-size: 13px;
  .rid {
    color: #909399;
    font-family: monospace;
    font-size: 12px;
  }
  .hint {
    color: #909399;
    font-size: 12px;
  }
}
.actions {
  display: flex;
  gap: 8px;
}
.empty-tip {
  color: #909399;
  font-size: 13px;
  padding: 40px 0;
  text-align: center;
  border: 1px dashed #dcdfe6;
  border-radius: 4px;
  background: #fafbfc;
}
.warn {
  color: #e6a23c;
  font-size: 12px;
  margin-top: 8px;
}
.legend-help {
  margin-top: 14px;
  background: #f5f7fa;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 10px 14px;
  font-size: 12px;
  color: #606266;
  h4 { margin: 0 0 6px 0; font-size: 13px; color: #303133; }
  ul { margin: 0; padding-left: 18px; line-height: 1.7; }
}
</style>
