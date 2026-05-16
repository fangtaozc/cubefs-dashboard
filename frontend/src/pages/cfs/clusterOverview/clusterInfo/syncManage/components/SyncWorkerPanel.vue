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
    size="1100px"
    append-to-body
    @close="handleClose"
  >
    <div slot="title">{{ $t('sync.entermaintenance') }}</div>
    <div class="drawer-body">
      <div class="toolbar">
        <el-button size="small" @click="loadData">{{ $t('button.refresh') }}</el-button>
      </div>
      <div v-if="dataList.length" class="summary-bar">
        <span>{{ dataList.length }} 个执行器</span>
        <span class="summary-sep">·</span>
        <span>运行中任务 {{ totalRunningTasks }}</span>
        <span class="summary-sep">·</span>
        <span>CPU 均值 {{ avgCPU }}</span>
        <span class="summary-sep">·</span>
        <span>内存均值 {{ avgMem }}</span>
      </div>
      <el-table :data="dataList" border>
        <el-table-column label="addr" min-width="180" prop="addr"></el-table-column>
        <el-table-column :label="$t('sync.version')" min-width="120">
          <template slot-scope="scope">
            <span>{{ formatVersion(scope.row.version) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('sync.runningtasks')" min-width="110" prop="runningTasks"></el-table-column>
        <el-table-column :label="$t('sync.queuedtasks')" min-width="110" prop="queuedTasks"></el-table-column>
        <el-table-column label="MB/s" min-width="90">
          <template slot-scope="scope">
            <span>{{ formatNumber(scope.row.bandwidthMBps) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="limit MB/s" min-width="100" prop="bandwidthMBpsLimit"></el-table-column>
        <el-table-column :label="$t('sync.cpupercent')" min-width="130">
          <template slot-scope="scope">
            <span>{{ formatCPU(scope.row.cpuPercent, scope.row.cpuCores) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('sync.mempercent')" min-width="160">
          <template slot-scope="scope">
            <span>{{ formatMem(scope.row.memPercent, scope.row.memTotalMB) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('sync.loadscore')" min-width="90">
          <template slot-scope="scope">
            <span>{{ formatNumber(scope.row.loadScore) }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="$t('common.action')" min-width="160">
          <template slot-scope="scope">
            <MoreOPerate :count="3" :i18n="$i18n">
              <el-button type="text" @click="openWorker(scope.row)">{{ $t('sync.diagnostics') }}</el-button>
              <el-button type="text" @click="$emit('view-tasks', scope.row.addr)">{{ $t('sync.viewtasks') }}</el-button>
            </MoreOPerate>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <SyncWorkerDrawer
      :visible.sync="workerDrawerVisible"
      :cluster-name="clusterName"
      :worker="currentWorker"
      @refresh="loadData"
      @view-tasks="handleViewTasks"
    />
  </el-drawer>
</template>

<script>
import MoreOPerate from '@/pages/components/moreOPerate'
import { getSyncNodeList } from '@/api/cfs/cluster'
import SyncWorkerDrawer from './SyncWorkerDrawer.vue'

export default {
  name: 'SyncWorkerPanel',
  components: {
    MoreOPerate,
    SyncWorkerDrawer,
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
    focusOwner: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      innerVisible: false,
      dataList: [],
      workerDrawerVisible: false,
      currentWorker: {},
    }
  },
  computed: {
    totalRunningTasks() {
      return this.dataList.reduce((sum, w) => sum + (Number(w.runningTasks) || 0), 0)
    },
    avgCPU() {
      const valid = this.dataList.filter(w => Number.isFinite(Number(w.cpuPercent)))
      if (!valid.length) return '-'
      const avg = valid.reduce((s, w) => s + Number(w.cpuPercent), 0) / valid.length
      return `${avg.toFixed(1)}%`
    },
    avgMem() {
      const valid = this.dataList.filter(w => Number.isFinite(Number(w.memPercent)))
      if (!valid.length) return '-'
      const avg = valid.reduce((s, w) => s + Number(w.memPercent), 0) / valid.length
      return `${avg.toFixed(1)}%`
    },
  },
  watch: {
    visible: {
      immediate: true,
      handler(val) {
        this.innerVisible = val
        if (val) {
          this.loadData()
        }
      },
    },
    focusOwner(val) {
      if (!val) {
        return
      }
      if (this.innerVisible) {
        this.focusWorker(val)
      }
    },
  },
  methods: {
    handleClose() {
      this.$emit('update:visible', false)
    },
    formatVersion(version) {
      if (!version) return '-'
      const match = String(version).match(/v\d+\.\d+\.\d+[\w.-]*/)
      return match ? match[0] : String(version)
    },
    formatNumber(value, digits = 2) {
      if (value == null || value === '') return '-'
      const num = Number(value)
      if (!Number.isFinite(num)) return '-'
      if (Number.isInteger(num)) return `${num}`
      return num.toFixed(digits)
    },
    formatPercent(value) {
      if (value == null || value === '') return '-'
      const num = Number(value)
      if (!Number.isFinite(num)) return '-'
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
    async loadData() {
      const { data } = await getSyncNodeList({
        cluster_name: this.clusterName,
      })
      this.dataList = data || []
      if (this.focusOwner) {
        this.focusWorker(this.focusOwner)
      }
    },
    focusWorker(owner) {
      const worker = (this.dataList || []).find(item => item.addr === owner)
      if (!worker) {
        return
      }
      this.openWorker(worker)
    },
    openWorker(row) {
      this.currentWorker = { ...row }
      this.workerDrawerVisible = true
    },
    handleViewTasks(owner) {
      this.$emit('view-tasks', owner)
    },
  },
}
</script>

<style lang="scss" scoped>
.drawer-body {
  padding: 0 20px 20px;
}

.toolbar {
  margin-bottom: 12px;
}

.summary-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  margin-bottom: 12px;
  padding: 6px 10px;
  font-size: 12px;
  color: #6b7280;
  background: #f8fafc;
  border-radius: 6px;
}

.summary-sep {
  color: #d1d5db;
}
</style>
