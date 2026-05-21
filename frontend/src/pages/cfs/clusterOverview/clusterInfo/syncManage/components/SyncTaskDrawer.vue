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
  <el-dialog
    :visible.sync="innerVisible"
    width="920px"
    top="6vh"
    append-to-body
    custom-class="sync-task-dialog"
    @close="handleClose"
  >
    <div slot="title" class="dialog-title">
      <div>
        <div class="dialog-title__label">{{ $t('sync.tasks') }}</div>
        <div class="dialog-title__value">{{ taskId }}</div>
      </div>
      <el-tag size="small" :type="statusTagType">{{ statusText }}</el-tag>
    </div>
    <div class="dialog-body">
      <div class="summary-grid summary-grid--2col">
        <div class="summary-card">
          <div class="summary-card__label">{{ $t('sync.startedat') }}</div>
          <div class="summary-card__value">{{ formatTime(task && task.startedAt) }}</div>
        </div>
        <div class="summary-card">
          <div class="summary-card__label">{{ $t('sync.doneat') }}</div>
          <div class="summary-card__value">{{ formatTime(task && task.doneAt) }}</div>
        </div>
      </div>

      <div v-if="hasDataFlow" class="section">
        <div class="section-title">{{ $t('sync.datapath') }}</div>
        <div class="dataflow-row">
          <div class="dataflow-ep">
            <div class="dataflow-ep__kind" :class="`dataflow-ep__kind--${srcConfig.kind}`">{{ srcConfig.kind.toUpperCase() }}</div>
            <div class="dataflow-ep__name">{{ srcConfig.name }}</div>
            <div v-if="srcConfig.sub" class="dataflow-ep__sub">{{ srcConfig.sub }}</div>
            <div v-else class="dataflow-ep__sub dataflow-ep__sub--muted">（根目录）</div>
          </div>
          <div class="dataflow-connector" :class="flowConnectorClass">
            <div class="dataflow-connector__track">
              <div class="dataflow-connector__fill"></div>
              <div v-if="isActive" class="dataflow-connector__shine"></div>
            </div>
            <svg class="dataflow-connector__arrowhead" width="12" height="12" viewBox="0 0 12 12">
              <polygon points="0,0 12,6 0,12" :fill="arrowColor"/>
            </svg>
          </div>
          <div class="dataflow-ep">
            <div class="dataflow-ep__kind" :class="`dataflow-ep__kind--${dstConfig.kind}`">{{ dstConfig.kind.toUpperCase() }}</div>
            <div class="dataflow-ep__name">{{ dstConfig.name }}</div>
            <div v-if="dstConfig.sub" class="dataflow-ep__sub">{{ dstConfig.sub }}</div>
            <div v-else class="dataflow-ep__sub dataflow-ep__sub--muted">（根目录）</div>
          </div>
        </div>
      </div>

      <div class="section">
        <div class="section-title">{{ $t('sync.basicinfo') }}</div>
        <div class="info-grid">
          <div class="info-item">
            <div class="info-item__label">taskID</div>
            <div class="info-item__value">{{ taskId }}</div>
          </div>
          <div class="info-item">
            <div class="info-item__label">ruleID</div>
            <div class="info-item__value">{{ ruleId }}</div>
          </div>
          <div class="info-item">
            <div class="info-item__label">{{ $t('common.type') }}</div>
            <div class="info-item__value">{{ typeText }}</div>
          </div>
          <div class="info-item">
            <div class="info-item__label">{{ $t('sync.state') }}</div>
            <div class="info-item__value">{{ statusText }}</div>
          </div>
        </div>
      </div>

      <div v-if="errorText !== '-'" class="section">
        <div class="section-title">{{ $t('sync.taskerror') }}</div>
        <div class="error-panel">{{ errorText }}</div>
      </div>

      <el-tabs v-model="activeName">
        <el-tab-pane :label="$t('sync.progress')" name="progress">
          <div v-if="hasProgress" class="progress-section">
            <!-- 带宽摘要 -->
            <div class="bw-row">
              <div v-if="currentBandwidthText" class="bw-card bw-card--current">
                <div class="bw-card__label">当前速率</div>
                <div class="bw-card__value">{{ currentBandwidthText }}</div>
              </div>
              <div v-if="throughputText !== '-'" class="bw-card" :class="currentBandwidthText ? 'bw-card--avg' : ''">
                <div class="bw-card__label">{{ progressBarStatus !== null ? '总传输速率' : '平均带宽' }}</div>
                <div class="bw-card__value">{{ throughputText }}</div>
              </div>
            </div>
            <!-- 文件进度 -->
            <div class="progress-row">
              <div class="progress-row__header">
                <span class="progress-row__label">文件</span>
                <span class="progress-row__value">
                  {{ (task.progress.filesDone || 0) + (task.progress.filesSkipped || 0) }} / {{ task.progress.filesTotal }}
                  <span v-if="task.progress.filesSkipped > 0" class="progress-row__skip">（跳过 {{ task.progress.filesSkipped }}）</span>
                </span>
              </div>
              <el-progress
                :percentage="filesPct"
                :stroke-width="10"
                :status="progressBarStatus"
                :class="progressBarStatus !== null ? 'bar-terminal' : ''"
              />
            </div>
            <!-- 容量进度 -->
            <div class="progress-row" style="margin-top: 16px;">
              <div class="progress-row__header">
                <span class="progress-row__label">容量</span>
                <span class="progress-row__value">
                  {{ formatBytes((task.progress.bytesDone || 0) + (task.progress.bytesSkipped || 0)) }} / {{ formatBytes(task.progress.bytesTotal) }}
                </span>
              </div>
              <el-progress
                :percentage="bytesPct"
                :stroke-width="10"
                :status="progressBarStatus"
                :class="progressBarStatus !== null ? 'bar-terminal' : ''"
              />
            </div>
            <!-- 附加统计 -->
            <div class="progress-stats">
              <div class="progress-stat">
                <div class="progress-stat__label">跳过文件</div>
                <div class="progress-stat__value">{{ task.progress.filesSkipped }}</div>
              </div>
              <div v-if="task.progress.bytesSkipped > 0" class="progress-stat">
                <div class="progress-stat__label">跳过容量</div>
                <div class="progress-stat__value">{{ formatBytes(task.progress.bytesSkipped) }}</div>
              </div>
              <div class="progress-stat">
                <div class="progress-stat__label">失败</div>
                <div class="progress-stat__value progress-stat__value--danger">{{ task.progress.filesFailed }}</div>
              </div>
            </div>
          </div>
          <div v-else class="empty-text">
            {{ isActive ? '任务运行中，暂无进度数据' : '无进度数据' }}
          </div>
        </el-tab-pane>

        <el-tab-pane
          v-if="shards.length > 0"
          :label="`分片详情 (${shards.length})`"
          name="shards"
        >
          <el-table :data="shardsPagedData" size="small" border class="shard-detail-table">
            <el-table-column label="分片" width="60" align="center">
              <template slot-scope="scope">
                <span>{{ scope.row.shardIdx ?? scope.$index + (shardCurrentPage - 1) * shardPageSize }}</span>
              </template>
            </el-table-column>
            <el-table-column label="执行节点" min-width="150">
              <template slot-scope="scope">
                <a v-if="scope.row.owner" class="shard-owner-link" @click="$emit('show-worker', scope.row.owner)">{{ scope.row.owner }}</a>
                <span v-else class="muted">-</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="100" align="center">
              <template slot-scope="scope">
                <el-tag :type="shardTagType(scope.row.status)" size="mini" disable-transitions>{{ scope.row.status || '-' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="文件(完成/总)" min-width="130">
              <template slot-scope="scope">
                <template v-if="scope.row.progress && scope.row.progress.filesTotal > 0">
                  <span>{{ (scope.row.progress.filesDone || 0) + (scope.row.progress.filesSkipped || 0) }} / {{ scope.row.progress.filesTotal }}</span>
                </template>
                <span v-else class="muted">-</span>
              </template>
            </el-table-column>
            <el-table-column label="容量(完成/总)" min-width="160">
              <template slot-scope="scope">
                <template v-if="scope.row.progress && scope.row.progress.bytesTotal > 0">
                  <span>{{ formatBytes((scope.row.progress.bytesDone || 0) + (scope.row.progress.bytesSkipped || 0)) }} / {{ formatBytes(scope.row.progress.bytesTotal) }}</span>
                </template>
                <span v-else class="muted">-</span>
              </template>
            </el-table-column>
            <el-table-column label="跳过文件" width="90" align="right">
              <template slot-scope="scope">
                <span v-if="scope.row.progress && scope.row.progress.filesSkipped > 0" class="skip-num">{{ scope.row.progress.filesSkipped }}</span>
                <span v-else class="muted">-</span>
              </template>
            </el-table-column>
            <el-table-column label="跳过容量" width="100" align="right">
              <template slot-scope="scope">
                <span v-if="scope.row.progress && scope.row.progress.bytesSkipped > 0" class="skip-num">{{ formatBytes(scope.row.progress.bytesSkipped) }}</span>
                <span v-else class="muted">-</span>
              </template>
            </el-table-column>
            <el-table-column label="失败" width="70" align="right">
              <template slot-scope="scope">
                <span v-if="scope.row.progress && scope.row.progress.filesFailed > 0" class="fail-num">{{ scope.row.progress.filesFailed }}</span>
                <span v-else class="muted">-</span>
              </template>
            </el-table-column>
          </el-table>
          <div v-if="shards.length > shardPageSize" class="shard-pagination">
            <el-pagination
              :current-page.sync="shardCurrentPage"
              :page-size="shardPageSize"
              :total="shards.length"
              layout="total, prev, pager, next"
              small
            />
          </div>
        </el-tab-pane>

        <el-tab-pane
          v-if="skippedSamples.length > 0"
          :label="`跳过的文件 (${skippedSamples.length}${task.progress && task.progress.filesSkipped > skippedSamples.length ? '+' : ''})`"
          name="skipped"
        >
          <div class="skipped-note">
            共跳过 {{ task.progress && task.progress.filesSkipped || 0 }} 个文件（目标已存在且内容一致）。
            <span v-if="task.progress && task.progress.filesSkipped > skippedSamples.length">
              以下为前 {{ skippedSamples.length }} 条样本：
            </span>
          </div>
          <ul class="skipped-list">
            <li v-for="(key, i) in skippedSamples" :key="i" class="skipped-item">{{ key }}</li>
          </ul>
        </el-tab-pane>
        <el-tab-pane :label="$t('sync.rawdetail')" name="raw">
          <pre class="code-block">{{ taskText }}</pre>
        </el-tab-pane>
      </el-tabs>
    </div>
  </el-dialog>
</template>

<script>
import { formatDate } from '@/utils'

export default {
  name: 'SyncTaskDrawer',
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    task: {
      type: Object,
      default() {
        return {}
      },
    },
  },
  data() {
    return {
      innerVisible: false,
      activeName: 'progress',
      shardCurrentPage: 1,
      shardPageSize: 5,
    }
  },
  computed: {
    taskId() {
      return this.task?.taskID || this.task?.taskId || this.task?.id || '-'
    },
    ruleId() {
      return this.task?.ruleID || this.task?.ruleId || this.task?.request?.ruleId || this.task?.Request?.ruleId || '-'
    },
    typeText() {
      return this.task?.type || this.task?.opcode || '-'
    },
    statusText() {
      return this.task?.status || '-'
    },
    ownerText() {
      return this.task?.owner || '-'
    },
    shardIndex() {
      return this.task?.shardIdx ?? this.task?.shardIndex ?? '-'
    },
    shardTotal() {
      return this.task?.shardTotal ?? '-'
    },
    shardText() {
      if (this.shardIndex === '-' && this.shardTotal === '-') {
        return '-'
      }
      return `${this.shardIndex}/${this.shardTotal}`
    },
    errorText() {
      return this.task?.error || this.task?.lastError || '-'
    },
    taskText() {
      return JSON.stringify(this.task || {}, null, 2)
    },
    statusTagType() {
      const status = `${this.task?.status || ''}`.toLowerCase()
      if (status === 'running') return 'warning'
      if (status === 'succeeded') return 'success'
      if (status === 'failed' || status === 'cancelled' || status === 'cancelling') return 'danger'
      return 'info'
    },
    isActive() {
      const s = this.task?.status
      return s === 'running' || s === 'queued'
    },
    hasProgress() {
      const p = this.task?.progress
      return p && (p.filesTotal > 0 || p.bytesDone > 0)
    },
    filesPct() {
      const p = this.task?.progress
      if (!p || !p.filesTotal) return 0
      if (this.task?.status === 'succeeded') return 100
      const processed = (p.filesDone || 0) + (p.filesSkipped || 0)
      return Math.min(Math.round((processed / p.filesTotal) * 100), 100)
    },
    bytesPct() {
      const p = this.task?.progress
      if (!p || !p.bytesTotal) return 0
      return Math.min(Math.round(((p.bytesDone || 0) + (p.bytesSkipped || 0)) / p.bytesTotal * 100), 100)
    },
    progressBarStatus() {
      const s = this.task?.status
      if (s === 'succeeded') return 'success'
      if (s === 'failed') return 'exception'
      if (s === 'cancelled') return 'warning'
      return null
    },
    throughputText() {
      // For terminal tasks compute from wall-clock time (more reliable than stored value)
      if (this.progressBarStatus !== null) {
        const v = this.finalAvgBandwidthMBps
        if (v && v > 0) {
          if (v >= 1) return `${v.toFixed(1)} MB/s`
          return `${(v * 1024).toFixed(0)} KB/s`
        }
        return '-'
      }
      const v = this.task?.progress?.throughputMBps
      if (!v || v <= 0) return '-'
      if (v >= 1) return `${v.toFixed(1)} MB/s`
      return `${(v * 1024).toFixed(0)} KB/s`
    },
    currentBandwidthText() {
      // Hide current bandwidth for terminal tasks (task has ended)
      if (this.progressBarStatus !== null) return null
      const v = this.task?.progress?.currentBandwidthMBps
      if (!v || v <= 0) return null
      if (v >= 1) return `${v.toFixed(1)} MB/s`
      return `${(v * 1024).toFixed(0)} KB/s`
    },
    finalAvgBandwidthMBps() {
      const task = this.task
      if (!task?.startedAt || !task?.doneAt || !task?.progress?.bytesDone) return null
      const elapsed = (new Date(task.doneAt) - new Date(task.startedAt)) / 1000
      if (elapsed <= 0) return null
      return task.progress.bytesDone / 1024 / 1024 / elapsed
    },
    skippedSamples() {
      return this.task?.progress?.skippedSamples || []
    },
    shards() {
      return this.task?._shards || []
    },
    shardsPagedData() {
      const start = (this.shardCurrentPage - 1) * this.shardPageSize
      return this.shards.slice(start, start + this.shardPageSize)
    },
    hasDataFlow() {
      const rc = this.task?._ruleConfig
      return !!(rc?.src?.kind && rc?.dst?.kind)
    },
    srcConfig() {
      return this._epConfig(this.task?._ruleConfig?.src)
    },
    dstConfig() {
      return this._epConfig(this.task?._ruleConfig?.dst)
    },
    flowConnectorClass() {
      const s = (this.task?.status || '').toLowerCase()
      if (s === 'running') return 'dataflow-connector--running'
      if (s === 'succeeded') return 'dataflow-connector--success'
      if (s === 'failed') return 'dataflow-connector--failed'
      if (s === 'cancelled' || s === 'cancelling') return 'dataflow-connector--cancelled'
      return ''
    },
    arrowColor() {
      const s = (this.task?.status || '').toLowerCase()
      if (s === 'running') return '#3b82f6'
      if (s === 'succeeded') return '#16a34a'
      if (s === 'failed') return '#dc2626'
      if (s === 'cancelled' || s === 'cancelling') return '#d97706'
      return '#9ca3af'
    },
  },
  watch: {
    visible: {
      immediate: true,
      handler(val) {
        this.innerVisible = val
      },
    },
  },
  methods: {
    handleClose() {
      this.activeName = 'progress'
      this.shardCurrentPage = 1
      this.$emit('update:visible', false)
    },
    shardTagType(status) {
      const map = { queued: 'info', running: '', succeeded: 'success', failed: 'danger', cancelled: 'warning', cancelling: 'warning' }
      return map[status] ?? 'info'
    },
    _epConfig(ep) {
      if (!ep) return { kind: '?', name: '-', sub: '' }
      const kind = ep.kind || '?'
      let name = '-'
      let sub = ''
      if (kind === 's3' || kind === 'tos' || kind === 'bos') {
        name = ep.bucket || '-'
        sub = ep.prefix ? ep.prefix : ''
      } else if (kind === 'cfs') {
        name = ep.vol || '-'
        sub = ep.path || ''
      } else if (kind === 'local') {
        name = ep.path || '-'
      } else {
        name = ep.bucket || ep.vol || ep.path || '-'
      }
      return { kind, name, sub }
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
  },
}
</script>

<style lang="scss" scoped>
/* 数据路径可视化 */
.dataflow-row {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 20px 24px;
  background: linear-gradient(135deg, #f0f9ff 0%, #fafafa 50%, #f0fdf4 100%);
  border: 1px solid #e5e7eb;
  border-radius: 14px;
}

.dataflow-ep {
  flex: 1;
  min-width: 0;
  padding: 14px 16px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(0,0,0,.06);
}

.dataflow-ep__kind {
  display: inline-block;
  padding: 2px 8px;
  margin-bottom: 8px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: .5px;
  border-radius: 4px;
  color: #fff;
  background: #6b7280;

  &--s3, &--tos, &--bos { background: #f97316; }
  &--cfs { background: #2563eb; }
  &--local { background: #059669; }
}

.dataflow-ep__name {
  font-size: 15px;
  font-weight: 600;
  color: #111827;
  word-break: break-all;
  line-height: 1.4;
}

.dataflow-ep__sub {
  margin-top: 4px;
  font-size: 12px;
  color: #374151;
  font-family: monospace;
  word-break: break-all;

  &--muted {
    color: #9ca3af;
    font-style: italic;
    font-family: inherit;
  }
}

/* 连接器：流动动画轨道 + 箭头 */
.dataflow-connector {
  flex: 0 0 80px;
  display: flex;
  align-items: center;
  gap: 0;
  padding: 0 8px;
}

.dataflow-connector__track {
  flex: 1;
  height: 4px;
  border-radius: 2px;
  background: #e5e7eb;
  position: relative;
  overflow: hidden;
}

.dataflow-connector__fill {
  position: absolute;
  inset: 0;
  border-radius: 2px;
  background: #e5e7eb;
  transition: background 0.3s;
}

.dataflow-connector--running .dataflow-connector__fill { background: #bfdbfe; }
.dataflow-connector--success .dataflow-connector__fill { background: #bbf7d0; }
.dataflow-connector--failed .dataflow-connector__fill { background: #fecaca; }
.dataflow-connector--cancelled .dataflow-connector__fill { background: #fed7aa; }

.dataflow-connector__shine {
  position: absolute;
  top: 0;
  left: 0;
  height: 100%;
  width: 45%;
  background: linear-gradient(90deg, transparent 0%, rgba(59,130,246,.75) 50%, transparent 100%);
  animation: flow-shine 1.3s ease-in-out infinite;
  pointer-events: none;
}

@keyframes flow-shine {
  0%   { transform: translateX(-120%); }
  100% { transform: translateX(290%); }
}

.dataflow-connector__arrowhead {
  flex: 0 0 12px;
  display: block;
}

.dialog-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-right: 24px;
}

.dialog-title__label {
  font-size: 14px;
  color: #6b7280;
}

.dialog-title__value {
  margin-top: 6px;
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.dialog-body {
  padding-top: 4px;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 20px;

  &--2col {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.summary-card {
  padding: 16px;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #f6faf8 100%);
}

.summary-card__label {
  margin-bottom: 8px;
  font-size: 12px;
  color: #6b7280;
}

.summary-card__value {
  font-size: 16px;
  font-weight: 600;
  color: #111827;
  word-break: break-all;
}

.section {
  margin-bottom: 20px;
}

.section-title {
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 600;
  color: #111827;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.info-item {
  min-height: 82px;
  padding: 14px 16px;
  border-radius: 10px;
  background: #f8fafc;
}

.info-item__label {
  margin-bottom: 8px;
  font-size: 12px;
  color: #6b7280;
}

.info-item__value {
  font-size: 14px;
  line-height: 1.5;
  color: #111827;
  word-break: break-all;
}

.error-panel {
  padding: 14px 16px;
  line-height: 1.6;
  color: #b42318;
  background: #fef3f2;
  border: 1px solid #fecdca;
  border-radius: 10px;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 进度区域 */
.progress-section {
  padding: 4px 0;
}

/* 带宽摘要行 */
.bw-row {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.bw-card {
  flex: 1;
  padding: 12px 16px;
  border-radius: 10px;
  background: #eff6ff;
  border: 1px solid #bfdbfe;

  &--current {
    background: #eff6ff;
    border-color: #bfdbfe;
  }

  &--avg {
    background: #f8fafc;
    border-color: #e2e8f0;
  }
}

.bw-card__label {
  font-size: 11px;
  color: #6b7280;
  margin-bottom: 4px;
}

.bw-card__value {
  font-size: 20px;
  font-weight: 700;
  color: #1d4ed8;

  .bw-card--avg & {
    font-size: 18px;
    color: #374151;
    font-weight: 600;
  }
}

.progress-row__header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 6px;
}

.progress-row__label {
  font-size: 13px;
  font-weight: 500;
  color: #374151;
}

.progress-row__value {
  font-size: 13px;
  color: #6b7280;
}

.progress-row__skip {
  font-size: 12px;
  color: #e6a23c;
}

.progress-stats {
  display: flex;
  gap: 24px;
  margin-top: 20px;
  padding: 14px 16px;
  border-radius: 10px;
  background: #f8fafc;
}

.progress-stat {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.progress-stat__label {
  font-size: 12px;
  color: #6b7280;
}

.progress-stat__value {
  font-size: 16px;
  font-weight: 600;
  color: #111827;

  &--danger {
    color: #dc2626;
  }
}

.bar-terminal ::v-deep .el-progress-bar__inner {
  transition: none !important;
}

.empty-text {
  padding: 24px 0;
  font-size: 13px;
  color: #9ca3af;
  text-align: center;
}

/* Raw JSON 代码块：浅色背景，深色文字，可读 */
.code-block {  margin: 0;
  max-height: 360px;
  padding: 16px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.6;
  color: #1e293b;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 跳过文件列表 */
.skipped-note {
  margin-bottom: 12px;
  font-size: 13px;
  color: #6b7280;
}

.skipped-list {
  margin: 0;
  padding: 0;
  max-height: 320px;
  overflow-y: auto;
  list-style: none;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
}

.skipped-item {
  padding: 7px 14px;
  font-size: 12px;
  font-family: monospace;
  color: #374151;
  border-bottom: 1px solid #f1f5f9;
  word-break: break-all;

  &:last-child {
    border-bottom: none;
  }

  &:nth-child(odd) {
    background: #f8fafc;
  }
}

/* 分片详情表格 */
.shard-detail-table {
  margin-top: 4px;
  font-size: 12px;
}

.shard-owner-link {
  color: #409eff;
  cursor: pointer;
  word-break: break-all;

  &:hover {
    color: #337ecc;
  }
}

.muted {
  color: #c0c4cc;
}

.skip-num {
  color: #e6a23c;
}

.fail-num {
  color: #f56c6c;
  font-weight: 600;
}

.shard-pagination {
  display: flex;
  justify-content: flex-end;
  margin-top: 12px;
}

@media (max-width: 1200px) {
  .summary-grid,
  .info-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
