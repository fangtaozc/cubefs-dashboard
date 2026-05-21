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
        <div class="title">{{ clusterName }} / {{ $t('router.testmanage') }}</div>
      </div>
      <el-tabs v-model="activeName">
        <el-tab-pane :label="$t('bench.rules')" name="rules">
          <BenchRuleTab
            :cluster-name="clusterName"
            @view-tasks="handleRuleViewTasks"
          />
        </el-tab-pane>
        <el-tab-pane :label="$t('bench.tasks')" name="tasks">
          <BenchTaskTab
            :cluster-name="clusterName"
            :filters="taskFilters"
          />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<script>
import mixin from '@/pages/cfs/clusterOverview/mixin'
import BenchRuleTab from './components/BenchRuleTab.vue'
import BenchTaskTab from './components/BenchTaskTab.vue'

export default {
  name: 'TestManage',
  components: {
    BenchRuleTab,
    BenchTaskTab,
  },
  mixins: [mixin],
  data() {
    return {
      activeName: 'rules',
      taskFilters: {
        status: '',
        ruleID: '',
      },
    }
  },
  watch: {
    '$route.query': {
      immediate: true,
      handler(query) {
        this.activeName = query?.benchTab === 'tasks' ? 'tasks' : 'rules'
        this.taskFilters = {
          status: query?.status || '',
          ruleID: query?.ruleID || '',
        }
      },
    },
  },
  methods: {
    handleRuleViewTasks(ruleID = '') {
      this.taskFilters = { status: '', ruleID }
      this.activeName = 'tasks'
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
