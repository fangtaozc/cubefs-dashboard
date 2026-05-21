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
  <el-card class="container">
    <div class="toolbar">
      <div class="filters">
        <el-input
          v-model.trim="searchData.addr"
          clearable
          size="small"
          :placeholder="$t('resource.inputnodeip')"
        ></el-input>
        <el-select
          v-model="searchData.state"
          clearable
          size="small"
          :placeholder="$t('sync.state')"
        >
          <el-option
            v-for="item in stateOptions"
            :key="item"
            :label="item"
            :value="item"
          ></el-option>
        </el-select>
        <el-button type="primary" size="small" @click="loadData">{{ $t('button.search') }}</el-button>
        <el-button size="small" @click="resetFilters">{{ $t('button.reset') }}</el-button>
      </div>
      <el-button size="small" @click="loadData">{{ $t('button.refresh') }}</el-button>
    </div>

    <u-page-table :data="filteredList" :page-size="page.per_page" border>
      <el-table-column :label="$t('resource.nodeaddr')" min-width="200">
        <template slot-scope="scope">
          <a @click="openWorker(scope.row)">{{ scope.row.addr }}</a>
        </template>
      </el-table-column>
      <el-table-column :label="$t('sync.state')" min-width="120">
        <template slot-scope="scope">
          <el-tag size="mini" :type="workerStateType(scope.row)">{{ workerState(scope.row) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('sync.version')" min-width="140" prop="version"></el-table-column>
      <el-table-column :label="$t('sync.runningtasks')" min-width="110" prop="runningTasks"></el-table-column>
      <el-table-column :label="$t('sync.queuedtasks')" min-width="110" prop="queuedTasks"></el-table-column>
      <el-table-column :label="$t('sync.cpupercent')" min-width="150">
        <template slot-scope="scope">
          <span>{{ formatCPU(scope.row.cpuPercent, scope.row.cpuCores) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('sync.mempercent')" min-width="180">
        <template slot-scope="scope">
          <span>{{ formatMem(scope.row.memPercent, scope.row.memTotalMB) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('sync.loadscore')" min-width="100">
        <template slot-scope="scope">
          <span>{{ formatNumber(scope.row.loadScore) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('sync.throughput')" min-width="120">
        <template slot-scope="scope">
          <span>{{ formatNumber(scope.row.bandwidthMBps) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.action')" min-width="180">
        <template slot-scope="scope">
          <MoreOPerate :count="4" :i18n="$i18n">
            <el-button type="text" @click="openWorker(scope.row)">{{ $t('common.detail') }}</el-button>
            <el-button type="text" @click="viewTasks(scope.row.addr)">{{ $t('sync.viewtasks') }}</el-button>
          </MoreOPerate>
        </template>
      </el-table-column>
    </u-page-table>

    <SyncWorkerDrawer
      :visible.sync="workerDrawerVisible"
      :cluster-name="clusterName"
      :worker="currentWorker"
      @refresh="loadData"
      @view-tasks="viewTasks"
    />
  </el-card>
</template>

<script>
import mixin from '@/pages/cfs/clusterOverview/mixin'
import MoreOPerate from '@/pages/components/moreOPerate'
import UPageTable from '@/pages/components/uPageTable'
import { getSyncNodeList } from '@/api/cfs/cluster'
import SyncWorkerDrawer from '../../syncManage/components/SyncWorkerDrawer.vue'

export default {
  name: 'SyncNode',
  components: {
    MoreOPerate,
    SyncWorkerDrawer,
    UPageTable,
  },
  mixins: [mixin],
  props: {
    info: {
      type: Object,
      default() {
        return {}
      },
    },
    focusAddr: {
      type: String,
      default: '',
    },
    openDetail: {
      type: Boolean,
      default: false,
    },
  },
  data() {
    return {
      dataList: [],
      searchData: {
        addr: '',
        state: '',
      },
      page: {
        per_page: 10,
      },
      workerDrawerVisible: false,
      currentWorker: {},
      pendingAutoOpen: false,
    }
  },
  computed: {
    filteredList() {
      return (this.dataList || []).filter((item) => {
        const matchAddr = !this.searchData.addr || `${item.addr || ''}`.includes(this.searchData.addr)
        const matchState = !this.searchData.state || this.workerState(item) === this.searchData.state
        return matchAddr && matchState
      })
    },
    stateOptions() {
      return Array.from(new Set((this.dataList || []).map(item => this.workerState(item)).filter(Boolean)))
    },
  },
  watch: {
    filteredList: {
      deep: true,
      immediate: true,
      handler(list) {
        this.emitInfo(list)
      },
    },
    workerDrawerVisible(val, oldVal) {
      if (!val && oldVal) {
        this.handleWorkerDialogClose()
      }
    },
    openDetail: {
      immediate: true,
      handler(val) {
        this.pendingAutoOpen = val
      },
    },
    focusAddr() {
      if (this.openDetail) {
        this.pendingAutoOpen = true
        this.tryAutoOpen()
      }
    },
  },
  created() {
    this.loadData()
  },
  methods: {
    workerState(row) {
      if (row?.state) {
        return row.state
      }
      if (row?.isActive === true) {
        return 'active'
      }
      if (row?.isActive === false) {
        return 'inactive'
      }
      return '-'
    },
    workerStateType(row) {
      const state = `${this.workerState(row)}`.toLowerCase()
      if (state === 'active') {
        return 'success'
      }
      if (state === 'draining') {
        return 'warning'
      }
      if (state === 'inactive' || state === 'decommissioned') {
        return 'info'
      }
      return 'info'
    },
    emitInfo(list) {
      const safeList = list || []
      const total = safeList.length
      const runningTasks = safeList.reduce((sum, item) => sum + Number(item.runningTasks || 0), 0)
      const queuedTasks = safeList.reduce((sum, item) => sum + Number(item.queuedTasks || 0), 0)
      const loadScoreTotal = safeList.reduce((sum, item) => sum + Number(item.loadScore || 0), 0)
      this.$emit('update:info', {
        node: total,
        runningTasks,
        queuedTasks,
        avgLoadScore: total ? (loadScoreTotal / total).toFixed(2) : '0.00',
      })
    },
    async loadData() {
      const { data } = await getSyncNodeList({
        cluster_name: this.clusterName,
      })
      this.dataList = data || []
      this.tryAutoOpen()
    },
    resetFilters() {
      this.searchData = {
        addr: '',
        state: '',
      }
    },
    openWorker(row) {
      this.currentWorker = { ...row }
      this.workerDrawerVisible = true
    },
    tryAutoOpen() {
      if (!this.pendingAutoOpen || !this.focusAddr) {
        return
      }
      const target = (this.dataList || []).find(item => item.addr === this.focusAddr)
      this.pendingAutoOpen = false
      if (target) {
        this.openWorker(target)
      }
    },
    handleWorkerDialogClose() {
      if (!this.$route.query.syncNodeAddr && !this.$route.query.syncNodeDialog) {
        return
      }
      const query = { ...this.$route.query }
      delete query.syncNodeAddr
      delete query.syncNodeDialog
      this.$router.replace({
        name: 'resourceManage',
        query,
      }).catch(() => {})
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
    viewTasks(owner) {
      this.$router.push({
        name: 'syncManage',
        query: {
          syncTab: 'tasks',
          owner,
        },
      })
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

.filters ::v-deep .el-input,
.filters ::v-deep .el-select {
  width: 180px;
}
</style>
