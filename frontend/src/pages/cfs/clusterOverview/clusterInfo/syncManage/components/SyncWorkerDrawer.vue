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
    <el-dialog
      :visible.sync="innerVisible"
      width="980px"
      top="5vh"
      append-to-body
      custom-class="sync-worker-dialog"
      @close="handleClose"
    >
      <div slot="title" class="dialog-title">
        <div>
          <div class="dialog-title__label">{{ $t('sync.workers') }}</div>
          <div class="dialog-title__value">{{ worker?.addr || '-' }}</div>
        </div>
        <el-tag size="small" :type="workerStateType">{{ workerStateText }}</el-tag>
      </div>
      <div class="dialog-body">
        <div class="toolbar">
          <el-button size="small" @click="reloadDiagnostics">{{ $t('button.refresh') }}</el-button>
          <el-button v-auth="'CFS_SYNCNODE_RELOAD'" size="small" @click="reloadConfig">{{ $t('sync.reload') }}</el-button>
          <el-button v-auth="'CFS_SYNCNODE_DRAIN'" size="small" type="warning" @click="drainNode">{{ $t('sync.drain') }}</el-button>
          <el-button v-auth="'CFS_SYNCNODE_RESTORE'" size="small" type="success" @click="restoreNode">{{ $t('sync.restore') }}</el-button>
          <el-button v-auth="'CFS_SYNCNODE_DECOMMISSION'" size="small" type="danger" @click="decommissionDialogVisible = true">{{ $t('common.offline') }}</el-button>
          <el-button size="small" @click="$emit('view-tasks', worker.addr)">{{ $t('sync.viewtasks') }}</el-button>
        </div>

        <div class="summary-grid">
          <div class="summary-card">
            <div class="summary-card__label">{{ $t('sync.runningtasks') }}</div>
            <div class="summary-card__value">{{ worker?.runningTasks ?? 0 }}</div>
          </div>
          <div class="summary-card">
            <div class="summary-card__label">{{ $t('sync.queuedtasks') }}</div>
            <div class="summary-card__value">{{ worker?.queuedTasks ?? 0 }}</div>
          </div>
          <div class="summary-card">
            <div class="summary-card__label">{{ $t('sync.loadscore') }}</div>
            <div class="summary-card__value">{{ formatNumber(worker?.loadScore) }}</div>
          </div>
          <div class="summary-card">
            <div class="summary-card__label">{{ $t('sync.throughput') }}</div>
            <div class="summary-card__value">{{ formatNumber(worker?.bandwidthMBps) }}</div>
          </div>
        </div>

        <el-tabs v-model="activeName">
          <el-tab-pane :label="$t('sync.basicinfo')" name="runtime">
            <div class="info-grid">
              <div class="info-item">
                <div class="info-item__label">addr</div>
                <div class="info-item__value">{{ worker?.addr || '-' }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.version') }}</div>
                <div class="info-item__value">{{ worker?.version || '-' }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.state') }}</div>
                <div class="info-item__value">{{ workerStateText }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.uptime') }}</div>
                <div class="info-item__value">{{ formatUptime(metricValue(worker?.uptimeSeconds, statInfo?.uptimeSeconds)) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">MB/s</div>
                <div class="info-item__value">{{ formatNumber(metricValue(worker?.bandwidthMBps)) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">limit MB/s</div>
                <div class="info-item__value">{{ formatNumber(metricValue(worker?.bandwidthMBpsLimit)) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.cpupercent') }}</div>
                <div class="info-item__value">{{ formatCPU(metricValue(worker?.cpuPercent), metricValue(worker?.cpuCores)) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.mempercent') }}</div>
                <div class="info-item__value">{{ formatMem(metricValue(worker?.memPercent), metricValue(worker?.memTotalMB)) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.loadscore') }}</div>
                <div class="info-item__value">{{ formatNumber(metricValue(worker?.loadScore)) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.concurrenttasks') }}</div>
                <div class="info-item__value">{{ metricValue(statInfo?.concurrentTasks) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.maxconcurrenttasks') }}</div>
                <div class="info-item__value">{{ metricValue(worker?.maxConcurrentTasks) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.taskfailurerate') }}</div>
                <div class="info-item__value">{{ formatRatioPercent(metricValue(worker?.lastTaskFailureRate)) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.boltdbhealthy') }}</div>
                <div class="info-item__value">{{ formatHealth(metricValue(worker?.boltDBHealthy, statInfo?.boltdbHealthy)) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">{{ $t('sync.reloadfailures') }}</div>
                <div class="info-item__value">{{ metricValue(worker?.reloadFailures, statInfo?.reloadFailuresTotal) }}</div>
              </div>
              <div class="info-item">
                <div class="info-item__label">isActive</div>
                <div class="info-item__value">{{ formatBoolean(metricValue(worker?.isActive)) }}</div>
              </div>
            </div>

            <div class="section">
              <div class="section-title">{{ $t('sync.currenttasks') }}</div>
              <u-page-table :data="tasks" :page-size="6" border>
                <el-table-column label="taskID" min-width="180">
                  <template slot-scope="scope">
                    <span>{{ getTaskId(scope.row) }}</span>
                  </template>
                </el-table-column>
                <el-table-column label="ruleID" min-width="140">
                  <template slot-scope="scope">
                    <span>{{ getRuleId(scope.row) }}</span>
                  </template>
                </el-table-column>
                <el-table-column :label="$t('sync.state')" min-width="100" prop="status"></el-table-column>
                <el-table-column :label="$t('sync.startedat')" min-width="160">
                  <template slot-scope="scope">
                    <span>{{ formatTime(scope.row.startedAt) }}</span>
                  </template>
                </el-table-column>
              </u-page-table>
            </div>
          </el-tab-pane>
          <el-tab-pane :label="$t('sync.diagnostics')" name="diagnostics">
            <el-alert :title="$t('sync.reloadtips')" type="info" :closable="false" class="mg-bt-s"></el-alert>
            <el-alert :title="$t('sync.scheduledrulestips')" type="warning" :closable="false" class="mg-bt-s"></el-alert>
            <div class="section">
              <div class="section-title">{{ $t('sync.version') }}</div>
              <pre class="code-block">{{ versionText }}</pre>
            </div>
            <div class="section">
              <div class="section-title">{{ $t('sync.stat') }}</div>
              <pre class="code-block">{{ statText }}</pre>
            </div>
          </el-tab-pane>
          <el-tab-pane :label="$t('sync.advanced')" name="advanced">
            <pre class="code-block">{{ advancedText }}</pre>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-dialog>

    <el-dialog
      :visible.sync="decommissionDialogVisible"
      :title="$t('common.offline')"
      width="420px"
      append-to-body
    >
      <el-radio-group v-model="decommissionMode">
        <el-radio label="smooth">{{ $t('sync.smoothoffline') }}</el-radio>
        <el-radio label="force">{{ $t('sync.forceoffline') }}</el-radio>
      </el-radio-group>
      <div class="confirm-text">{{ decommissionMode === 'force' ? $t('sync.confirmforcedecommission') : $t('sync.confirmsmoothdecommission') }}</div>
      <div class="task-summary">
        <div>{{ $t('sync.runningtasks') }}: {{ worker?.runningTasks ?? 0 }}</div>
        <div>{{ $t('sync.queuedtasks') }}: {{ worker?.queuedTasks ?? 0 }}</div>
        <div>{{ $t('sync.affectedtasks') }}: {{ tasks.length }}</div>
      </div>
      <div slot="footer">
        <el-button @click="decommissionDialogVisible = false">{{ $t('button.cancel') }}</el-button>
        <el-button type="danger" @click="decommissionNode">{{ $t('button.submit') }}</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { formatDate } from '@/utils'
import UPageTable from '@/pages/components/uPageTable'
import {
  decommissionSyncNode,
  drainSyncNode,
  getSyncNodeStat,
  getSyncNodeTasks,
  getSyncNodeVersion,
  reloadSyncNode,
  restoreSyncNode,
} from '@/api/cfs/cluster'

export default {
  name: 'SyncWorkerDrawer',
  components: {
    UPageTable,
  },
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    clusterName: {
      type: String,
      required: true,
    },
    worker: {
      type: Object,
      default() {
        return {}
      },
    },
  },
  data() {
    return {
      innerVisible: false,
      activeName: 'runtime',
      versionInfo: {},
      statInfo: {},
      tasks: [],
      decommissionDialogVisible: false,
      decommissionMode: 'smooth',
    }
  },
  computed: {
    versionText() {
      return JSON.stringify(this.versionInfo || {}, null, 2)
    },
    statText() {
      return JSON.stringify(this.statInfo || {}, null, 2)
    },
    advancedText() {
      return JSON.stringify({
        isActive: this.worker?.isActive,
        state: this.worker?.state,
        worker: this.worker,
      }, null, 2)
    },
    workerStateText() {
      if (this.worker?.state) {
        return this.worker.state
      }
      if (this.worker?.isActive === true) {
        return 'active'
      }
      if (this.worker?.isActive === false) {
        return 'inactive'
      }
      return '-'
    },
    workerStateType() {
      const state = `${this.workerStateText}`.toLowerCase()
      if (state === 'active') {
        return 'success'
      }
      if (state === 'draining') {
        return 'warning'
      }
      if (state === 'decommissioned' || state === 'inactive') {
        return 'info'
      }
      return 'danger'
    },
  },
  watch: {
    visible: {
      immediate: true,
      handler(val) {
        this.innerVisible = val
        if (val && this.worker?.addr) {
          this.reloadDiagnostics()
        }
      },
    },
    worker: {
      deep: true,
      handler(val) {
        if (this.innerVisible && val?.addr) {
          this.reloadDiagnostics()
        }
      },
    },
  },
  methods: {
    handleClose() {
      this.activeName = 'runtime'
      this.$emit('update:visible', false)
    },
    formatTime(value) {
      return value ? formatDate(value) : '-'
    },
    metricValue(...values) {
      const value = values.find(item => item !== undefined && item !== null && item !== '')
      return value === undefined ? '-' : value
    },
    formatNumber(value, digits = 2) {
      if (value === '-') {
        return '-'
      }
      const num = Number(value)
      if (!Number.isFinite(num)) {
        return value
      }
      if (Number.isInteger(num)) {
        return `${num}`
      }
      return num.toFixed(digits)
    },
    formatPercent(value) {
      if (value === '-') {
        return '-'
      }
      const num = Number(value)
      if (!Number.isFinite(num)) {
        return value
      }
      return `${num.toFixed(2)}%`
    },
    formatCPU(percent, cores) {
      const p = Number(percent)
      const c = Number(cores)
      const pStr = Number.isFinite(p) ? `${p.toFixed(1)}%` : '-'
      if (Number.isFinite(c) && c > 0) return `${pStr} (${c} cores)`
      return pStr
    },
    formatMem(percent, totalMB) {
      const p = Number(percent)
      const t = Number(totalMB)
      const pStr = Number.isFinite(p) ? `${p.toFixed(1)}%` : '-'
      if (Number.isFinite(t) && t > 0) {
        const totalGB = t / 1024
        const usedGB = p * totalGB / 100
        return `${pStr} (${usedGB.toFixed(1)}/${totalGB.toFixed(1)} GB)`
      }
      return pStr
    },
    formatRatioPercent(value) {
      if (value === '-') {
        return '-'
      }
      const num = Number(value)
      if (!Number.isFinite(num)) {
        return value
      }
      return `${(num * 100).toFixed(2)}%`
    },
    formatUptime(value) {
      if (value === '-') {
        return '-'
      }
      const total = Math.max(0, Math.floor(Number(value)))
      if (!Number.isFinite(total)) {
        return value
      }
      const days = Math.floor(total / 86400)
      const hours = Math.floor((total % 86400) / 3600)
      const minutes = Math.floor((total % 3600) / 60)
      const seconds = total % 60
      if (days > 0) {
        return `${days}d ${hours}h`
      }
      if (hours > 0) {
        return `${hours}h ${minutes}m`
      }
      if (minutes > 0) {
        return `${minutes}m ${seconds}s`
      }
      return `${seconds}s`
    },
    formatHealth(value) {
      if (value === true) {
        return this.$t('common.health')
      }
      if (value === false) {
        return this.$t('common.unhealthy')
      }
      return '-'
    },
    formatBoolean(value) {
      if (value === true) {
        return 'true'
      }
      if (value === false) {
        return 'false'
      }
      return '-'
    },
    getTaskId(row) {
      return row?.taskID || row?.taskId || row?.id || '-'
    },
    getRuleId(row) {
      return row?.ruleID || row?.ruleId || row?.request?.ruleId || row?.Request?.ruleId || '-'
    },
    async reloadDiagnostics() {
      if (!this.worker?.addr) {
        return
      }
      const [versionRes, statRes, tasksRes] = await Promise.all([
        getSyncNodeVersion({
          cluster_name: this.clusterName,
          addr: this.worker.addr,
        }),
        getSyncNodeStat({
          cluster_name: this.clusterName,
          addr: this.worker.addr,
        }),
        getSyncNodeTasks({
          cluster_name: this.clusterName,
          addr: this.worker.addr,
        }),
      ])
      this.versionInfo = versionRes.data || {}
      this.statInfo = statRes.data || {}
      this.tasks = tasksRes.data || []
    },
    async reloadConfig() {
      await reloadSyncNode({
        cluster_name: this.clusterName,
        addr: this.worker.addr,
      })
      this.$message.success(this.$t('sync.reload') + this.$t('common.xxsuc'))
      this.reloadDiagnostics()
    },
    async drainNode() {
      await this.$confirm(this.$t('sync.confirmdrain'), this.$t('common.notice'), {
        confirmButtonText: this.$t('common.yes'),
        cancelButtonText: this.$t('common.no'),
      })
      await drainSyncNode({
        cluster_name: this.clusterName,
        addr: this.worker.addr,
      })
      this.$message.success(this.$t('sync.drain') + this.$t('common.xxsuc'))
      this.$emit('refresh')
      this.reloadDiagnostics()
    },
    async restoreNode() {
      await this.$confirm(this.$t('sync.confirmrestore'), this.$t('common.notice'), {
        confirmButtonText: this.$t('common.yes'),
        cancelButtonText: this.$t('common.no'),
      })
      await restoreSyncNode({
        cluster_name: this.clusterName,
        addr: this.worker.addr,
      })
      this.$message.success(this.$t('sync.restore') + this.$t('common.xxsuc'))
      this.$emit('refresh')
      this.reloadDiagnostics()
    },
    async decommissionNode() {
      await decommissionSyncNode({
        cluster_name: this.clusterName,
        addr: this.worker.addr,
        force: this.decommissionMode === 'force',
      })
      this.$message.success(this.$t('common.offline') + this.$t('common.xxsuc'))
      this.decommissionDialogVisible = false
      this.$emit('refresh')
      this.reloadDiagnostics()
    },
  },
}
</script>

<style lang="scss" scoped>
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

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 20px;
}

.summary-card {
  padding: 16px;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  background: linear-gradient(180deg, #ffffff 0%, #f7fafc 100%);
}

.summary-card__label {
  margin-bottom: 8px;
  font-size: 12px;
  color: #6b7280;
}

.summary-card__value {
  font-size: 18px;
  font-weight: 600;
  color: #111827;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 20px;
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

.section {
  margin-top: 20px;
}

.section-title {
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 600;
  color: #111827;
}

.code-block {
  margin: 0;
  max-height: 280px;
  padding: 16px;
  overflow: auto;
  line-height: 1.6;
  color: #1f2937;
  background: #0f172a;
  border-radius: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}

.confirm-text {
  margin-top: 16px;
  line-height: 1.6;
  color: #606266;
}

.task-summary {
  margin-top: 12px;
  padding: 12px;
  line-height: 1.8;
  background: #f8fafc;
  border-radius: 10px;
}

@media (max-width: 1200px) {
  .summary-grid,
  .info-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
