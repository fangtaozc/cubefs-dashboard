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
  <div>
    <div class="toolbar">
      <div class="filters">
        <el-select v-model="localFilters.status" clearable size="small" :placeholder="$t('sync.state')">
          <el-option v-for="item in statuses" :key="item" :label="item" :value="item"></el-option>
        </el-select>
        <el-input v-model.trim="localFilters.ruleID" size="small" clearable placeholder="ruleID"></el-input>
        <el-input v-model.trim="localFilters.owner" size="small" clearable :placeholder="$t('sync.owner')"></el-input>
        <el-button size="small" type="primary" @click="loadData">{{ $t('button.search') }}</el-button>
        <el-button size="small" @click="resetFilters">{{ $t('button.reset') }}</el-button>
      </div>
      <div class="toolbar-actions">
        <span class="live-indicator" title="每秒自动刷新"><span class="live-dot"></span>live</span>
        <el-button size="small" :loading="loading" icon="el-icon-refresh" @click="loadData()">{{ $t('button.refresh') }}</el-button>
        <el-button
          v-auth="'CFS_SYNCNODE_DISPATCH'"
          type="primary"
          size="small"
          @click="openCreateDialog"
        >{{ $t('sync.dispatchtask') }}</el-button>
        <el-button
          v-auth="'CFS_SYNCTASK_EXPORT'"
          size="small"
          @click="exportTasks"
        >{{ $t('button.export') }}</el-button>
      </div>
    </div>

    <u-page-table :data="sortedDataList" :page-size="page.per_page" row-key="taskID" border>
      <el-table-column label="taskID" min-width="220">
        <template slot-scope="scope">
          <span>{{ getTaskId(scope.row) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="ruleID" min-width="160">
        <template slot-scope="scope">
          <span>{{ getRuleId(scope.row) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.type')" min-width="100" prop="type"></el-table-column>
      <el-table-column :label="$t('sync.state')" min-width="110">
        <template slot-scope="scope">
          <el-tag :type="statusTagType(scope.row.status)" size="mini" disable-transitions>{{ scope.row.status || '-' }}</el-tag>
        </template>
      </el-table-column>

      <!-- 节点 / 进度：每个分片一行，含进度条 -->
      <el-table-column label="节点 / 进度" min-width="300">
        <template slot-scope="scope">
          <template v-if="scope.row.shards && scope.row.shards.length > 0">
            <div
              v-for="(shard, idx) in scope.row.shards"
              :key="idx"
              class="shard-row"
            >
              <div class="shard-header">
                <a
                  v-if="shard.owner"
                  class="shard-owner"
                  @click="showWorker(shard.owner)"
                >{{ shard.owner }}</a>
                <span v-else class="shard-owner shard-owner--none">-</span>
                <el-tag
                  v-if="shard.status && shard.status !== 'running' && shard.status !== 'queued'"
                  :type="statusTagType(shard.status)"
                  size="mini"
                  disable-transitions
                  class="shard-status-tag"
                >{{ shard.status }}</el-tag>
                <el-button
                  v-if="shard.status === 'failed' && shard.taskID"
                  v-auth="'CFS_SYNCTASK_RETRY'"
                  type="text"
                  size="mini"
                  class="shard-retry-btn"
                  @click.stop="retryShardTask(shard.taskID)"
                >重试</el-button>
              </div>
              <template v-if="shard.progress && shard.progress.filesTotal > 0">
                <el-progress
                  :percentage="shard.status === 'succeeded' ? 100 : filesPct(shard.progress)"
                  :stroke-width="5"
                  :status="shardProgressStatus(shard.status, scope.row.status)"
                  :show-text="false"
                  :class="['shard-bar', isTerminalStatus(shard.status) ? 'bar-terminal' : '']"
                />
                <div class="shard-stats">
                  <span>{{ (shard.progress.filesDone || 0) + (shard.progress.filesSkipped || 0) }}/{{ shard.progress.filesTotal }}</span>
                  <span v-if="shard.progress.filesSkipped > 0" class="shard-skipped">跳过 {{ shard.progress.filesSkipped }}</span>
                  <span>{{ formatBytes(shard.progress.bytesDone) }}/{{ formatBytes(shard.progress.bytesTotal) }}</span>
                  <span v-if="shard.progress.bytesSkipped > 0" class="shard-skipped">跳过 {{ formatBytes(shard.progress.bytesSkipped) }}</span>
                  <span v-if="formatMBps(shard.progress.currentBandwidthMBps)" class="shard-throughput">{{ formatMBps(shard.progress.currentBandwidthMBps) }}</span>
                  <span v-else-if="formatMBps(shard.progress.throughputMBps)" class="shard-throughput shard-throughput--avg">均 {{ formatMBps(shard.progress.throughputMBps) }}</span>
                </div>
              </template>
              <div v-else class="shard-pending">
                {{ shard.status === 'running' || shard.status === 'queued' ? '列举中…' : '无进度' }}
              </div>
            </div>
          </template>
          <span v-else>-</span>
        </template>
      </el-table-column>

      <!-- 总进度 -->
      <el-table-column label="总进度" min-width="200">
        <template slot-scope="scope">
          <template v-if="scope.row.totalProgress && scope.row.totalProgress.filesTotal > 0">
            <div class="total-progress-item">
              <div class="total-progress-label">
                <span>文件</span>
                <span>{{ (scope.row.totalProgress.filesDone || 0) + (scope.row.totalProgress.filesSkipped || 0) }}/{{ scope.row.totalProgress.filesTotal }}</span>
              </div>
              <el-progress
                :percentage="scope.row.status === 'succeeded' ? 100 : filesPct(scope.row.totalProgress)"
                :stroke-width="5"
                :status="progressStatus(scope.row.status)"
                :show-text="false"
                :class="isTerminalStatus(scope.row.status) ? 'bar-terminal' : ''"
              />
            </div>
            <div class="total-progress-item" style="margin-top: 6px;">
              <div class="total-progress-label">
                <span>容量</span>
                <span>{{ formatBytes(scope.row.totalProgress.bytesDone) }}/{{ formatBytes(scope.row.totalProgress.bytesTotal) }}</span>
              </div>
              <el-progress
                :percentage="bytesPct(scope.row.totalProgress)"
                :stroke-width="5"
                :status="progressStatus(scope.row.status)"
                :show-text="false"
                :class="isTerminalStatus(scope.row.status) ? 'bar-terminal' : ''"
              />
            </div>
            <!-- 跳过汇总：文件数 + 字节数放在一行，比嵌入进度数字更醒目 -->
            <div v-if="scope.row.totalProgress.filesSkipped > 0 || scope.row.totalProgress.bytesSkipped > 0" class="total-skip-summary">
              <span class="total-skip-label">跳过</span>
              <span class="total-skip-value">
                {{ scope.row.totalProgress.filesSkipped }} 个文件
                <template v-if="scope.row.totalProgress.bytesSkipped > 0"> · {{ formatBytes(scope.row.totalProgress.bytesSkipped) }}</template>
              </span>
            </div>
            <template v-if="isTerminalStatus(scope.row.status)">
              <div v-if="formatMBps(computedFinalBandwidthMBps(scope.row))" class="total-progress-throughput">
                <span class="total-progress-throughput-label">总传输速率</span>
                <span class="total-progress-throughput-value total-progress-throughput-value--avg">{{ formatMBps(computedFinalBandwidthMBps(scope.row)) }}</span>
              </div>
            </template>
            <template v-else>
              <div v-if="formatMBps(scope.row.totalProgress.currentBandwidthMBps)" class="total-progress-throughput">
                <span class="total-progress-throughput-label">当前速率</span>
                <span class="total-progress-throughput-value">{{ formatMBps(scope.row.totalProgress.currentBandwidthMBps) }}</span>
              </div>
              <div v-if="formatMBps(scope.row.totalProgress.throughputMBps)" class="total-progress-throughput total-progress-throughput--avg">
                <span class="total-progress-throughput-label">均速</span>
                <span class="total-progress-throughput-value total-progress-throughput-value--avg">{{ formatMBps(scope.row.totalProgress.throughputMBps) }}</span>
              </div>
            </template>
          </template>
          <span v-else class="no-progress-text">
            {{ isActive(scope.row.status) ? '列举中…' : '-' }}
          </span>
        </template>
      </el-table-column>

      <!-- 开始时间（可排序） -->
      <el-table-column min-width="160">
        <template slot="header">
          <span class="sortable-header" @click="handleSort('startedAt')">
            {{ $t('sync.startedat') }}
            <i :class="sortIconClass('startedAt')"></i>
          </span>
        </template>
        <template slot-scope="scope">
          <span>{{ formatTime(scope.row.startedAt) }}</span>
        </template>
      </el-table-column>

      <!-- 结束时间（可排序） -->
      <el-table-column min-width="160">
        <template slot="header">
          <span class="sortable-header" @click="handleSort('doneAt')">
            {{ $t('sync.doneat') }}
            <i :class="sortIconClass('doneAt')"></i>
          </span>
        </template>
        <template slot-scope="scope">
          <span>{{ formatTime(scope.row.doneAt) }}</span>
        </template>
      </el-table-column>

      <el-table-column :label="$t('common.action')" min-width="180">
        <template slot-scope="scope">
          <MoreOPerate :count="4" :i18n="$i18n">
            <el-button type="text" @click="showDetail(scope.row)">{{ $t('common.detail') }}</el-button>
            <el-button v-auth="'CFS_SYNCTASK_CANCEL'" type="text" @click="cancelTask(scope.row)">{{ $t('sync.canceltask') }}</el-button>
            <el-button v-auth="'CFS_SYNCTASK_RETRY'" type="text" @click="retryTask(scope.row)">{{ $t('sync.retrytask') }}</el-button>
            <el-button v-auth="'CFS_SYNCTASK_DELETE'" type="text" class="danger-text" @click="deleteTask(scope.row)">{{ $t('common.delete') }}</el-button>
          </MoreOPerate>
        </template>
      </el-table-column>
    </u-page-table>

    <SyncTaskDispatchDialog
      :visible.sync="createVisible"
      :cluster-name="clusterName"
      :value="createPayload"
      @confirm="dispatchTask"
    />
    <SyncTaskDrawer
      :visible.sync="detailVisible"
      :task="detailRecord"
      @show-worker="showWorker"
    />
  </div>
</template>

<script>
import { download, formatDate } from '@/utils'
import MoreOPerate from '@/pages/components/moreOPerate'
import UPageTable from '@/pages/components/uPageTable'
import {
  cancelSyncTask,
  deleteSyncTask,
  dispatchSyncTask,
  getSyncTask,
  getSyncTaskExportUrl,
  getSyncTaskList,
  retrySyncTask,
} from '@/api/cfs/cluster'
import SyncTaskDispatchDialog from './SyncTaskDispatchDialog.vue'
import SyncTaskDrawer from './SyncTaskDrawer.vue'

export default {
  name: 'SyncTaskTab',
  components: {
    MoreOPerate,
    SyncTaskDispatchDialog,
    SyncTaskDrawer,
    UPageTable,
  },
  props: {
    clusterName: {
      type: String,
      required: true,
    },
    filters: {
      type: Object,
      default() {
        return {}
      },
    },
  },
  data() {
    return {
      statuses: ['queued', 'running', 'succeeded', 'failed', 'cancelled', 'cancelling'],
      localFilters: {
        status: '',
        ruleID: '',
        owner: '',
      },
      dataList: [],
      loading: false,
      sortBy: 'startedAt',
      sortOrder: 'desc',
      detailVisible: false,
      detailRecord: {},
      createVisible: false,
      createPayload: '',
      page: {
        per_page: 10,
      },
      refreshTimer: null,
    }
  },
  computed: {
    hasActiveTasks() {
      return this.dataList.some(t => this.isActive(t.status))
    },
    sortedDataList() {
      if (!this.dataList.length) return []
      const list = [...this.dataList]
      const { sortBy, sortOrder } = this
      const factor = sortOrder === 'asc' ? 1 : -1
      list.sort((a, b) => {
        let av = a[sortBy]
        let bv = b[sortBy]
        if (sortBy === 'startedAt' || sortBy === 'doneAt') {
          av = av ? new Date(av).getTime() : 0
          bv = bv ? new Date(bv).getTime() : 0
        }
        if (av < bv) return -1 * factor
        if (av > bv) return 1 * factor
        return 0
      })
      return list
    },
  },
  watch: {
    filters: {
      deep: true,
      immediate: true,
      handler(val) {
        this.localFilters = {
          status: val?.status || '',
          ruleID: val?.ruleID || '',
          owner: val?.owner || '',
        }
        this.loadData()
      },
    },
  },
  mounted() {
    this.refreshTimer = setInterval(() => {
      this.loadData(true)
    }, 1000)
  },
  beforeDestroy() {
    if (this.refreshTimer) {
      clearInterval(this.refreshTimer)
      this.refreshTimer = null
    }
  },
  methods: {
    async loadData(silent = false) {
      if (!silent) this.loading = true
      try {
        const { data } = await getSyncTaskList({
          cluster_name: this.clusterName,
          ...this.localFilters,
        })
        if (silent) {
          this.patchDataList(data || [])
        } else {
          this.dataList = data || []
        }
      } finally {
        if (!silent) this.loading = false
      }
    },
    // Silent auto-refresh: patch rows in-place and only trigger Vue reactivity
    // when content actually changed (JSON fingerprint). Rows with identical data
    // are skipped entirely — no setter calls, no re-render.
    patchDataList(newList) {
      const newById = new Map(newList.map(t => [t.taskID, t]))
      const oldById = new Map(this.dataList.map((t, i) => [t.taskID, i]))
      for (const [id, item] of newById) {
        const idx = oldById.get(id)
        if (idx !== undefined) {
          const existing = this.dataList[idx]
          if (JSON.stringify(existing) !== JSON.stringify(item)) {
            Object.assign(existing, item)
          }
        }
      }
      for (let i = this.dataList.length - 1; i >= 0; i--) {
        if (!newById.has(this.dataList[i].taskID)) {
          this.dataList.splice(i, 1)
        }
      }
      for (const item of newList) {
        if (!oldById.has(item.taskID)) {
          this.dataList.push(item)
        }
      }
    },
    resetFilters() {
      this.localFilters = {
        status: '',
        ruleID: '',
        owner: '',
      }
      this.loadData()
    },
    formatTime(value) {
      if (!value) return '-'
      const d = new Date(value)
      if (isNaN(d.getTime()) || d.getFullYear() < 2000) return '-'
      return formatDate(value)
    },
    formatBytes(bytes) {
      if (!bytes || bytes === 0) return '0 B'
      const units = ['B', 'KB', 'MB', 'GB', 'TB']
      const i = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
      return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
    },
    formatMBps(mbps) {
      if (!mbps || mbps <= 0) return null
      if (mbps >= 1) return mbps.toFixed(1) + ' MB/s'
      return (mbps * 1024).toFixed(0) + ' KB/s'
    },
    filesPct(progress) {
      if (!progress || !progress.filesTotal) return 0
      const processed = (progress.filesDone || 0) + (progress.filesSkipped || 0)
      return Math.min(Math.round((processed / progress.filesTotal) * 100), 100)
    },
    bytesPct(progress) {
      if (!progress || !progress.bytesTotal) return 0
      return Math.min(Math.round((progress.bytesDone / progress.bytesTotal) * 100), 100)
    },
    progressStatus(status) {
      if (status === 'succeeded') return 'success'
      if (status === 'failed') return 'exception'
      if (status === 'cancelled') return 'warning'
      return null
    },
    shardProgressStatus(shardStatus, parentStatus) {
      if (shardStatus === 'succeeded') return 'success'
      if (shardStatus === 'cancelled') return 'warning'
      if (shardStatus === 'failed') {
        // Parent still running — shard failure is not yet final (other shards may succeed).
        // Show as warning (yellow) instead of exception (red) to avoid false alarm.
        return (parentStatus === 'running' || parentStatus === 'queued') ? 'warning' : 'exception'
      }
      return null
    },
    isTerminalStatus(status) {
      return status === 'succeeded' || status === 'failed' || status === 'cancelled'
    },
    computedFinalBandwidthMBps(row) {
      if (!row?.startedAt || !row?.doneAt || !row?.totalProgress?.bytesDone) return null
      const elapsed = (new Date(row.doneAt) - new Date(row.startedAt)) / 1000
      if (elapsed <= 0) return null
      return row.totalProgress.bytesDone / 1024 / 1024 / elapsed
    },
    isActive(status) {
      return status === 'running' || status === 'queued'
    },
    statusTagType(status) {
      const map = {
        queued: 'info',
        running: '',
        succeeded: 'success',
        failed: 'danger',
        cancelled: 'warning',
        cancelling: 'warning',
      }
      return map[status] ?? 'info'
    },
    handleSort(field) {
      if (this.sortBy === field) {
        this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc'
      } else {
        this.sortBy = field
        this.sortOrder = 'desc'
      }
    },
    sortIconClass(field) {
      if (this.sortBy !== field) return 'el-icon-d-caret'
      return this.sortOrder === 'asc' ? 'el-icon-caret-top' : 'el-icon-caret-bottom'
    },
    getTaskId(row) {
      return row?.taskID || row?.taskId || row?.id || '-'
    },
    getRuleId(row) {
      return row?.ruleID || row?.ruleId || row?.request?.ruleId || row?.Request?.ruleId || '-'
    },
    buildSampleTaskPayload() {
      const taskID = `manual-${Date.now()}`
      return JSON.stringify({
        id: taskID,
        opcode: 121,
        Request: {
          taskId: taskID,
          ruleId: '<填写规则 ID>',
        },
      }, null, 2)
    },
    openCreateDialog() {
      this.createPayload = this.buildSampleTaskPayload()
      this.createVisible = true
    },
    showWorker(owner) {
      this.$emit('show-worker', owner)
    },
    async showDetail(row) {
      const { data } = await getSyncTask({
        cluster_name: this.clusterName,
        id: this.getTaskId(row),
      })
      this.detailRecord = data || {}
      this.detailVisible = true
    },
    async cancelTask(row) {
      await cancelSyncTask({
        cluster_name: this.clusterName,
        id: this.getTaskId(row),
      })
      this.$message.success(this.$t('sync.canceltask') + this.$t('common.xxsuc'))
      this.loadData()
    },
    async retryTask(row) {
      await retrySyncTask({
        cluster_name: this.clusterName,
        id: this.getTaskId(row),
      })
      this.$message.success(this.$t('sync.retrytask') + this.$t('common.xxsuc'))
      this.loadData()
    },
    async retryShardTask(shardTaskID) {
      await retrySyncTask({
        cluster_name: this.clusterName,
        id: shardTaskID,
      })
      this.$message.success(this.$t('sync.retrytask') + this.$t('common.xxsuc'))
      this.loadData()
    },
    async deleteTask(row) {
      const id = this.getTaskId(row)
      try {
        await this.$confirm(
          `${this.$t('common.confirmto')}${this.$t('common.delete')} ${id}？`,
          this.$t('common.notice'),
          {
            confirmButtonText: this.$t('common.yes'),
            cancelButtonText: this.$t('common.no'),
            type: 'warning',
          },
        )
      } catch {
        return
      }
      await deleteSyncTask({ cluster_name: this.clusterName, id })
      this.$message.success(this.$t('common.delete') + this.$t('common.xxsuc'))
      this.loadData()
    },
    async dispatchTask(text) {
      let body = null
      try {
        body = JSON.parse(text)
      } catch (error) {
        this.$message.error(this.$t('common.failed') + ': JSON parse error')
        return
      }
      const { data } = await dispatchSyncTask({
        cluster_name: this.clusterName,
        ...body,
      })
      this.createVisible = false
      this.$message.success(`${this.$t('sync.dispatchtask')}${this.$t('common.xxsuc')}`)
      if (data?.taskID || data?.taskId || data?.id) {
        this.localFilters = {
          ...this.localFilters,
          ruleID: body?.Request?.ruleId || body?.Request?.ruleID || body?.ruleID || this.localFilters.ruleID,
        }
      }
      this.loadData()
    },
    exportTasks() {
      const url = getSyncTaskExportUrl({
        cluster_name: this.clusterName,
        ...this.localFilters,
      })
      download(url, 'sync-tasks.ndjson')
    },
  },
}
</script>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.filters {
  display: flex;
  align-items: center;
  gap: 8px;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.live-indicator {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #67c23a;
  font-weight: 500;
  letter-spacing: 0.5px;
}

.live-dot {
  display: inline-block;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #67c23a;
  animation: live-pulse 1.5s ease-in-out infinite;
}

@keyframes live-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.4; transform: scale(0.7); }
}

.filters ::v-deep .el-input,
.filters ::v-deep .el-select {
  width: 180px;
}

/* 分片行 */
.shard-row {
  padding: 4px 0;

  & + .shard-row {
    border-top: 1px solid #f0f0f0;
    margin-top: 4px;
    padding-top: 6px;
  }
}

.shard-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
  flex-wrap: wrap;
}

.shard-owner {
  font-size: 12px;
  color: #409eff;
  cursor: pointer;
  word-break: break-all;

  &--none {
    color: #909399;
    cursor: default;
  }
}

.shard-status-tag {
  flex-shrink: 0;
}

.shard-retry-btn {
  flex-shrink: 0;
  padding: 0;
  font-size: 11px;
  color: #e6a23c;

  &:hover {
    color: #cf9236;
  }
}

.shard-bar {
  margin-bottom: 2px;

  ::v-deep .el-progress-bar__inner {
    transition: width 4.5s linear;
  }
}

.bar-terminal ::v-deep .el-progress-bar__inner {
  transition: none !important;
}

.shard-stats {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: #909399;
}

.shard-skipped {
  color: #e6a23c;
  flex-shrink: 0;
}

.shard-throughput {
  color: #409eff;
  flex-shrink: 0;

  &--avg {
    color: #909399;
  }
}

.shard-pending {
  font-size: 11px;
  color: #c0c4cc;
  font-style: italic;
}

/* 总进度 */
.total-progress-item {
  font-size: 12px;

  ::v-deep .el-progress-bar__inner {
    transition: width 4.5s linear;
  }
}

.total-progress-label {
  display: flex;
  justify-content: space-between;
  color: #606266;
  margin-bottom: 3px;
}

.total-progress-skip-badge {
  margin-left: 6px;
  font-size: 11px;
  color: #e6a23c;
}

.total-skip-summary {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 5px;
  padding: 3px 6px;
  border-radius: 4px;
  background: #fdf6ec;
  font-size: 11px;
}

.total-skip-label {
  color: #b87333;
  font-weight: 500;
  flex-shrink: 0;
}

.total-skip-value {
  color: #e6a23c;
}

.total-progress-throughput {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  margin-top: 5px;

  &--avg {
    margin-top: 2px;
  }
}

.total-progress-throughput-label {
  color: #909399;
}

.total-progress-throughput-value {
  color: #409eff;
  font-weight: 500;

  &--avg {
    color: #909399;
    font-weight: 400;
  }
}

.no-progress-text {
  font-size: 12px;
  color: #c0c4cc;
  font-style: italic;
}

/* 可排序列标题 */
.sortable-header {
  cursor: pointer;
  user-select: none;

  &:hover {
    color: #409eff;
  }

  i {
    margin-left: 4px;
    font-size: 12px;
  }
}
</style>
