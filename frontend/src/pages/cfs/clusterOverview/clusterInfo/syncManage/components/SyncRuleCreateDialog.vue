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
    :title="`${$t('common.create')}${$t('sync.rules')}`"
    width="780px"
    append-to-body
    @open="onOpen"
    @close="handleClose"
  >
    <el-tabs v-model="activeTab" @tab-click="onTabClick">
      <el-tab-pane :label="$t('sync.basicinfo')" name="form">
        <el-form label-width="120px" size="small" class="rule-form">
          <el-form-item label="Rule ID" required>
            <el-input v-model.trim="form.id" placeholder="r-local-to-s3" clearable></el-input>
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="form.type" style="width: 180px;">
              <el-option label="sync" value="sync"></el-option>
              <el-option label="copy" value="copy"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="执行方式">
            <el-radio-group v-model="form.scheduleMode">
              <el-radio label="cron">定时调度</el-radio>
              <el-radio label="once">立即执行（一次）</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="form.scheduleMode === 'cron'" :label="$t('sync.schedule')">
            <el-input v-model.trim="form.schedule" placeholder="*/30 * * * * *" clearable></el-input>
          </el-form-item>
          <el-form-item :label="$t('sync.parallelism')">
            <el-input-number v-model="form.parallelism" :min="1" :max="64" :precision="0" style="width: 160px;"></el-input-number>
            <span class="field-hint">单任务内并发文件数</span>
          </el-form-item>
          <el-form-item label="分片策略">
            <el-select v-model="form.shardingStrategy" style="width: 220px;">
              <el-option label="不分片（单节点执行）" value=""></el-option>
              <el-option label="hash（按 key 哈希均匀分片）" value="hash"></el-option>
              <el-option label="auto（master 自动探测前缀）" value="auto"></el-option>
              <el-option label="prefix（按指定前缀分片）" value="prefix"></el-option>
            </el-select>
            <div class="field-hint" style="margin-left: 0; margin-top: 4px;">
              <template v-if="!form.shardingStrategy">不分片：任务派发到单个节点顺序执行，适合数据量较小的规则</template>
              <template v-else-if="form.shardingStrategy === 'hash'">hash：分片数自动截断为 min(并发度, 在线节点数)，任务分发到不同节点并行；单节点时退化为 1 个任务，分片无效</template>
              <template v-else-if="form.shardingStrategy === 'auto'">auto：master 自动探测后端顶层前缀按前缀分片；单节点时多个分片串行执行，但每个分片只 list 自己的前缀，适合大数据量场景</template>
              <template v-else>prefix：手动指定前缀，分片数 = 前缀数；单节点时串行执行，但每个分片只 list 自己的前缀，适合大数据量场景</template>
            </div>
          </el-form-item>
          <el-form-item v-if="form.shardingStrategy === 'prefix' || form.shardingStrategy === 'auto'" label="分片前缀">
            <el-input
              v-model="form.shardPrefixesText"
              type="textarea"
              :rows="3"
              placeholder="每行一个前缀，如：&#10;prefix-a/&#10;prefix-b/"
            ></el-input>
            <div class="field-hint">
              {{ form.shardingStrategy === 'prefix' ? 'prefix 策略必填，每行一个前缀，分片数 = 前缀数' : 'auto 策略可选，作为白名单过滤 master 自动探测到的前缀' }}
            </div>
          </el-form-item>

          <el-divider>Src（源）</el-divider>
          <el-form-item label="Src 类型">
            <el-radio-group v-model="form.srcKind">
              <el-radio label="local">local</el-radio>
              <el-radio label="cubefs">cubefs</el-radio>
              <el-radio label="backend">已配置后端</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'local'" label="Src 路径">
            <el-input v-model.trim="form.srcPath" placeholder="/data/source/" clearable></el-input>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'cubefs'" label="卷名">
            <el-select v-model="form.srcVolume" filterable clearable placeholder="选择或输入卷名" style="width: 100%;">
              <el-option v-for="v in volumes" :key="v.name" :label="v.name" :value="v.name"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'cubefs'" label="Src 前缀">
            <el-input v-model.trim="form.srcPrefix" placeholder="/prefix/（可选）" clearable></el-input>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'backend'" label="Src 后端">
            <el-select v-model="form.srcBackendId" clearable placeholder="选择存储后端" style="width: 100%;">
              <el-option
                v-for="b in backends"
                :key="b.id"
                :label="`[${b.kind}] ${b.name} (${b.bucket})`"
                :value="b.id"
              ></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'backend'" label="Src 前缀">
            <el-input v-model.trim="form.srcBackendPrefix" placeholder="可选，覆盖路径前缀" clearable></el-input>
          </el-form-item>

          <el-divider>Dst（目标）</el-divider>
          <el-form-item label="Dst 类型">
            <el-radio-group v-model="form.dstKind">
              <el-radio label="local">local</el-radio>
              <el-radio label="cubefs">cubefs</el-radio>
              <el-radio label="backend">已配置后端</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'local'" label="Dst 路径">
            <el-input v-model.trim="form.dstPath" placeholder="/data/dest/" clearable></el-input>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'cubefs'" label="卷名">
            <el-select v-model="form.dstVolume" filterable clearable placeholder="选择或输入卷名" style="width: 100%;">
              <el-option v-for="v in volumes" :key="v.name" :label="v.name" :value="v.name"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'cubefs'" label="Dst 前缀">
            <el-input v-model.trim="form.dstPrefix" placeholder="/prefix/（可选）" clearable></el-input>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'backend'" label="Dst 后端">
            <el-select v-model="form.dstBackendId" clearable placeholder="选择存储后端" style="width: 100%;">
              <el-option
                v-for="b in backends"
                :key="b.id"
                :label="`[${b.kind}] ${b.name} (${b.bucket})`"
                :value="b.id"
              ></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'backend'" label="Dst 前缀">
            <el-input v-model.trim="form.dstBackendPrefix" placeholder="可选，覆盖路径前缀" clearable></el-input>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="JSON" name="json">
        <el-input
          v-model="editorValue"
          type="textarea"
          :rows="20"
          resize="none"
          placeholder="输入规则 JSON"
        ></el-input>
      </el-tab-pane>
    </el-tabs>

    <div slot="footer">
      <el-button @click="handleClose">{{ $t('button.cancel') }}</el-button>
      <el-button type="primary" :loading="submitting" @click="handleConfirm">{{ $t('button.submit') }}</el-button>
    </div>
  </el-dialog>
</template>

<script>
import { getSyncStorageBackendConfig, getSyncStorageBackendList, getVolList } from '@/api/cfs/cluster'

const emptyForm = () => ({
  id: '',
  type: 'sync',
  scheduleMode: 'cron',
  schedule: '',
  parallelism: 3,
  shardingStrategy: 'hash',
  shardPrefixesText: '',
  srcKind: 'local',
  srcPath: '',
  srcVolume: '',
  srcPrefix: '',
  srcBackendId: null,
  srcBackendPrefix: '',
  dstKind: 'backend',
  dstPath: '',
  dstVolume: '',
  dstPrefix: '',
  dstBackendId: null,
  dstBackendPrefix: '',
})

export default {
  name: 'SyncRuleCreateDialog',
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    clusterName: {
      type: String,
      required: true,
    },
  },
  data() {
    return {
      innerVisible: false,
      activeTab: 'form',
      form: emptyForm(),
      editorValue: '',
      backends: [],
      backendConfigs: {},
      volumes: [],
      submitting: false,
    }
  },
  computed: {
    generatedJson() {
      const payload = {
        id: this.form.id || 'r-unnamed',
        type: this.form.type,
      }
      if (this.form.scheduleMode === 'cron' && this.form.schedule) {
        payload.schedule = this.form.schedule
      }
      const src = this.buildStorageConfig('src')
      if (src) payload.src = src
      const dst = this.buildStorageConfig('dst')
      if (dst) payload.dst = dst
      if (this.form.parallelism > 0) {
        payload.parallelism = this.form.parallelism
      }
      if (this.form.shardingStrategy) {
        payload.shardingStrategy = this.form.shardingStrategy
      }
      const prefixes = this.form.shardPrefixesText
        .split('\n')
        .map(s => s.trim())
        .filter(Boolean)
      if (prefixes.length) {
        payload.shardPrefixes = prefixes
      }
      return JSON.stringify(payload, null, 2)
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
    async onOpen() {
      await Promise.all([this.loadBackends(), this.loadVolumes()])
    },
    onTabClick(tab) {
      if (tab.name === 'json') {
        this.editorValue = this.generatedJson
      }
    },
    async loadBackends() {
      try {
        const { data } = await getSyncStorageBackendList({ cluster_name: this.clusterName })
        this.backends = data || []
      } catch (_) {
        this.backends = []
      }
    },
    async loadVolumes() {
      try {
        const { data } = await getVolList({ cluster_name: this.clusterName })
        this.volumes = data || []
      } catch (_) {
        this.volumes = []
      }
    },
    buildStorageConfig(side) {
      const kind = this.form[`${side}Kind`]
      if (kind === 'local') {
        const path = this.form[`${side}Path`]
        if (!path) return null
        return { kind: 'local', path }
      }
      if (kind === 'cubefs') {
        const volume = this.form[`${side}Volume`]
        if (!volume) return null
        const cfg = { kind: 'cfs-sync', volume }
        const prefix = this.form[`${side}Prefix`]
        if (prefix) cfg.prefix = prefix
        return cfg
      }
      if (kind === 'backend') {
        const backendId = this.form[`${side}BackendId`]
        if (!backendId) return null
        const cfg = this.backendConfigs[backendId]
        if (!cfg) {
          return { kind: 's3', __pending: backendId }
        }
        const result = { ...cfg }
        const prefix = this.form[`${side}BackendPrefix`]
        if (prefix) result.prefix = prefix
        return result
      }
      return null
    },
    handleClose() {
      this.form = emptyForm()
      this.editorValue = ''
      this.activeTab = 'form'
      this.submitting = false
      this.$emit('update:visible', false)
      this.$emit('close')
    },
    async handleConfirm() {
      const runImmediately = this.form.scheduleMode === 'once'
      if (this.activeTab === 'form') {
        if (!this.form.id) {
          this.$message.warning('Rule ID 不能为空')
          return
        }
        this.submitting = true
        try {
          await this.resolveBackendConfigs()
          this.$emit('confirm', this.generatedJson, runImmediately)
        } finally {
          this.submitting = false
        }
      } else {
        this.$emit('confirm', this.editorValue, false)
      }
    },
    async resolveBackendConfigs() {
      const ids = new Set()
      if (this.form.srcKind === 'backend' && this.form.srcBackendId) ids.add(this.form.srcBackendId)
      if (this.form.dstKind === 'backend' && this.form.dstBackendId) ids.add(this.form.dstBackendId)
      await Promise.all(
        [...ids].map(async (id) => {
          if (this.backendConfigs[id]) return
          try {
            const { data } = await getSyncStorageBackendConfig({ cluster_name: this.clusterName, id })
            this.$set(this.backendConfigs, id, data)
          } catch (_) {
            // config fetch failed — buildStorageConfig returns placeholder
          }
        }),
      )
    },
  },
}
</script>

<style lang="scss" scoped>
.rule-form {
  margin-top: 4px;
}

.field-hint {
  margin-left: 10px;
  font-size: 12px;
  color: #9ca3af;
}

::v-deep .el-divider__text {
  font-size: 12px;
  color: #6b7280;
}
</style>
