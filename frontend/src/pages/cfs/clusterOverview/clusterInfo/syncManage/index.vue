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
    <el-card>
      <div class="toolbar">
        <div class="title">{{ clusterName }} / {{ $t('router.syncmanage') }}</div>
      </div>
      <el-tabs v-model="activeName">
        <el-tab-pane :label="$t('sync.rules')" name="rules">
          <SyncRuleTab
            :cluster-name="clusterName"
            @view-tasks="handleRuleViewTasks"
          />
        </el-tab-pane>
        <el-tab-pane :label="$t('sync.tasks')" name="tasks">
          <SyncTaskTab
            :cluster-name="clusterName"
            :filters="taskFilters"
            @show-worker="handleTaskShowWorker"
          />
        </el-tab-pane>
        <el-tab-pane :label="$t('sync.storagebackends')" name="backends">
          <SyncStorageBackendTab :cluster-name="clusterName" />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script>
import mixin from '@/pages/cfs/clusterOverview/mixin'
import SyncRuleTab from './components/SyncRuleTab.vue'
import SyncTaskTab from './components/SyncTaskTab.vue'
import SyncStorageBackendTab from './components/SyncStorageBackendTab.vue'

export default {
  name: 'SyncManage',
  components: {
    SyncRuleTab,
    SyncTaskTab,
    SyncStorageBackendTab,
  },
  mixins: [mixin],
  data() {
    return {
      activeName: 'rules',
      taskFilters: {
        status: '',
        ruleID: '',
        owner: '',
      },
    }
  },
  watch: {
    '$route.query': {
      immediate: true,
      handler(query) {
        this.activeName = query?.syncTab === 'tasks' ? 'tasks' : 'rules'
        this.taskFilters = {
          status: query?.status || '',
          ruleID: query?.ruleID || '',
          owner: query?.owner || '',
        }
      },
    },
  },
  methods: {
    handleRuleViewTasks(ruleID = '') {
      this.taskFilters = { status: '', ruleID, owner: '' }
      this.activeName = 'tasks'
    },
    handleTaskShowWorker(owner = '') {
      if (!owner) {
        return
      }
      this.$router.push({
        name: 'resourceManage',
        query: {
          nodeTab: 'syncNode',
          syncNodeAddr: owner,
          syncNodeDialog: '1',
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
  margin-bottom: 16px;
}

.title {
  font-size: 16px;
  font-weight: 600;
}
</style>
