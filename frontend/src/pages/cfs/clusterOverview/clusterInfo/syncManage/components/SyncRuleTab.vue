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
        <el-select v-model="filters.state" clearable size="small" :placeholder="$t('sync.state')">
          <el-option v-for="item in states" :key="item" :label="item" :value="item"></el-option>
        </el-select>
        <el-button size="small" type="primary" @click="loadData">{{ $t('button.search') }}</el-button>
        <el-button size="small" @click="resetFilters">{{ $t('button.reset') }}</el-button>
      </div>
      <el-button
        v-auth="'CFS_SYNCRULE_CREATE'"
        type="primary"
        size="small"
        @click="openCreate"
      >{{ $t('common.create') }}{{ $t('sync.rules') }}</el-button>
    </div>
    <u-page-table :data="dataList" :page-size="page.per_page" border>
      <el-table-column :label="$t('sync.ruleid')" min-width="180">
        <template slot-scope="scope">
          <span>{{ getConfig(scope.row, 'id') }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.type')" min-width="100">
        <template slot-scope="scope">
          <span>{{ getConfig(scope.row, 'type') }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('sync.state')" prop="state" min-width="100"></el-table-column>
      <el-table-column :label="$t('sync.schedule')" min-width="160">
        <template slot-scope="scope">
          <span>{{ getConfig(scope.row, 'schedule') || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('sync.shardstrategy')" min-width="120">
        <template slot-scope="scope">
          <span>{{ getConfig(scope.row, 'shardingStrategy') || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('sync.parallelism')" min-width="100">
        <template slot-scope="scope">
          <span>{{ getConfig(scope.row, 'parallelism') || '-' }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('sync.lastRunStatus')" prop="lastRunStatus" min-width="120"></el-table-column>
      <el-table-column :label="$t('sync.lastRunAt')" min-width="160">
        <template slot-scope="scope">
          <span>{{ formatTime(scope.row.lastRunAt) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('sync.updatedat')" min-width="160">
        <template slot-scope="scope">
          <span>{{ formatTime(scope.row.updatedAt) }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="$t('common.action')" min-width="300">
        <template slot-scope="scope">
          <MoreOPerate :count="4" :i18n="$i18n">
            <el-button type="text" @click="emitViewTasks(getConfig(scope.row, 'id'))">{{ $t('sync.viewtasks') }}</el-button>
            <el-button v-auth="'CFS_SYNCRULE_TRIGGER'" type="text" @click="triggerRule(scope.row)">{{ $t('sync.trigger') }}</el-button>
            <el-button v-auth="'CFS_SYNCRULE_UPDATE'" type="text" @click="openEdit(scope.row)">{{ $t('common.edit') }}</el-button>
            <el-button
              v-if="scope.row.state !== 'paused'"
              v-auth="'CFS_SYNCRULE_PAUSE'"
              type="text"
              @click="pauseRule(scope.row)"
            >{{ $t('common.off') }}</el-button>
            <el-button
              v-else
              v-auth="'CFS_SYNCRULE_RESUME'"
              type="text"
              @click="resumeRule(scope.row)"
            >{{ $t('common.on') }}</el-button>
            <el-button v-auth="'CFS_SYNCRULE_DELETE'" type="text" style="color: #ed4014" @click="deleteRule(scope.row)">{{ $t('common.delete') }}</el-button>
          </MoreOPerate>
        </template>
      </el-table-column>
    </u-page-table>
    <SyncRuleCreateDialog
      :visible.sync="createDialogVisible"
      :cluster-name="clusterName"
      @confirm="handleCreateConfirm"
    />
    <SyncRuleEditorDialog
      :visible.sync="editDialogVisible"
      mode="edit"
      :value="editorValue"
      @confirm="handleEditConfirm"
    />
  </div>
</template>

<script>
import { formatDate } from '@/utils'
import MoreOPerate from '@/pages/components/moreOPerate'
import UPageTable from '@/pages/components/uPageTable'
import {
  createSyncRule,
  deleteSyncRule,
  getSyncRuleList,
  pauseSyncRule,
  resumeSyncRule,
  triggerSyncRule,
  updateSyncRule,
} from '@/api/cfs/cluster'
import SyncRuleEditorDialog from './SyncRuleEditorDialog.vue'
import SyncRuleCreateDialog from './SyncRuleCreateDialog.vue'

export default {
  name: 'SyncRuleTab',
  components: {
    MoreOPerate,
    SyncRuleEditorDialog,
    SyncRuleCreateDialog,
    UPageTable,
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
      filters: {
        state: '',
      },
      states: ['active', 'paused', 'degraded'],
      createDialogVisible: false,
      editDialogVisible: false,
      editorValue: '',
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
      const { data } = await getSyncRuleList({
        cluster_name: this.clusterName,
        state: this.filters.state,
      })
      this.dataList = data || []
    },
    resetFilters() {
      this.filters.state = ''
      this.loadData()
    },
    getConfig(row, key) {
      return row?.config?.[key]
    },
    formatTime(value) {
      if (!value) return '-'
      const d = new Date(value)
      if (isNaN(d.getTime()) || d.getFullYear() < 2000) return '-'
      return formatDate(value)
    },
    openCreate() {
      this.createDialogVisible = true
    },
    openEdit(row) {
      this.editorValue = JSON.stringify(row?.config || {}, null, 2)
      this.editDialogVisible = true
    },
    emitViewTasks(ruleID) {
      this.$emit('view-tasks', ruleID)
    },
    async handleCreateConfirm(text, runImmediately) {
      let body = null
      try {
        body = JSON.parse(text)
      } catch (_) {
        this.$message.error(this.$t('common.failed') + ': JSON parse error')
        return
      }
      await createSyncRule({ cluster_name: this.clusterName, ...body })
      this.$message.success(this.$t('common.create') + this.$t('common.xxsuc'))
      this.createDialogVisible = false
      if (runImmediately && body.id) {
        await triggerSyncRule({ cluster_name: this.clusterName, id: body.id })
        this.$message.success(this.$t('sync.trigger') + this.$t('common.xxsuc'))
        this.emitViewTasks(body.id)
      }
      this.loadData()
    },
    async handleEditConfirm(text) {
      let body = null
      try {
        body = JSON.parse(text)
      } catch (_) {
        this.$message.error(this.$t('common.failed') + ': JSON parse error')
        return
      }
      await updateSyncRule({ cluster_name: this.clusterName, ...body })
      this.$message.success(this.$t('common.edit') + this.$t('common.xxsuc'))
      this.editDialogVisible = false
      this.loadData()
    },
    async triggerRule(row) {
      const id = this.getConfig(row, 'id')
      await triggerSyncRule({
        cluster_name: this.clusterName,
        id,
      })
      this.$message.success(this.$t('sync.trigger') + this.$t('common.xxsuc'))
      this.emitViewTasks(id)
      this.loadData()
    },
    async pauseRule(row) {
      await pauseSyncRule({
        cluster_name: this.clusterName,
        id: this.getConfig(row, 'id'),
      })
      this.$message.success(this.$t('common.off') + this.$t('common.xxsuc'))
      this.loadData()
    },
    async resumeRule(row) {
      await resumeSyncRule({
        cluster_name: this.clusterName,
        id: this.getConfig(row, 'id'),
      })
      this.$message.success(this.$t('common.on') + this.$t('common.xxsuc'))
      this.loadData()
    },
    async deleteRule(row) {
      await this.$confirm(`${this.$t('common.confirmto')}${this.$t('common.delete')} ${this.getConfig(row, 'id')} ?`, this.$t('common.notice'), {
        confirmButtonText: this.$t('common.yes'),
        cancelButtonText: this.$t('common.no'),
      })
      await deleteSyncRule({
        cluster_name: this.clusterName,
        id: this.getConfig(row, 'id'),
      })
      this.$message.success(this.$t('common.delete') + this.$t('common.xxsuc'))
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
