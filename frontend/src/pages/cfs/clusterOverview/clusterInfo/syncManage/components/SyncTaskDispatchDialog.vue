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
    :title="$t('sync.dispatchtask')"
    width="720px"
    append-to-body
    @open="onOpen"
    @close="handleClose"
  >
    <el-alert
      :title="$t('sync.dispatchtips')"
      type="info"
      :closable="false"
      class="mg-bt-s"
    ></el-alert>

    <el-tabs v-model="activeTab" @tab-click="onTabClick">
      <el-tab-pane :label="$t('sync.basicinfo')" name="form">
        <el-form label-width="110px" size="small" class="dispatch-form">
          <el-form-item label="ruleID" required>
            <el-input v-model.trim="form.ruleId" placeholder="可关联已有规则" clearable></el-input>
          </el-form-item>
          <el-form-item label="taskID">
            <el-input v-model.trim="form.taskId" placeholder="留空自动生成" clearable></el-input>
          </el-form-item>
          <el-form-item label="并发度">
            <el-input-number v-model="form.parallelism" :min="0" :precision="0" style="width: 160px;"></el-input-number>
            <span class="field-hint">单任务内并发文件数，0 = 沿用规则配置</span>
          </el-form-item>
          <el-form-item :label="$t('sync.shardtotal')">
            <el-input-number v-model="form.shardTotal" :min="0" :precision="0" style="width: 160px;"></el-input-number>
            <span class="field-hint">将任务切分为 N 个分片并行执行，≤1 = 不分片</span>
          </el-form-item>

          <el-divider>Src（可选）</el-divider>
          <el-form-item label="Src 类型">
            <el-radio-group v-model="form.srcKind">
              <el-radio label="">不指定</el-radio>
              <el-radio label="local">local</el-radio>
              <el-radio label="cubefs">cubefs</el-radio>
              <el-radio label="backend">已配置后端</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'local'" label="Src Path">
            <el-input v-model.trim="form.srcPath" placeholder="/data/path/" clearable></el-input>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'cubefs'" label="CubeFS Volume">
            <el-input v-model.trim="form.srcVolume" placeholder="volume name" clearable></el-input>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'cubefs'" label="Src Prefix">
            <el-input v-model.trim="form.srcPrefix" placeholder="/prefix/" clearable></el-input>
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
          <el-form-item v-if="form.srcKind === 'backend' && form.srcBackendId" label="Src 路径前缀">
            <el-input v-model.trim="form.srcBackendPrefix" placeholder="覆盖后端默认前缀（可选）" clearable></el-input>
          </el-form-item>

          <el-divider>Dst（可选）</el-divider>
          <el-form-item label="Dst 类型">
            <el-radio-group v-model="form.dstKind">
              <el-radio label="">不指定</el-radio>
              <el-radio label="local">local</el-radio>
              <el-radio label="cubefs">cubefs</el-radio>
              <el-radio label="backend">已配置后端</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'local'" label="Dst Path">
            <el-input v-model.trim="form.dstPath" placeholder="/data/path/" clearable></el-input>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'cubefs'" label="CubeFS Volume">
            <el-input v-model.trim="form.dstVolume" placeholder="volume name" clearable></el-input>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'cubefs'" label="Dst Prefix">
            <el-input v-model.trim="form.dstPrefix" placeholder="/prefix/" clearable></el-input>
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
          <el-form-item v-if="form.dstKind === 'backend' && form.dstBackendId" label="Dst 路径前缀">
            <el-input v-model.trim="form.dstBackendPrefix" placeholder="覆盖后端默认前缀（可选）" clearable></el-input>
          </el-form-item>

        </el-form>
      </el-tab-pane>

      <el-tab-pane label="JSON" name="json">
        <el-input
          v-model="editorValue"
          type="textarea"
          :rows="16"
          resize="none"
          :placeholder="$t('sync.taskpayload')"
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
import { getSyncStorageBackendConfig, getSyncStorageBackendList } from '@/api/cfs/cluster'

const emptyForm = () => ({
  ruleId: '',
  taskId: '',
  parallelism: 0,
  shardTotal: 0,
  srcKind: '',
  srcPath: '',
  srcVolume: '',
  srcPrefix: '',
  srcBackendId: null,
  srcBackendPrefix: '',
  dstKind: '',
  dstPath: '',
  dstVolume: '',
  dstPrefix: '',
  dstBackendId: null,
  dstBackendPrefix: '',
})

export default {
  name: 'SyncTaskDispatchDialog',
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    clusterName: {
      type: String,
      required: true,
    },
    value: {
      type: String,
      default: '',
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
      submitting: false,
    }
  },
  computed: {
    generatedJson() {
      const taskId = this.form.taskId || `manual-${Date.now()}`
      const payload = {
        id: taskId,
        opcode: 121,
        Request: {
          taskId,
          ruleId: this.form.ruleId,
        },
      }
      if (this.form.parallelism > 0) {
        payload.Request.parallelism = this.form.parallelism
      }
      if (this.form.shardTotal > 1) {
        payload.Request.shardTotal = this.form.shardTotal
      }
      const src = this.buildStorageConfig(this.form.srcKind, {
        path: this.form.srcPath,
        volume: this.form.srcVolume,
        prefix: this.form.srcPrefix,
        backendId: this.form.srcBackendId,
        backendPrefix: this.form.srcBackendPrefix,
      })
      if (src) {
        payload.Request.src = src
      }
      const dst = this.buildStorageConfig(this.form.dstKind, {
        path: this.form.dstPath,
        volume: this.form.dstVolume,
        prefix: this.form.dstPrefix,
        backendId: this.form.dstBackendId,
        backendPrefix: this.form.dstBackendPrefix,
      })
      if (dst) {
        payload.Request.dst = dst
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
    value: {
      immediate: true,
      handler(val) {
        this.editorValue = val
        if (val) {
          this.tryParseIntoForm(val)
        }
      },
    },
  },
  methods: {
    async onOpen() {
      await this.loadBackends()
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
    buildStorageConfig(kind, opts) {
      if (!kind) return null
      if (kind === 'local') {
        if (!opts.path) return null
        return { kind: 'local', path: opts.path }
      }
      if (kind === 'cubefs') {
        if (!opts.volume) return null
        const cfg = { kind: 'cfs-sync', volume: opts.volume }
        if (opts.prefix) cfg.prefix = opts.prefix
        return cfg
      }
      if (kind === 'backend') {
        const cfg = this.backendConfigs[opts.backendId]
        if (!cfg) return null
        const result = { ...cfg }
        if (opts.backendPrefix) {
          result.prefix = opts.backendPrefix
        }
        return result
      }
      return null
    },
    tryParseIntoForm(json) {
      try {
        const obj = JSON.parse(json)
        this.form.taskId = obj.id || obj.Request?.taskId || ''
        this.form.ruleId = obj.Request?.ruleId || obj.Request?.ruleID || ''
        this.form.parallelism = obj.Request?.parallelism || 0
        this.form.shardTotal = obj.Request?.shardTotal || 0
      } catch (_) {
        // non-parseable JSON — leave form empty
      }
    },
    handleClose() {
      this.form = emptyForm()
      this.activeTab = 'form'
      this.submitting = false
      this.$emit('update:visible', false)
      this.$emit('close')
    },
    async handleConfirm() {
      if (this.activeTab === 'form') {
        if (!this.form.ruleId) {
          this.$message.warning('ruleID 不能为空')
          return
        }
        // resolve backend configs before emitting
        this.submitting = true
        try {
          await this.resolveBackendConfigs()
          this.$emit('confirm', this.generatedJson)
        } finally {
          this.submitting = false
        }
      } else {
        this.$emit('confirm', this.editorValue)
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
            this.backendConfigs[id] = data
          } catch (_) {
            // config fetch failed — leave as null, buildStorageConfig will skip it
          }
        }),
      )
    },
  },
}
</script>

<style lang="scss" scoped>
.dispatch-form {
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
