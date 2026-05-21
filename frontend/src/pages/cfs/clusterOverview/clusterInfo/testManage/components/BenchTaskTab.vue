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
        <el-select v-model="localFilters.status" clearable size="small" :placeholder="$t('bench.taskstatus')">
          <el-option v-for="item in statuses" :key="item" :label="item" :value="item"></el-option>
        </el-select>
        <el-input v-model.trim="localFilters.ruleID" size="small" clearable :placeholder="$t('bench.ruleid')"></el-input>
        <el-button size="small" type="primary" @click="loadData">{{ $t('button.search') }}</el-button>
        <el-button size="small" @click="resetFilters">{{ $t('button.reset') }}</el-button>
      </div>
      <el-button size="small" :loading="loading" icon="el-icon-refresh" @click="loadData">{{ $t('button.refresh') }}</el-button>
    </div>

    <u-page-table :data="dataList" :page-size="page.per_page" border>
      <el-table-column :label="$t('bench.taskid')" min-width="220">
        <template slot-scope="scope">
          <span>{{ scope.row.taskID }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('bench.ruleid')" min-width="160">
        <template slot-scope="scope">
          <span>{{ scope.row.ruleID || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('bench.taskstatus')" min-width="110">
        <template slot-scope="scope">
          <el-tag :type="statusTagType(scope.row.status)" size="mini" disable-transitions>{{ scope.row.status || '-' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.createtime')" min-width="160">
        <template slot-scope="scope">
          <span>{{ formatTime(scope.row.createdAt) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.action')" min-width="180">
        <template slot-scope="scope">
          <el-button type="text" @click="viewDetail(scope.row)">{{ $t('bench.viewdetail') }}</el-button>
          <el-button
            v-if="scope.row.status === 'running'"
            v-auth="'CFS_BENCHTASK_CANCEL'"
            type="text"
            style="color: #ed4014"
            @click="cancelTask(scope.row)"
          >{{ $t('bench.cancel') }}</el-button>
          <el-button
            v-if="scope.row.status !== 'running'"
            v-auth="'CFS_BENCHTASK_RETRY'"
            type="text"
            @click="retryTask(scope.row)"
          >{{ $t('bench.retry') }}</el-button>
          <el-button
            v-if="scope.row.status !== 'running'"
            v-auth="'CFS_BENCHTASK_DELETE'"
            type="text"
            style="color: #ed4014"
            @click="deleteTask(scope.row)"
          >{{ $t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </u-page-table>

    <BenchTaskDrawer
      :visible.sync="drawerVisible"
      :task="selectedTask"
    />
  </div>
</template>

<script>
import { formatDate } from '@/utils'
import UPageTable from '@/pages/components/uPageTable'
import { getBenchTaskList, cancelBenchTask, retryBenchTask, deleteBenchTask } from '@/api/cfs/cluster'
import BenchTaskDrawer from './BenchTaskDrawer.vue'

export default {
  name: 'BenchTaskTab',
  components: {
    UPageTable,
    BenchTaskDrawer,
  },
  props: {
    clusterName: {
      type: String,
      required: true,
    },
    filters: {
      type: Object,
      default: () => ({ status: '', ruleID: '' }),
    },
  },
  data() {
    return {
      dataList: [],
      loading: false,
      drawerVisible: false,
      selectedTask: null,
      localFilters: {
        status: '',
        ruleID: '',
      },
      statuses: ['running', 'succeeded', 'failed', 'cancelled'],
      page: {
        per_page: 10,
      },
    }
  },
  watch: {
    filters: {
      immediate: true,
      handler(val) {
        this.localFilters = { status: val.status || '', ruleID: val.ruleID || '' }
        this.loadData()
      },
    },
  },
  methods: {
    async loadData() {
      this.loading = true
      try {
        const { data } = await getBenchTaskList({
          cluster_name: this.clusterName,
          status: this.localFilters.status || undefined,
          rule_id: this.localFilters.ruleID || undefined,
        })
        // Fan-out 任务在 ledger 里是 1 父 + N 子 (taskID="<parent>/<N>" /
        // parentTaskID 非空)；用户只关心父任务的整体状态，shard 子任务的
        // 细节通过父任务的 详情抽屉里"Shards"区域展示。这里过滤掉子记录
        // 避免列表里出现 N+1 行让人困惑。
        const raw = data || []
        this.dataList = raw.filter(t => !t.parentTaskID && !(t.taskID && t.taskID.includes('/')))
      } finally {
        this.loading = false
      }
    },
    resetFilters() {
      this.localFilters = { status: '', ruleID: '' }
      this.loadData()
    },
    formatTime(value) {
      if (!value) return '-'
      const d = new Date(value)
      if (isNaN(d.getTime()) || d.getFullYear() < 2000) return '-'
      return formatDate(value)
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
    viewDetail(row) {
      this.selectedTask = row
      this.drawerVisible = true
    },
    async cancelTask(row) {
      await this.$confirm(
        `${this.$t('bench.cancel')} ${row.taskID} ?`,
        this.$t('common.notice'),
        {
          confirmButtonText: this.$t('common.yes'),
          cancelButtonText: this.$t('common.no'),
        }
      )
      await cancelBenchTask({ cluster_name: this.clusterName, id: row.taskID })
      this.$message.success(this.$t('bench.cancel') + this.$t('common.xxsuc'))
      this.loadData()
    },
    async retryTask(row) {
      await retryBenchTask({ cluster_name: this.clusterName, id: row.taskID })
      this.$message.success(this.$t('bench.retry') + this.$t('common.xxsuc'))
      this.loadData()
    },
    async deleteTask(row) {
      try {
        await this.$confirm(`确定删除任务 ${row.taskID} 吗？该操作不可恢复。`, '提示', { type: 'warning' })
      } catch (_) { return }
      try {
        await deleteBenchTask({ cluster_name: this.clusterName, id: row.taskID })
        this.$message.success('已删除')
        this.loadData()
      } catch (e) {
        this.$message.error('删除失败：' + (e.message || e))
      }
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
</style>
