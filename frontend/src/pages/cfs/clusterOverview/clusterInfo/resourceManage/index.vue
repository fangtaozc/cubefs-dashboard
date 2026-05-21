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
    <el-radio-group v-model="activeName" style="margin-bottom: 10px;">
      <el-radio-button label="node">{{ $t('common.replica') }}{{ $t('common.nodes') }}</el-radio-button>
      <el-radio-button label="syncNode">{{ $t('sync.workernodes') }}</el-radio-button>
      <el-radio-button label="blobStoreNode" :disabled="!ebsClusterList || !ebsClusterList.length">{{ $t('common.ec') }}{{ $t('common.nodes') }}</el-radio-button>
      <el-radio-button label="metaNode">{{ $t('common.meta') }}{{ $t('common.nodes') }}</el-radio-button>
    </el-radio-group>

    <div v-if="activeName === 'syncNode'" class="mg-bt-s flex">
      <span class="fontType"><span>{{ $t('sync.workernodecount') }}:</span> <span class="mg-lf-m"></span>{{ info.node || 0 }}</span>
      <span class="fontType mg-lf-m"><span>{{ $t('sync.runningtasks') }}:</span> <span class="mg-lf-m"></span>{{ info.runningTasks || 0 }}</span>
      <span class="fontType mg-lf-m"><span>{{ $t('sync.queuedtasks') }}:</span> <span class="mg-lf-m"></span>{{ info.queuedTasks || 0 }}</span>
      <span class="fontType mg-lf-m"><span>{{ $t('sync.avgloadscore') }}:</span> <span class="mg-lf-m"></span>{{ info.avgLoadScore || '0.00' }}</span>
    </div>

    <div v-else-if="activeName !== 'blobStoreNode'" class="mg-bt-s flex">
      <span class="fontType"><span>{{ $t('common.total') }}{{ $t('common.nodes') }}:</span> <span class="mg-lf-m"></span>{{ info.node || 0 }}</span>
      <span class="fontType mg-lf-m"><span>{{ $t('common.total') }}{{ $t('common.partitions') }}:</span> <span class="mg-lf-m"></span>{{ info.partition || 0 }}</span>
      <span class="fontType mg-lf-m"><span>{{ $t('common.total') }}{{ $t('common.size') }}:</span> <span class="mg-lf-m"></span>{{ (info.total || 0) | renderSize }}</span>
      <div class="mg-lf-m progress">
        <span>{{ (info.used || 0) | renderSize }}/{{ usagePercentText }}</span>
        <el-progress
          v-if="info.node !== 0"
          :stroke-width="10"
          :show-text="false"
          :percentage="usagePercent"
          :color="[
            { color: '#f56c6c', percentage: 100 },
            { color: '#e6a23c', percentage: 80 },
            { color: '#5cb87a', percentage: 60 },
            { color: '#1989fa', percentage: 40 },
            { color: '#6f7ad3', percentage: 20 },
          ]"
        >
        </el-progress>
      </div>
    </div>

    <component
      :is="items[activeName].component"
      v-if="items[activeName].name === activeName"
      v-bind="activeComponentProps"
      :info.sync="info"
    />
  </div>
</template>

<script>
import Node from './dataNode/node.vue'
import MetaNode from './metaNode/metaNode.vue'
import BlobStoreNode from './blobStoreNode/index.vue'
import SyncNode from './syncNode/index.vue'
import { renderSize } from '@/utils'
import mixin from '@/pages/cfs/clusterOverview/mixin'

export default {
  name: 'ResourceManage',
  components: {
    BlobStoreNode,
    MetaNode,
    Node,
    SyncNode,
  },
  filters: {
    renderSize(val) {
      return renderSize(val, 1)
    },
  },
  mixins: [mixin],
  data() {
    return {
      activeName: 'node',
      items: {
        node: {
          name: 'node',
          component: 'Node',
        },
        syncNode: {
          name: 'syncNode',
          component: 'SyncNode',
        },
        metaNode: {
          name: 'metaNode',
          component: 'MetaNode',
        },
        blobStoreNode: {
          name: 'blobStoreNode',
          component: 'BlobStoreNode',
        },
      },
      info: {
        node: 0,
        partition: 0,
        total: 0,
        used: 0,
        runningTasks: 0,
        queuedTasks: 0,
        avgLoadScore: '0.00',
      },
    }
  },
  computed: {
    usagePercent() {
      if (!this.info.total) {
        return 0
      }
      return Number(((this.info.used / this.info.total) * 100).toFixed(0))
    },
    usagePercentText() {
      return `${this.usagePercent}%`
    },
    activeComponentProps() {
      if (this.activeName === 'syncNode') {
        return {
          focusAddr: this.$route.query.syncNodeAddr || '',
          openDetail: this.$route.query.syncNodeDialog === '1',
        }
      }
      return {}
    },
  },
  watch: {
    '$route.query': {
      immediate: true,
      handler(query) {
        if (query?.nodeTab && this.items[query.nodeTab]) {
          this.activeName = query.nodeTab
        }
      },
    },
  },
}
</script>

<style lang="scss" scoped>
.fontType {
  font-family: 'Microsoft YaHei';
  font-style: normal;
  font-weight: 400;
  font-size: 14px;
  line-height: 14px;
  color: #000000;
}

.progress {
  width: 100px;
  position: relative;
  top: -5px;
  left: 10px;
}
</style>
