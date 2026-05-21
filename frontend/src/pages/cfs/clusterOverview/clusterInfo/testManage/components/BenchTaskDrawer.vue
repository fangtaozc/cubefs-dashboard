<!--
 Copyright 2023 The CubeFS Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
 implied. See the License for the specific language governing
 permissions and limitations under the License.
-->

<template>
  <el-drawer
    :visible.sync="innerVisible"
    :title="$t('bench.viewdetail')"
    direction="rtl"
    size="880px"
    append-to-body
  >
    <div v-if="task" class="drawer-content">
      <!-- Metadata -->
      <el-descriptions border :column="2" size="small" class="section">
        <el-descriptions-item :label="$t('bench.taskid')">{{ task.taskID }}</el-descriptions-item>
        <el-descriptions-item :label="$t('bench.ruleid')">{{ task.ruleID || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('bench.taskstatus')">
          <el-tag :type="statusTagType(task.status)" size="mini" disable-transitions>{{ task.status || '-' }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('common.createtime')">{{ formatTime(task.createdAt) }}</el-descriptions-item>
        <el-descriptions-item v-if="task.error" :label="$t('common.taskerror')" :span="2">
          <span style="color: #f56c6c">{{ task.error }}</span>
        </el-descriptions-item>
      </el-descriptions>

      <!-- Fan-out shard table (parent task with shards) -->
      <div v-if="task.shards && task.shards.length > 0" class="section">
        <div class="section-title">Shards ({{ task.shardsDone }}/{{ task.shardTotal }})</div>
        <BenchShardHeatmap v-if="hasShardStages" :shards="task.shards" :storage-type="task.storageType || ''" />
        <el-table :data="task.shards" border size="small">
          <el-table-column :label="$t('bench.shardidx')" prop="shardIdx" width="80"></el-table-column>
          <el-table-column :label="$t('bench.nodeaddr')" prop="nodeAddr" min-width="160"></el-table-column>
          <el-table-column :label="$t('bench.duration')" min-width="110">
            <template slot-scope="scope">
              <span>{{ scope.row.duration ? formatDuration(scope.row.duration) : '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="$t('common.taskerror')" min-width="160">
            <template slot-scope="scope">
              <span :style="scope.row.error ? 'color:#f56c6c' : ''">{{ scope.row.error || '-' }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- Single-task bench result stages -->
      <div v-else-if="task.benchResult && task.benchResult.stages && task.benchResult.stages.length > 0" class="section">
        <div class="section-title">{{ $t('bench.stages') }}</div>
        <BenchStageCharts :stages="task.benchResult.stages" :storage-type="task.storageType || ''" />
        <el-table :data="task.benchResult.stages" border size="small">
          <el-table-column label="Stage" prop="name" min-width="120"></el-table-column>
          <el-table-column :label="$t('bench.throughput')" min-width="120">
            <template slot-scope="scope">
              <span>{{ scope.row.throughputMBs != null ? scope.row.throughputMBs.toFixed(2) + ' MB/s' : '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="$t('bench.iops')" min-width="100">
            <template slot-scope="scope">
              <span>{{ scope.row.opsPerSec != null ? scope.row.opsPerSec.toFixed(0) : '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="$t('bench.latencyp50')" min-width="110">
            <template slot-scope="scope">
              <span>{{ scope.row.latency && scope.row.latency.p50 != null ? scope.row.latency.p50.toFixed(0) + ' μs' : '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="$t('bench.latencyp99')" min-width="110">
            <template slot-scope="scope">
              <span>{{ scope.row.latency && scope.row.latency.p99 != null ? scope.row.latency.p99.toFixed(0) + ' μs' : '-' }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="$t('bench.duration')" min-width="110">
            <template slot-scope="scope">
              <span>{{ scope.row.durationSec != null ? scope.row.durationSec.toFixed(1) + ' s' : '-' }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <!-- No result data -->
      <el-empty
        v-else-if="task.status !== 'running'"
        :description="$t('component.nodata')"
      ></el-empty>
    </div>
  </el-drawer>
</template>

<script>
import { formatDate } from '@/utils'
import BenchStageCharts from './BenchStageCharts.vue'
import BenchShardHeatmap from './BenchShardHeatmap.vue'

export default {
  name: 'BenchTaskDrawer',
  components: { BenchStageCharts, BenchShardHeatmap },
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    task: {
      type: Object,
      default: null,
    },
  },
  computed: {
    innerVisible: {
      get() { return this.visible },
      set(val) { this.$emit('update:visible', val) },
    },
    hasShardStages() {
      const sh = this.task && this.task.shards
      return Array.isArray(sh) && sh.some(s => s && s.stages && s.stages.length > 0)
    },
  },
  methods: {
    formatTime(value) {
      if (!value) return '-'
      const d = new Date(value)
      if (isNaN(d.getTime()) || d.getFullYear() < 2000) return '-'
      return formatDate(value)
    },
    formatDuration(ms) {
      if (ms < 1000) return `${ms} ms`
      if (ms < 60000) return `${(ms / 1000).toFixed(1)} s`
      return `${Math.floor(ms / 60000)} m ${Math.round((ms % 60000) / 1000)} s`
    },
    statusTagType(status) {
      const map = {
        running: '',
        succeeded: 'success',
        failed: 'danger',
        cancelled: 'info',
      }
      return map[status] || 'info'
    },
  },
}
</script>

<style lang="scss" scoped>
.drawer-content {
  padding: 16px 20px;
}

.section {
  margin-bottom: 24px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 10px;
  color: #303133;
}
</style>
