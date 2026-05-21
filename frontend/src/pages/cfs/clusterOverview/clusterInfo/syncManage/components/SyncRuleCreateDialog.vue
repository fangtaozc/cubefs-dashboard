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
    :title="isViewMode ? `${$t('sync.rules')}${$t('common.detail')}` : isEditMode ? `${$t('common.edit')}${$t('sync.rules')}` : `${$t('common.create')}${$t('sync.rules')}`"
    width="780px"
    append-to-body
    @open="onOpen"
    @close="handleClose"
  >
    <el-tabs v-model="activeTab" @tab-click="onTabClick">
      <el-tab-pane :label="$t('sync.basicinfo')" name="form">
        <el-form label-width="120px" size="small" class="rule-form">
          <el-form-item label="Rule ID" required>
            <el-input v-model.trim="form.id" placeholder="r-local-to-s3" clearable :readonly="isEditMode || isViewMode" :disabled="isEditMode || isViewMode"></el-input>
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="form.type" style="width: 180px;" :disabled="isViewMode">
              <el-option label="sync" value="sync"></el-option>
              <el-option label="load" value="load"></el-option>
              <el-option label="check" value="check"></el-option>
            </el-select>
            <div class="field-hint" style="margin-left: 0; margin-top: 4px;">
              <template v-if="form.type === 'sync'">sync：增量同步 src → dst，跳过已存在且一致的文件</template>
              <template v-else-if="form.type === 'load'">load：全量加载 src → dst，写后验证 size；默认 temp_rename 原子写入</template>
              <template v-else>check：双向一致性校验，不移动数据；可配合 auto_fix 自动修复</template>
            </div>
          </el-form-item>
          <el-form-item label="执行方式">
            <el-radio-group v-model="form.scheduleMode" :disabled="isViewMode">
              <el-radio label="cron">定时调度</el-radio>
              <el-radio label="once">{{ isEditMode || isViewMode ? '无调度（手动触发）' : '立即执行（一次）' }}</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="form.scheduleMode === 'cron'" :label="$t('sync.schedule')">
            <el-input v-model.trim="form.schedule" placeholder="*/30 * * * * *" clearable :disabled="isViewMode"></el-input>
          </el-form-item>
          <el-form-item :label="$t('sync.parallelism')">
            <el-input-number v-model="form.parallelism" :min="1" :max="64" :precision="0" style="width: 160px;" :disabled="isViewMode"></el-input-number>
            <span class="field-hint">单任务内并发文件数</span>
          </el-form-item>

          <!-- sync 特有 -->
          <el-form-item v-if="form.type === 'sync'" label="拷贝后处理">
            <el-select v-model="form.afterCopy" style="width: 280px;" :disabled="isViewMode">
              <el-option label="保留源文件（默认）" value=""></el-option>
              <el-option label="verify_then_delete_src（验证后删源）" value="verify_then_delete_src"></el-option>
            </el-select>
            <div class="field-hint" style="margin-left: 0; margin-top: 4px;">
              verify_then_delete_src：写入 dst 并 Head 确认 size 一致后删除 src，实现迁移语义
            </div>
          </el-form-item>

          <!-- load 特有 -->
          <el-form-item v-if="form.type === 'load'" label="下载策略">
            <el-select v-model="form.downloadStrategy" style="width: 280px;" :disabled="isViewMode">
              <el-option label="temp_rename（默认，原子写入）" value=""></el-option>
              <el-option label="direct（直接写入目标 key）" value="direct"></el-option>
            </el-select>
            <div class="field-hint" style="margin-left: 0; margin-top: 4px;">
              temp_rename：先写 &lt;dst&gt;.downloading.&lt;taskID&gt; 再 rename，防止目标出现半写文件
            </div>
          </el-form-item>

          <!-- check 特有 -->
          <el-form-item v-if="form.type === 'check'" label="不一致处理">
            <el-select v-model="form.onMismatch" style="width: 280px;" :disabled="isViewMode">
              <el-option label="alert（默认，仅上报差异）" value=""></el-option>
              <el-option label="auto_fix（自动调度 sync 子任务修复）" value="auto_fix"></el-option>
              <el-option label="ignore（忽略差异，不上报）" value="ignore"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.type === 'check'" label="采样策略">
            <el-select v-model="form.sampleStrategy" style="width: 200px;" :disabled="isViewMode">
              <el-option label="full（全量，默认）" value=""></el-option>
              <el-option label="random（随机采样）" value="random"></el-option>
              <el-option label="oldest（最旧优先）" value="oldest"></el-option>
              <el-option label="largest（最大优先）" value="largest"></el-option>
            </el-select>
            <span v-if="!form.sampleStrategy || form.sampleStrategy === 'full'" class="field-hint">对所有文件执行 ETag 校验（最慢但最准确）</span>
            <span v-else class="field-hint">仅对采样子集执行校验，适合快速抽检</span>
          </el-form-item>
          <el-form-item v-if="form.type === 'check' && form.sampleStrategy && form.sampleStrategy !== 'full'" label="采样比例">
            <el-input-number v-model="form.sampleRate" :min="0.01" :max="1" :precision="2" :step="0.1" style="width: 160px;" :disabled="isViewMode"></el-input-number>
            <span class="field-hint">0.01 ~ 1.0；实际采样数 = floor(总不一致数 × 采样比例)</span>
          </el-form-item>
          <el-form-item label="分片策略">
            <el-select v-model="form.shardingStrategy" style="width: 220px;" :disabled="isViewMode">
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
              :disabled="isViewMode"
            ></el-input>
            <div class="field-hint">
              {{ form.shardingStrategy === 'prefix' ? 'prefix 策略必填，每行一个前缀，分片数 = 前缀数' : 'auto 策略可选，作为白名单过滤 master 自动探测到的前缀' }}
            </div>
          </el-form-item>

          <el-divider>Src（源）</el-divider>
          <el-form-item label="Src 类型">
            <el-radio-group v-model="form.srcKind" :disabled="isViewMode">
              <el-radio label="local">local</el-radio>
              <el-radio label="cubefs">cubefs</el-radio>
              <el-radio label="backend">已配置后端</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'local'" label="Src 路径">
            <MountPathSelect
              v-model="form.srcPath"
              :options="mountPathOptions"
              :disabled="isViewMode"
              placeholder="/data/source/  （选择已挂载路径或自行输入）"
              hint="候选来自 syncnode 实际容器内的挂载点；选中后仍可继续编辑或清空。"
            />
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'cubefs'" label="卷名">
            <el-select v-model="form.srcVolume" filterable clearable placeholder="选择或输入卷名" style="width: 100%;" :disabled="isViewMode">
              <el-option v-for="v in volumes" :key="v.name" :label="v.name" :value="v.name"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'cubefs'" label="Src 路径">
            <el-input v-model.trim="form.srcPrefix" placeholder="/ （留空默认根目录）" clearable :disabled="isViewMode"></el-input>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'backend'" label="Src 后端">
            <el-select v-model="form.srcBackendId" clearable placeholder="选择存储后端" style="width: 100%;" :disabled="isViewMode">
              <el-option
                v-for="b in backends"
                :key="b.id"
                :label="`[${b.kind}] ${b.name} (${b.bucket})`"
                :value="b.id"
              ></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.srcKind === 'backend'" label="Src 前缀">
            <el-input v-model.trim="form.srcBackendPrefix" placeholder="可选，覆盖路径前缀" clearable :disabled="isViewMode"></el-input>
          </el-form-item>

          <el-divider>Dst（目标）</el-divider>
          <el-form-item label="Dst 类型">
            <el-radio-group v-model="form.dstKind" :disabled="isViewMode">
              <el-radio label="local">local</el-radio>
              <el-radio label="cubefs">cubefs</el-radio>
              <el-radio label="backend">已配置后端</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'local'" label="Dst 路径">
            <MountPathSelect
              v-model="form.dstPath"
              :options="mountPathOptions"
              :disabled="isViewMode"
              placeholder="/data/dest/  （选择已挂载路径或自行输入）"
              hint="候选来自 syncnode 实际容器内的挂载点；选中后仍可继续编辑或清空。"
            />
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'cubefs'" label="卷名">
            <el-select v-model="form.dstVolume" filterable clearable placeholder="选择或输入卷名" style="width: 100%;" :disabled="isViewMode">
              <el-option v-for="v in volumes" :key="v.name" :label="v.name" :value="v.name"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'cubefs'" label="Dst 路径">
            <el-input v-model.trim="form.dstPrefix" placeholder="/ （留空默认根目录）" clearable :disabled="isViewMode"></el-input>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'backend'" label="Dst 后端">
            <el-select v-model="form.dstBackendId" clearable placeholder="选择存储后端" style="width: 100%;" :disabled="isViewMode">
              <el-option
                v-for="b in backends"
                :key="b.id"
                :label="`[${b.kind}] ${b.name} (${b.bucket})`"
                :value="b.id"
              ></el-option>
            </el-select>
          </el-form-item>
          <el-form-item v-if="form.dstKind === 'backend'" label="Dst 前缀">
            <el-input v-model.trim="form.dstBackendPrefix" placeholder="可选，覆盖路径前缀" clearable :disabled="isViewMode"></el-input>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="JSON" name="json">
        <div v-if="!isViewMode" class="json-toolbar">
          <el-button size="small" type="primary" plain @click="loadJsonIntoForm">加载到表单</el-button>
          <span class="json-hint">编辑 JSON 后点击"加载到表单"可同步回基础信息页</span>
        </div>
        <el-input
          v-model="editorValue"
          type="textarea"
          :rows="20"
          resize="none"
          placeholder="输入规则 JSON"
          :disabled="isViewMode"
        ></el-input>
      </el-tab-pane>
    </el-tabs>

    <div slot="footer">
      <el-button @click="handleClose">{{ isViewMode ? $t('button.close') : $t('button.cancel') }}</el-button>
      <el-button v-if="!isViewMode" type="primary" :loading="submitting" @click="handleConfirm">{{ $t('button.submit') }}</el-button>
    </div>
  </el-dialog>
</template>

<script>
import { getSyncStorageBackendConfig, getSyncStorageBackendList, getSyncNodeList, getVolList } from '@/api/cfs/cluster'
import MountPathSelect from '@/components/MountPathSelect.vue'

const emptyForm = () => ({
  id: '',
  type: 'sync',
  scheduleMode: 'cron',
  schedule: '',
  parallelism: 3,
  // sync-specific
  afterCopy: '',
  // load-specific
  downloadStrategy: '',
  // check-specific
  onMismatch: '',
  sampleStrategy: '',
  sampleRate: 0.5,
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
  components: { MountPathSelect },
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    clusterName: {
      type: String,
      required: true,
    },
    mode: {
      type: String,
      default: 'create',
    },
    config: {
      type: Object,
      default: null,
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
      // populated by loadMountPaths(); shared between Src/Dst MountPathSelect
      mountPathOptions: { cubefs: [], external: [] },
      submitting: false,
    }
  },
  computed: {
    isEditMode() {
      return this.mode === 'edit'
    },
    isViewMode() {
      return this.mode === 'view'
    },
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
      // type-specific fields
      if (this.form.type === 'sync' && this.form.afterCopy) {
        payload.afterCopy = this.form.afterCopy
      }
      if (this.form.type === 'load' && this.form.downloadStrategy) {
        payload.downloadStrategy = this.form.downloadStrategy
      }
      if (this.form.type === 'check') {
        if (this.form.onMismatch) payload.onMismatch = this.form.onMismatch
        if (this.form.sampleStrategy) payload.sampleStrategy = this.form.sampleStrategy
        if (this.form.sampleStrategy && this.form.sampleStrategy !== 'full' && this.form.sampleRate > 0) {
          payload.sampleRate = this.form.sampleRate
        }
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
      if (!this.isEditMode && !this.isViewMode) {
        this.form = emptyForm()
        this.editorValue = ''
        this.activeTab = 'form'
      }
      await Promise.all([this.loadBackends(), this.loadVolumes(), this.loadMountPaths()])
      if ((this.isEditMode || this.isViewMode) && this.config) {
        this.editorValue = JSON.stringify(this.config, null, 2)
        this.fillFormFromConfig(this.config)
      }
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
    // 与测试管理 BenchRuleCreateDialog 保持一致：从所有在线 syncnode 拉取容器内
    // 实际挂载点，按 (path, fsType) 去重，fuse.cubefs 进 "CubeFS 卷" 分组，
    // 其他（本地 / GPFS / NFS 等）进 "本地 / GPFS / 其他" 分组。
    async loadMountPaths() {
      try {
        const { data } = await getSyncNodeList({ cluster_name: this.clusterName })
        const nodes = Array.isArray(data) ? data : (data?.data || [])
        const seen = new Set()
        const cubefs = []
        const external = []
        for (const n of nodes) {
          const mps = (n && n.mountPoints) || []
          for (const m of mps) {
            if (!m || !m.path) continue
            const key = m.path + '|' + (m.fsType || '')
            if (seen.has(key)) continue
            seen.add(key)
            const entry = { path: m.path, fsType: m.fsType || '' }
            if ((m.fsType || '').toLowerCase().startsWith('fuse.cubefs')) {
              cubefs.push(entry)
            } else {
              external.push(entry)
            }
          }
        }
        cubefs.sort((a, b) => a.path.localeCompare(b.path))
        external.sort((a, b) => a.path.localeCompare(b.path))
        this.mountPathOptions = { cubefs, external }
      } catch (_) {
        this.mountPathOptions = { cubefs: [], external: [] }
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
        const prefix = this.form[`${side}Prefix`]
        return { kind: 'cfs', vol: volume, path: prefix || '/' }
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
    loadJsonIntoForm() {
      let config
      try {
        config = JSON.parse(this.editorValue)
      } catch (_) {
        this.$message.error('JSON 格式错误，请检查后重试')
        return
      }
      const errors = []
      if (!config.id) errors.push('缺少 id 字段')
      if (!config.src) errors.push('缺少 src 配置')
      if (!config.dst) errors.push('缺少 dst 配置')
      if (errors.length) {
        this.$message.warning(errors.join('；'))
        return
      }
      this.fillFormFromConfig(config)
      this.activeTab = 'form'
      this.$message.success('已加载到表单，请检查各字段')
    },
    fillFormFromConfig(config) {
      if (!config) return
      this.form.id = config.id || ''
      this.form.type = config.type || 'sync'
      if (config.schedule) {
        this.form.scheduleMode = 'cron'
        this.form.schedule = config.schedule
      } else {
        this.form.scheduleMode = 'once'
        this.form.schedule = ''
      }
      this.form.parallelism = config.parallelism || 3
      this.form.shardingStrategy = config.shardingStrategy || ''
      this.form.shardPrefixesText = (config.shardPrefixes || []).join('\n')
      // type-specific fields
      this.form.afterCopy = config.afterCopy || ''
      this.form.downloadStrategy = config.downloadStrategy || ''
      this.form.onMismatch = config.onMismatch || ''
      this.form.sampleStrategy = config.sampleStrategy || ''
      this.form.sampleRate = config.sampleRate || 0.5
      if (config.src) this.parseSideConfig('src', config.src)
      if (config.dst) this.parseSideConfig('dst', config.dst)
    },
    parseSideConfig(side, cfg) {
      if (!cfg) return
      if (cfg.kind === 'local') {
        this.form[`${side}Kind`] = 'local'
        this.form[`${side}Path`] = cfg.path || ''
      } else if (cfg.kind === 'cfs' || cfg.kind === 'cfs-sync') {
        // "cfs" is the canonical kind; "cfs-sync" was a past frontend bug — handle both
        this.form[`${side}Kind`] = 'cubefs'
        this.form[`${side}Volume`] = cfg.vol || cfg.volume || ''
        this.form[`${side}Prefix`] = cfg.path || cfg.prefix || ''
      } else if (cfg.kind) {
        this.form[`${side}Kind`] = 'backend'
        this.form[`${side}BackendPrefix`] = cfg.prefix || ''
        const match = this.backends.find(b => b.kind === cfg.kind && b.bucket === cfg.bucket)
        this.form[`${side}BackendId`] = match ? match.id : null
      }
    },
    async handleConfirm() {
      const runImmediately = !this.isEditMode && this.form.scheduleMode === 'once'
      if (this.activeTab === 'form') {
        if (!this.form.id) {
          this.$message.warning('Rule ID 不能为空')
          return
        }
        const srcBroken = this.form.srcKind === 'backend' && !this.form.srcBackendId
        const dstBroken = this.form.dstKind === 'backend' && !this.form.dstBackendId
        if (srcBroken || dstBroken) {
          this.$message.warning('存在无法匹配的存储后端配置，请切换到 JSON 标签页手动编辑')
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

.json-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.json-hint {
  font-size: 12px;
  color: #9ca3af;
}

::v-deep .el-divider__text {
  font-size: 12px;
  color: #6b7280;
}
</style>
