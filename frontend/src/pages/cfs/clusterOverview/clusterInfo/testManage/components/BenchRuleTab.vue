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
        <!-- no filter on rules tab for now -->
      </div>
      <el-button
        v-auth="'CFS_BENCHRULE_CREATE'"
        type="primary"
        size="small"
        @click="openCreate"
      >{{ $t('bench.createbenchrule') }}</el-button>
    </div>
    <u-page-table :data="dataList" :page-size="page.per_page" border>
      <el-table-column :label="$t('bench.ruleid')" min-width="160">
        <template slot-scope="scope">
          <span>{{ scope.row.id }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('bench.rulename')" min-width="160">
        <template slot-scope="scope">
          <span>{{ scope.row.name || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('bench.storagetype')" min-width="100">
        <template slot-scope="scope">
          <span>{{ scope.row.storageType || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('bench.parallelism')" min-width="100">
        <template slot-scope="scope">
          <span>{{ scope.row.parallelism || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.createtime')" min-width="160">
        <template slot-scope="scope">
          <span>{{ formatTime(scope.row.createdAt) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="最近运行" min-width="200">
        <template slot-scope="scope">
          <span v-if="!runSummary[scope.row.id]" style="color:#909399;">未运行</span>
          <span v-else>
            <el-tag :type="statusTagType(runSummary[scope.row.id].lastStatus)" size="mini" disable-transitions>
              {{ runSummary[scope.row.id].lastStatus }}
            </el-tag>
            <span style="margin-left:6px;color:#606266;font-size:12px;">
              {{ formatTime(runSummary[scope.row.id].lastAt) }}
            </span>
            <span style="margin-left:6px;color:#909399;font-size:12px;">
              · 共 {{ runSummary[scope.row.id].total }} 次
            </span>
          </span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.action')" min-width="260">
        <template slot-scope="scope">
          <MoreOPerate :count="4" :i18n="$i18n">
            <el-button v-auth="'CFS_BENCHRULE_TRIGGER'" type="text" @click="triggerRule(scope.row)">{{ $t('bench.trigger') }}</el-button>
            <el-button v-auth="'CFS_BENCHTASK_LIST'" type="text" @click="openTrends(scope.row)">查看趋势</el-button>
            <el-button v-auth="'CFS_BENCHRULE_UPDATE'" type="text" @click="openEdit(scope.row)">{{ $t('common.edit') }}</el-button>
            <el-button v-auth="'CFS_BENCHRULE_DELETE'" type="text" style="color: #ed4014" @click="deleteRule(scope.row)">{{ $t('bench.deletebenchrule') }}</el-button>
          </MoreOPerate>
        </template>
      </el-table-column>
    </u-page-table>
    <BenchRuleCreateDialog
      :visible.sync="createDialogVisible"
      :cluster-name="clusterName"
      :mode="editRow ? 'edit' : 'create'"
      :row="editRow"
      @confirm="handleCreateConfirm"
      @close="editRow = null"
    />
    <BenchTrendsDrawer
      :visible.sync="trendsDrawerVisible"
      :cluster-name="clusterName"
      :rule="trendsRule"
    />
  </div>
</template>

<script>
import { formatDate } from '@/utils'
import MoreOPerate from '@/pages/components/moreOPerate'
import UPageTable from '@/pages/components/uPageTable'
import {
  getBenchRuleList,
  getBenchTaskList,
  deleteBenchRule,
  triggerBenchRule,
} from '@/api/cfs/cluster'
import BenchRuleCreateDialog from './BenchRuleCreateDialog.vue'
import BenchTrendsDrawer from './BenchTrendsDrawer.vue'

export default {
  name: 'BenchRuleTab',
  components: {
    MoreOPerate,
    UPageTable,
    BenchRuleCreateDialog,
    BenchTrendsDrawer,
  },
  props: {
    clusterName: {
      type: String,
      required: true,
    },
  },
  data() {
    return {
      dataList: [],
      runSummary: {}, // ruleID -> { lastStatus, lastAt, total }
      createDialogVisible: false,
      editRow: null,
      trendsDrawerVisible: false,
      trendsRule: null,
      page: {
        per_page: 10,
      },
    }
  },
  mounted() {
    this.loadData()
  },
  methods: {
    async loadData() {
      const { data } = await getBenchRuleList({ cluster_name: this.clusterName })
      this.dataList = data || []
      this.loadRunSummary()
    },
    // 拉一次所有 task list（ruleID 不过滤），客户端按 ruleID 分组取最近一条 +
    // 总数。master 端 list 按时间倒序，取索引 0 即最新。fan-out 父任务的
    // taskID 没有 '/'，shard 子任务 taskID 是 '<parent>/<N>' — 这里只统计
    // 父/单 shard 任务，跳过 shard child。
    async loadRunSummary() {
      try {
        const { data } = await getBenchTaskList({ cluster_name: this.clusterName })
        const tasks = Array.isArray(data) ? data : (data?.data || [])
        const summary = {}
        for (const t of tasks) {
          if (!t.ruleID) continue
          if (t.parentTaskID) continue // skip fan-out child shards
          if (t.taskID && t.taskID.includes('/')) continue
          const s = summary[t.ruleID] = summary[t.ruleID] || { lastStatus: '', lastAt: 0, total: 0 }
          s.total += 1
          const at = t.updatedAt || t.createdAt || 0
          if (at > s.lastAt) {
            s.lastAt = at
            s.lastStatus = t.status || ''
          }
        }
        this.runSummary = summary
      } catch (_) {
        // 静默失败，summary 显示"未运行"，不阻塞主列表
      }
    },
    statusTagType(s) {
      switch (s) {
        case 'succeeded': return 'success'
        case 'failed': return 'danger'
        case 'cancelled': return 'info'
        case 'running': return 'warning'
        default: return ''
      }
    },
    formatTime(value) {
      if (!value) return '-'
      const d = new Date(value)
      if (isNaN(d.getTime()) || d.getFullYear() < 2000) return '-'
      return formatDate(value)
    },
    openCreate() {
      this.editRow = null
      this.createDialogVisible = true
    },
    openEdit(row) {
      this.editRow = row
      this.createDialogVisible = true
    },
    openTrends(row) {
      this.trendsRule = row
      this.trendsDrawerVisible = true
    },
    async handleCreateConfirm() {
      this.createDialogVisible = false
      this.editRow = null
      this.loadData()
    },
    async triggerRule(row) {
      await triggerBenchRule({
        cluster_name: this.clusterName,
        id: row.id,
      })
      this.$message.success(this.$t('bench.trigger') + this.$t('common.xxsuc'))
      this.$emit('view-tasks', row.id)
      this.loadData()
    },
    async deleteRule(row) {
      await this.$confirm(
        `${this.$t('bench.confirmdeletebenchrule')} (${row.id})`,
        this.$t('common.notice'),
        {
          confirmButtonText: this.$t('common.yes'),
          cancelButtonText: this.$t('common.no'),
        }
      )
      await deleteBenchRule({ cluster_name: this.clusterName, id: row.id })
      this.$message.success(this.$t('bench.deletebenchrule') + this.$t('common.xxsuc'))
      this.loadData()
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
