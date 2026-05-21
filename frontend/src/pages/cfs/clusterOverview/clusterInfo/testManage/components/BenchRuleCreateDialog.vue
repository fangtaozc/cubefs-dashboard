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
    :title="isEditMode ? $t('bench.editbenchrule') : $t('bench.createbenchrule')"
    width="1120px"
    top="6vh"
    append-to-body
    custom-class="bench-dialog"
    @open="onOpen"
    @close="handleClose"
  >
    <el-form
      ref="form"
      :model="form"
      :rules="rules"
      size="small"
      label-position="top"
    >
      <!-- ============ 基础信息 ============ -->
      <section class="section">
        <header class="section-title">基础信息</header>
        <el-row :gutter="16">
          <el-col v-if="isEditMode" :span="6">
            <el-form-item :label="$t('bench.ruleid')" prop="id">
              <el-input v-model.trim="form.id" disabled></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="isEditMode ? 6 : 8">
            <el-form-item :label="$t('bench.rulename')" prop="name">
              <el-input v-model.trim="form.name" clearable placeholder="规则名称"></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item :label="$t('bench.storagetype')" prop="storageType">
              <el-select v-model="form.storageType" style="width: 100%;">
                <el-option label="S3 / 对象存储" value="s3"></el-option>
                <el-option label="POSIX / 本地或挂载路径" value="posix"></el-option>
                <el-option label="mdtest / 元数据压测 (MPI)" value="mdtest"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="6">
            <el-form-item :label="$t('bench.parallelism')" prop="parallelism">
              <el-input-number
                v-model="form.parallelism"
                :min="1"
                :max="128"
                :precision="0"
                style="width: 100%;"
              ></el-input-number>
            </el-form-item>
          </el-col>
        </el-row>
      </section>

      <!-- ============ S3 ============ -->
      <template v-if="form.storageType === 's3'">
        <section class="section">
          <header class="section-title">S3 对象存储</header>
          <el-row :gutter="16">
            <el-col :span="16">
              <el-form-item :label="$t('bench.backendid')" prop="backendID">
                <el-select
                  v-model="form.backendID"
                  clearable
                  filterable
                  placeholder="从同步管理已配置的存储后端中选择"
                  style="width: 100%;"
                >
                  <el-option
                    v-for="b in backends"
                    :key="b.id"
                    :label="`[${b.kind}] ${b.name} (${b.bucket})`"
                    :value="b.id"
                  ></el-option>
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="8">
              <el-form-item :label="$t('bench.keyprefix')">
                <el-input v-model.trim="form.keyPrefix" clearable placeholder="例如 benchmark/2026/"></el-input>
              </el-form-item>
            </el-col>
          </el-row>
        </section>

        <section class="section">
          <header class="section-title">
            <span>阶段配置 <small>（按顺序执行，共 {{ form.stages.length }} 个阶段）</small></span>
            <el-button-group>
              <el-button size="mini" plain icon="el-icon-document-copy" @click="loadS3Preset">加载示例</el-button>
              <el-button size="mini" plain icon="el-icon-plus" type="primary" @click="addStage('s3')">添加阶段</el-button>
            </el-button-group>
          </header>
          <div v-if="form.stages.length === 0" class="empty-tip">
            尚未配置阶段，请点上方"添加阶段"。
          </div>
          <div
            v-for="(stage, idx) in form.stages"
            :key="`s3-stage-${idx}`"
            class="stage"
            :class="{ 'is-expanded': stage._expanded }"
          >
            <header class="stage-head" @click="toggleStage(stage)">
              <i :class="stage._expanded ? 'el-icon-arrow-down' : 'el-icon-arrow-right'"></i>
              <span class="stage-name">#{{ idx + 1 }} {{ stage.name || '未命名阶段' }}</span>
              <span class="stage-summary">{{ s3StageSummary(stage) }}</span>
              <el-button type="text" class="stage-del" @click.stop="removeStage(idx)">删除</el-button>
            </header>
            <div v-show="stage._expanded" class="stage-body">
              <el-row :gutter="16">
                <el-col :span="8">
                  <el-form-item label="名称">
                    <el-input v-model.trim="stage.name" placeholder="如 write / read / mixed"></el-input>
                  </el-form-item>
                </el-col>
                <el-col :span="4">
                  <el-form-item label="并发">
                    <el-input-number v-model="stage.numjobs" :min="1" :max="1024" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="4">
                  <el-form-item label="运行(秒)">
                    <el-input-number v-model="stage.runtime" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="4">
                  <el-form-item label="对象数">
                    <el-input-number v-model="stage.numObjects" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="4">
                  <el-form-item label="结束清空">
                    <el-switch v-model="stage.deleteAll"></el-switch>
                  </el-form-item>
                </el-col>
                <el-col :span="24">
                  <el-form-item label="对象大小">
                    <el-radio-group v-model="stage.objectSize.mode" size="mini">
                      <el-radio-button label="fixed">固定</el-radio-button>
                      <el-radio-button label="range">区间分布</el-radio-button>
                    </el-radio-group>
                    <el-row v-if="stage.objectSize.mode === 'fixed'" :gutter="8" style="margin-top: 8px;">
                      <el-col :span="8">
                        <el-input-number v-model="stage.objectSize.fixedValue" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                      </el-col>
                      <el-col :span="4">
                        <el-select v-model="stage.objectSize.fixedUnit" style="width: 100%;">
                          <el-option v-for="u in SIZE_UNITS" :key="u.label" :label="u.label" :value="u.value"></el-option>
                        </el-select>
                      </el-col>
                    </el-row>
                    <el-row v-else :gutter="8" style="margin-top: 8px;">
                      <el-col :span="5"><el-input-number v-model="stage.objectSize.minValue" :min="0" :precision="0" placeholder="min" style="width: 100%;"></el-input-number></el-col>
                      <el-col :span="3"><el-select v-model="stage.objectSize.minUnit" style="width: 100%;"><el-option v-for="u in SIZE_UNITS" :key="u.label" :label="u.label" :value="u.value"></el-option></el-select></el-col>
                      <el-col :span="5"><el-input-number v-model="stage.objectSize.maxValue" :min="0" :precision="0" placeholder="max" style="width: 100%;"></el-input-number></el-col>
                      <el-col :span="3"><el-select v-model="stage.objectSize.maxUnit" style="width: 100%;"><el-option v-for="u in SIZE_UNITS" :key="u.label" :label="u.label" :value="u.value"></el-option></el-select></el-col>
                      <el-col :span="4"><el-select v-model="stage.objectSize.dist" style="width: 100%;"><el-option v-for="d in DIST_OPTIONS" :key="d.value" :label="d.label" :value="d.value"></el-option></el-select></el-col>
                    </el-row>
                  </el-form-item>
                </el-col>
                <el-col :span="24">
                  <el-form-item label="操作组合">
                    <div v-for="(op, oi) in stage.ops" :key="`op-${idx}-${oi}`" class="op-row">
                      <el-select v-model="op.type" style="width: 140px;">
                        <el-option v-for="t in OP_TYPES" :key="t" :label="t" :value="t"></el-option>
                      </el-select>
                      <span class="op-sep">权重</span>
                      <el-input-number v-model="op.weight" :min="1" :max="100" :precision="0" style="width: 120px;"></el-input-number>
                      <el-button type="text" class="op-del" @click="removeOp(idx, oi)">移除</el-button>
                    </div>
                    <el-button size="mini" plain icon="el-icon-plus" @click="addOp(idx)">添加操作</el-button>
                  </el-form-item>
                </el-col>
              </el-row>
            </div>
          </div>
        </section>
      </template>

      <!-- ============ POSIX ============ -->
      <template v-if="form.storageType === 'posix'">
        <section class="section">
          <header class="section-title">POSIX 路径</header>
          <el-row :gutter="16">
            <el-col :span="24">
              <el-form-item :label="$t('bench.mountpath')" prop="mountPath">
                <el-select
                  v-model="form.mountPath"
                  filterable
                  allow-create
                  default-first-option
                  clearable
                  style="width: 100%"
                  placeholder="选择已挂载路径，或自行输入"
                >
                  <el-option-group v-if="mountPathOptions.cubefs.length" label="CubeFS 卷">
                    <el-option
                      v-for="m in mountPathOptions.cubefs"
                      :key="'cfs-' + m.path"
                      :label="m.path"
                      :value="m.path"
                    >
                      <span>{{ m.path }}</span>
                      <span style="float:right;color:#909399;font-size:12px;">{{ m.fsType }}</span>
                    </el-option>
                  </el-option-group>
                  <el-option-group v-if="mountPathOptions.external.length" label="本地 / GPFS / 其他">
                    <el-option
                      v-for="m in mountPathOptions.external"
                      :key="'ext-' + m.path"
                      :label="m.path"
                      :value="m.path"
                    >
                      <span>{{ m.path }}</span>
                      <span style="float:right;color:#909399;font-size:12px;">{{ m.fsType || '未知' }}</span>
                    </el-option>
                  </el-option-group>
                </el-select>
                <div class="form-hint">候选来自 syncnode 实际容器内的挂载点；未在列表中的路径可直接键入。</div>
              </el-form-item>
            </el-col>
          </el-row>
        </section>

        <section class="section">
          <header class="section-title">FIO 默认参数 <small>（被各阶段覆盖前的默认值）</small></header>
          <el-row :gutter="16">
            <el-col :span="6">
              <el-form-item label="ioengine">
                <el-select v-model="form.fioDefaults.ioengine" clearable style="width: 100%;">
                  <el-option v-for="e in IOENGINES" :key="e" :label="e" :value="e"></el-option>
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="3">
              <el-form-item label="iodepth">
                <el-input-number v-model="form.fioDefaults.iodepth" :min="0" :precision="0" style="width: 100%;"></el-input-number>
              </el-form-item>
            </el-col>
            <el-col :span="3">
              <el-form-item label="numjobs">
                <el-input-number v-model="form.fioDefaults.numjobs" :min="0" :precision="0" style="width: 100%;"></el-input-number>
              </el-form-item>
            </el-col>
            <el-col :span="3">
              <el-form-item label="bs">
                <el-input v-model.trim="form.fioDefaults.bs" placeholder="4k/1m"></el-input>
              </el-form-item>
            </el-col>
            <el-col :span="3">
              <el-form-item label="size">
                <el-input v-model.trim="form.fioDefaults.size" placeholder="1g"></el-input>
              </el-form-item>
            </el-col>
            <el-col :span="3">
              <el-form-item label="runtime(秒)">
                <el-input-number v-model="form.fioDefaults.runtime" :min="0" :precision="0" style="width: 100%;"></el-input-number>
              </el-form-item>
            </el-col>
            <el-col :span="3">
              <el-form-item label="direct">
                <el-radio-group v-model="form.fioDefaults.direct">
                  <el-radio-button :label="0">关</el-radio-button>
                  <el-radio-button :label="1">开</el-radio-button>
                </el-radio-group>
              </el-form-item>
            </el-col>
            <el-col :span="18">
              <el-form-item label="额外参数 extraArgs">
                <el-input v-model.trim="form.fioDefaults.extraArgs" placeholder="原样透传给 fio，空格分隔"></el-input>
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="清理">
                <el-checkbox v-model="form.fioDefaults.cleanupAfterDone">完成后清理工作目录</el-checkbox>
              </el-form-item>
            </el-col>
          </el-row>
        </section>

        <section class="section">
          <header class="section-title">
            <span>阶段配置 <small>（按顺序执行，共 {{ form.fioStages.length }} 个阶段）</small></span>
            <el-button-group>
              <el-button size="mini" plain icon="el-icon-document-copy" @click="loadFioPreset">加载示例 (6 项 fio 标准压测)</el-button>
              <el-button size="mini" plain icon="el-icon-plus" type="primary" @click="addStage('posix')">添加阶段</el-button>
            </el-button-group>
          </header>
          <div v-if="form.fioStages.length === 0" class="empty-tip">
            尚未配置阶段，请点上方"添加阶段"或"加载示例"。
          </div>
          <div
            v-for="(stage, idx) in form.fioStages"
            :key="`posix-stage-${idx}`"
            class="stage"
            :class="{ 'is-expanded': stage._expanded, 'is-skip': stage.skip }"
          >
            <header class="stage-head" @click="toggleFioStage(stage)">
              <i :class="stage._expanded ? 'el-icon-arrow-down' : 'el-icon-arrow-right'"></i>
              <span class="stage-name">#{{ idx + 1 }} {{ stage.name || '未命名阶段' }}</span>
              <span class="stage-summary">{{ fioStageSummary(stage) }}</span>
              <el-checkbox v-model="stage.skip" class="stage-skip" @click.native.stop>跳过</el-checkbox>
              <el-button type="text" class="stage-del" @click.stop="removeFioStage(idx)">删除</el-button>
            </header>
            <div v-show="stage._expanded" class="stage-body">
              <el-row :gutter="16">
                <el-col :span="6">
                  <el-form-item label="名称">
                    <el-input v-model.trim="stage.name" placeholder="如 seq-write"></el-input>
                  </el-form-item>
                </el-col>
                <el-col :span="4">
                  <el-form-item label="IO 模式 (rw)">
                    <el-select v-model="stage.rw" style="width: 100%;" @change="onStageRwChange(stage)">
                      <el-option v-for="r in RW_MODES" :key="r" :label="r" :value="r"></el-option>
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="bs(覆盖)">
                    <el-input v-model.trim="stage.bs" placeholder="留空用默认"></el-input>
                  </el-form-item>
                </el-col>
                <el-col v-if="isMixedRW(stage.rw)" :span="4">
                  <el-form-item label="读取占比%">
                    <el-input-number v-model="stage.rwmixread" :min="0" :max="100" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="iodepth">
                    <el-input-number v-model="stage.iodepth" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="numjobs">
                    <el-input-number v-model="stage.numjobs" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="size">
                    <el-input v-model.trim="stage.size" placeholder="留空用默认"></el-input>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="runtime(秒)">
                    <el-input-number v-model="stage.runtime" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="direct">
                    <el-select v-model="stage.direct" clearable placeholder="跟随默认" style="width: 100%;">
                      <el-option label="关" :value="0"></el-option>
                      <el-option label="开" :value="1"></el-option>
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="9">
                  <el-form-item label="复用源阶段">
                    <el-select v-model="stage.sourceStage" clearable placeholder="选择前序写入阶段" style="width: 100%;" @change="onSourceStageChange(stage)">
                      <el-option
                        v-for="s in form.fioStages.filter((it, i) => i < idx && it.name)"
                        :key="s.name"
                        :label="s.name"
                        :value="s.name"
                      ></el-option>
                    </el-select>
                  </el-form-item>
                </el-col>
                <el-col :span="6">
                  <el-form-item label="复用文件">
                    <el-switch v-model="stage.reuseFiles" active-text="不重新生成"></el-switch>
                  </el-form-item>
                </el-col>
              </el-row>
            </div>
          </div>
        </section>
      </template>

      <!-- ============ mdtest (MPI metadata) ============ -->
      <template v-if="form.storageType === 'mdtest'">
        <section class="section">
          <header class="section-title">mdtest 工作目录 <small>（共享文件系统挂载路径，所有 shard 在该路径下各自建子目录）</small></header>
          <el-row :gutter="16">
            <el-col :span="24">
              <el-form-item :label="$t('bench.mountpath')" prop="mountPath">
                <el-select
                  v-model="form.mountPath"
                  filterable
                  allow-create
                  default-first-option
                  clearable
                  style="width: 100%"
                  placeholder="选择已挂载路径，或自行输入"
                >
                  <el-option-group v-if="mountPathOptions.cubefs.length" label="CubeFS 卷">
                    <el-option
                      v-for="m in mountPathOptions.cubefs"
                      :key="'cfs-' + m.path"
                      :label="m.path"
                      :value="m.path"
                    >
                      <span>{{ m.path }}</span>
                      <span style="float:right;color:#909399;font-size:12px;">{{ m.fsType }}</span>
                    </el-option>
                  </el-option-group>
                  <el-option-group v-if="mountPathOptions.external.length" label="本地 / GPFS / 其他">
                    <el-option
                      v-for="m in mountPathOptions.external"
                      :key="'ext-' + m.path"
                      :label="m.path"
                      :value="m.path"
                    >
                      <span>{{ m.path }}</span>
                      <span style="float:right;color:#909399;font-size:12px;">{{ m.fsType || '未知' }}</span>
                    </el-option>
                  </el-option-group>
                </el-select>
                <div class="form-hint">mdtest 通过 mpirun 在多个 shard 间分发，所有 shard 必须看到同一份共享 FS。</div>
              </el-form-item>
            </el-col>
          </el-row>
        </section>

        <section class="section">
          <header class="section-title">mdtest 默认参数</header>
          <el-row :gutter="16">
            <el-col :span="6">
              <el-form-item label="mpiBin">
                <el-input v-model.trim="form.mdtestDefaults.mpiBin" placeholder="默认 mpirun"></el-input>
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="mdtestBin">
                <el-input v-model.trim="form.mdtestDefaults.mdtestBin" placeholder="默认 mdtest"></el-input>
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="numTasks (MPI 进程数)">
                <el-input-number v-model="form.mdtestDefaults.numTasks" :min="1" :max="1024" :precision="0" style="width: 100%;"></el-input-number>
              </el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="extraArgs">
                <el-input v-model.trim="form.mdtestDefaults.extraArgs" placeholder="原样透传给 mdtest"></el-input>
              </el-form-item>
            </el-col>
          </el-row>
        </section>

        <section class="section">
          <header class="section-title">
            <span>阶段配置 <small>（按顺序执行，共 {{ form.mdtestStages.length }} 个阶段）</small></span>
            <el-button-group>
              <el-button size="mini" plain icon="el-icon-document-copy" @click="loadMdtestPreset">加载示例</el-button>
              <el-button size="mini" plain icon="el-icon-plus" type="primary" @click="addStage('mdtest')">添加阶段</el-button>
            </el-button-group>
          </header>
          <div v-if="form.mdtestStages.length === 0" class="empty-tip">
            尚未配置阶段，请点上方"添加阶段"。
          </div>
          <div
            v-for="(stage, idx) in form.mdtestStages"
            :key="`md-stage-${idx}`"
            class="stage"
            :class="{ 'is-expanded': stage._expanded, 'is-skip': stage.skip }"
          >
            <header class="stage-head" @click="toggleMdtestStage(stage)">
              <i :class="stage._expanded ? 'el-icon-arrow-down' : 'el-icon-arrow-right'"></i>
              <span class="stage-name">#{{ idx + 1 }} {{ stage.name || '未命名阶段' }}</span>
              <span class="stage-summary">{{ mdtestStageSummary(stage) }}</span>
              <el-checkbox v-model="stage.skip" class="stage-skip" @click.native.stop>跳过</el-checkbox>
              <el-button type="text" class="stage-del" @click.stop="removeMdtestStage(idx)">删除</el-button>
            </header>
            <div v-show="stage._expanded" class="stage-body">
              <el-row :gutter="16">
                <el-col :span="6">
                  <el-form-item label="名称">
                    <el-input v-model.trim="stage.name" placeholder="如 small-files / dir-tree"></el-input>
                  </el-form-item>
                </el-col>
                <el-col :span="6">
                  <el-form-item label="类型">
                    <el-radio-group v-model="stage._kind" size="mini" @change="onMdtestKindChange(stage)">
                      <el-radio-button label="both">文件+目录</el-radio-button>
                      <el-radio-button label="files">仅文件</el-radio-button>
                      <el-radio-button label="dirs">仅目录</el-radio-button>
                    </el-radio-group>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="iterations">
                    <el-input-number v-model="stage.iterations" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="numItems">
                    <el-input-number v-model="stage.numItems" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="numTasks">
                    <el-input-number v-model="stage.numTasks" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="uniqueDir">
                    <el-switch v-model="stage.uniqueDir"></el-switch>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="itemsPerDir">
                    <el-input-number v-model="stage.itemsPerDir" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="depth">
                    <el-input-number v-model="stage.depth" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="3">
                  <el-form-item label="branching">
                    <el-input-number v-model="stage.branching" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="6">
                  <el-form-item label="文件写入字节数">
                    <el-input-number v-model="stage.writeBytes" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="6">
                  <el-form-item label="文件读取字节数">
                    <el-input-number v-model="stage.readBytes" :min="0" :precision="0" style="width: 100%;"></el-input-number>
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="extraArgs">
                    <el-input v-model.trim="stage.extraArgs" placeholder="额外参数透传给 mdtest"></el-input>
                  </el-form-item>
                </el-col>
              </el-row>
            </div>
          </div>
        </section>
      </template>
    </el-form>
    <span slot="footer">
      <el-button size="small" @click="handleClose">{{ $t('common.cancel') }}</el-button>
      <el-button size="small" type="primary" :loading="submitting" @click="handleSubmit">{{ $t('common.confirm') }}</el-button>
    </span>
  </el-dialog>
</template>

<script>
import { createBenchRule, updateBenchRule, getSyncStorageBackendList, getSyncNodeList } from '@/api/cfs/cluster'

const OP_TYPES = ['put', 'get', 'delete', 'head', 'list']
const DIST_OPTIONS = [
  { label: '均匀分布', value: 'uniform' },
  { label: '帕累托分布', value: 'pareto' },
]
const SIZE_UNITS = [
  { label: 'B', value: 1 },
  { label: 'KB', value: 1024 },
  { label: 'MB', value: 1024 * 1024 },
  { label: 'GB', value: 1024 * 1024 * 1024 },
]
const IOENGINES = ['libaio', 'posixaio', 'psync', 'sync', 'mmap']
const RW_MODES = ['read', 'write', 'randread', 'randwrite', 'rw', 'randrw']

function emptyS3Stage(expanded = false) {
  return {
    _expanded: expanded,
    name: '',
    deleteAll: false,
    numjobs: 4,
    runtime: 30,
    numObjects: 0,
    objectSize: {
      mode: 'fixed',
      fixedValue: 4,
      fixedUnit: 1024,
      minValue: 0,
      minUnit: 1024,
      maxValue: 0,
      maxUnit: 1024,
      dist: 'uniform',
    },
    ops: [{ type: 'put', weight: 1 }],
  }
}

function emptyFioStage(expanded = false) {
  return {
    _expanded: expanded,
    name: '',
    rw: 'write',
    bs: '',
    rwmixread: 50,
    reuseFiles: false,
    sourceStage: '',
    skip: false,
    iodepth: 0,
    numjobs: 0,
    size: '',
    runtime: 0,
    direct: null,
  }
}

function emptyFioDefaults() {
  return {
    ioengine: 'libaio',
    iodepth: 0,
    numjobs: 0,
    bs: '',
    size: '',
    runtime: 0,
    direct: 1,
    extraArgs: '',
    cleanupAfterDone: false,
  }
}

function emptyMdtestDefaults() {
  return {
    mpiBin: '',
    mdtestBin: '',
    numTasks: 4,
    extraArgs: '',
  }
}

function emptyMdtestStage(expanded = false) {
  return {
    _expanded: expanded,
    _kind: 'both',
    name: '',
    skip: false,
    iterations: 1,
    numItems: 1000,
    itemsPerDir: 0,
    depth: 0,
    branching: 0,
    writeBytes: 0,
    readBytes: 0,
    onlyFiles: false,
    onlyDirs: false,
    uniqueDir: false,
    numTasks: 0,
    extraArgs: '',
  }
}

function mdtestStandardPreset() {
  const mk = (over) => {
    const s = { ...emptyMdtestStage(), ...over }
    s._kind = s.onlyFiles ? 'files' : (s.onlyDirs ? 'dirs' : 'both')
    return s
  }
  return [
    mk({ name: 'small-file-stress', onlyFiles: true, iterations: 3, numItems: 5000, writeBytes: 4096 }),
    mk({ name: 'dir-tree-stress', onlyDirs: true, iterations: 1, depth: 3, branching: 5, itemsPerDir: 100 }),
    mk({ name: 'mixed', iterations: 2, numItems: 1000, writeBytes: 16384 }),
  ]
}

// === fio 6 项标准压测预设（randwrite IOPS / randread IOPS / 顺序写带宽 /
// 顺序读带宽 / 随机写延迟 / 随机读延迟）— 与官方文档一致 ===
function fioStandardPreset() {
  const mk = (over) => ({ ...emptyFioStage(), ...over })
  return [
    mk({ name: 'rand-write-iops', rw: 'randwrite', bs: '4k', iodepth: 128, numjobs: 16, size: '100g', runtime: 1000 }),
    mk({ name: 'rand-read-iops',  rw: 'randread',  bs: '4k', iodepth: 128, numjobs: 16, size: '100g', runtime: 1000 }),
    mk({ name: 'seq-write-bw',    rw: 'write',     bs: '1m', iodepth: 64,  numjobs: 8,  size: '300g', runtime: 1000 }),
    mk({ name: 'seq-read-bw',     rw: 'read',      bs: '1m', iodepth: 64,  numjobs: 8,  size: '300g', runtime: 1000 }),
    mk({ name: 'rand-write-lat',  rw: 'randwrite', bs: '4k', iodepth: 4,   numjobs: 1,  size: '10g',  runtime: 0 }),
    mk({ name: 'rand-read-lat',   rw: 'randread',  bs: '4k', iodepth: 4,   numjobs: 1,  size: '10g',  runtime: 0 }),
  ]
}

function s3StandardPreset() {
  const mk = (over) => ({ ...emptyS3Stage(), ...over })
  return [
    mk({
      name: 'put-4k',
      ops: [{ type: 'put', weight: 1 }],
      numjobs: 16,
      runtime: 60,
      numObjects: 10000,
      objectSize: { mode: 'fixed', fixedValue: 4, fixedUnit: 1024, minValue: 0, minUnit: 1024, maxValue: 0, maxUnit: 1024, dist: 'uniform' },
    }),
    mk({
      name: 'get-4k',
      ops: [{ type: 'get', weight: 1 }],
      numjobs: 16,
      runtime: 60,
      numObjects: 10000,
      objectSize: { mode: 'fixed', fixedValue: 4, fixedUnit: 1024, minValue: 0, minUnit: 1024, maxValue: 0, maxUnit: 1024, dist: 'uniform' },
    }),
    mk({
      name: 'mixed-9put-1get-1m',
      ops: [{ type: 'put', weight: 9 }, { type: 'get', weight: 1 }],
      numjobs: 16,
      runtime: 120,
      numObjects: 1000,
      objectSize: { mode: 'fixed', fixedValue: 1, fixedUnit: 1024 * 1024, minValue: 0, minUnit: 1024, maxValue: 0, maxUnit: 1024, dist: 'uniform' },
    }),
  ]
}

function packObjSize(os) {
  if (os.mode === 'fixed') {
    const v = Number(os.fixedValue || 0) * Number(os.fixedUnit || 1)
    return v > 0 ? { fixed: v, dist: 'fixed' } : null
  }
  const min = Number(os.minValue || 0) * Number(os.minUnit || 1)
  const max = Number(os.maxValue || 0) * Number(os.maxUnit || 1)
  if (min <= 0 || max <= 0) return null
  return { min, max, dist: os.dist || 'uniform' }
}

function unpackObjSize(raw) {
  const base = { mode: 'fixed', fixedValue: 4, fixedUnit: 1024, minValue: 0, minUnit: 1024, maxValue: 0, maxUnit: 1024, dist: 'uniform' }
  if (!raw) return base
  if (raw.fixed) {
    const { value, unit } = pickUnit(raw.fixed)
    return { ...base, mode: 'fixed', fixedValue: value, fixedUnit: unit }
  }
  if (raw.min || raw.max) {
    const lo = pickUnit(raw.min)
    const hi = pickUnit(raw.max)
    return { ...base, mode: 'range', minValue: lo.value, minUnit: lo.unit, maxValue: hi.value, maxUnit: hi.unit, dist: raw.dist || 'uniform' }
  }
  return base
}

function pickUnit(bytes) {
  const n = Number(bytes || 0)
  if (n === 0) return { value: 0, unit: 1024 }
  for (let i = SIZE_UNITS.length - 1; i >= 0; i--) {
    const u = SIZE_UNITS[i].value
    if (n % u === 0) return { value: n / u, unit: u }
  }
  return { value: n, unit: 1 }
}

function humanSize(bytes) {
  if (!bytes) return '0'
  const u = pickUnit(bytes)
  const label = SIZE_UNITS.find(x => x.value === u.unit)?.label || 'B'
  return `${u.value}${label}`
}

export default {
  name: 'BenchRuleCreateDialog',
  props: {
    visible: { type: Boolean, default: false },
    clusterName: { type: String, required: true },
    mode: { type: String, default: 'create' },
    row: { type: Object, default: null },
  },
  data() {
    return {
      submitting: false,
      backends: [],
      mountPathOptions: { cubefs: [], external: [] }, // populated via loadMountPaths
      OP_TYPES,
      DIST_OPTIONS,
      SIZE_UNITS,
      IOENGINES,
      RW_MODES,
      form: {
        id: '',
        name: '',
        storageType: 's3',
        parallelism: 4,
        backendID: '',
        keyPrefix: '',
        stages: [],
        mountPath: '',
        fioDefaults: emptyFioDefaults(),
        fioStages: [],
        mdtestDefaults: emptyMdtestDefaults(),
        mdtestStages: [],
      },
      rules: {
        name: [{ required: true, message: this.$t('bench.rulename'), trigger: 'blur' }],
        storageType: [{ required: true, trigger: 'change' }],
        backendID: [{
          required: true,
          trigger: 'change',
          validator: (rule, value, cb) => {
            if (this.form.storageType === 's3' && !value) {
              return cb(new Error(this.$t('bench.backendid')))
            }
            cb()
          },
        }],
        mountPath: [{
          required: true,
          trigger: 'blur',
          validator: (rule, value, cb) => {
            if ((this.form.storageType === 'posix' || this.form.storageType === 'mdtest') && !value) {
              return cb(new Error(this.$t('bench.mountpath')))
            }
            cb()
          },
        }],
      },
    }
  },
  computed: {
    innerVisible: {
      get() { return this.visible },
      set(val) { this.$emit('update:visible', val) },
    },
    isEditMode() {
      return this.mode === 'edit'
    },
  },
  methods: {
    onOpen() {
      this.loadBackends()
      this.loadMountPaths()
      if (this.isEditMode && this.row) {
        this.form.id = this.row.id || ''
        this.form.name = this.row.name || ''
        this.form.storageType = this.row.storageType || 's3'
        this.form.parallelism = this.row.parallelism || 4
        this.form.backendID = this.row.backendID || ''
        this.form.keyPrefix = this.row.keyPrefix || ''
        this.form.mountPath = this.row.mountPath || ''
        this.form.fioDefaults = { ...emptyFioDefaults(), ...(this.row.fioDefaults || {}) }
        this.form.stages = (this.row.stages || []).map(s => ({
          _expanded: false,
          name: s.name || '',
          deleteAll: !!s.deleteAll,
          numjobs: s.numjobs || 0,
          runtime: s.runtime || 0,
          numObjects: s.numObjects || 0,
          objectSize: unpackObjSize(s.objectSize),
          ops: (s.ops && s.ops.length > 0) ? s.ops.map(o => ({ type: o.type, weight: o.weight || 1 })) : [{ type: 'put', weight: 1 }],
        }))
        this.form.fioStages = (this.row.fioStages || []).map(s => ({ ...emptyFioStage(), ...s, _expanded: false }))
        this.form.mdtestDefaults = { ...emptyMdtestDefaults(), ...(this.row.mdtestDefaults || {}) }
        this.form.mdtestStages = (this.row.mdtestStages || []).map(s => {
          const merged = { ...emptyMdtestStage(), ...s, _expanded: false }
          merged._kind = merged.onlyFiles ? 'files' : (merged.onlyDirs ? 'dirs' : 'both')
          return merged
        })
      } else {
        this.resetForm()
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
    // Pull mount-path inventory from every alive syncnode so the
    // mountPath dropdown shows what's actually available. We dedup by
    // (path, fsType) so a 5-node cluster doesn't show 5 copies of the
    // same /cfs/posix-bench. fuse.cubefs entries land in the "CubeFS"
    // group; everything else goes to "本地" — matches the visual split
    // the user asked for. The control is `allow-create`, so an unknown
    // path can still be typed in by hand.
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
    // Stage UX guard: a stage that REads from another stage's files
    // (sourceStage set) MUST set reuseFiles=true — otherwise fio reuses
    // the current stage's name, finds no files, creates empty ones via
    // --create_on_open=1, and "reads" millions of zero-byte ops with no
    // bandwidth / latency. We auto-flip the toggle when the user picks
    // a source stage or switches RW to a read mode; users can still
    // un-toggle it manually if they really want the broken behaviour.
    onSourceStageChange(stage) {
      if (stage.sourceStage) {
        stage.reuseFiles = true
      }
    },
    onStageRwChange(stage) {
      if ((stage.rw === 'read' || stage.rw === 'randread') && stage.sourceStage) {
        stage.reuseFiles = true
      }
    },
    generateRuleId() {
      // Strip a leading "bench-" / "bench" the user may have typed in
      // the rule name — the system already prepends "bench-" to identify
      // bench-rule IDs, so without this dedup we'd produce
      // `bench-bench-smoke-test-001-…` for a rule named "bench smoke test 001".
      let slug = (this.form.name || '')
        .toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 30)
      slug = slug.replace(/^bench(-|$)/, '').replace(/^-+/, '')
      const ts = Date.now().toString(36)
      return slug ? `bench-${slug}-${ts}` : `bench-${ts}`
    },
    resetForm() {
      this.form = {
        id: '',
        name: '',
        storageType: 's3',
        parallelism: 4,
        backendID: '',
        keyPrefix: '',
        stages: [emptyS3Stage(true)],
        mountPath: '',
        fioDefaults: emptyFioDefaults(),
        fioStages: [emptyFioStage(true)],
        mdtestDefaults: emptyMdtestDefaults(),
        mdtestStages: [emptyMdtestStage(true)],
      }
      this.$nextTick(() => {
        this.$refs.form && this.$refs.form.clearValidate()
      })
    },
    handleClose() {
      this.innerVisible = false
      this.$emit('close')
    },
    toggleStage(stage) { this.$set(stage, '_expanded', !stage._expanded) },
    toggleFioStage(stage) { this.$set(stage, '_expanded', !stage._expanded) },
    toggleMdtestStage(stage) { this.$set(stage, '_expanded', !stage._expanded) },
    addStage(kind) {
      if (kind === 's3') this.form.stages.push(emptyS3Stage(true))
      else if (kind === 'mdtest') this.form.mdtestStages.push(emptyMdtestStage(true))
      else this.form.fioStages.push(emptyFioStage(true))
    },
    removeStage(idx) { this.form.stages.splice(idx, 1) },
    removeFioStage(idx) { this.form.fioStages.splice(idx, 1) },
    removeMdtestStage(idx) { this.form.mdtestStages.splice(idx, 1) },
    addOp(stageIdx) { this.form.stages[stageIdx].ops.push({ type: 'put', weight: 1 }) },
    removeOp(stageIdx, opIdx) { this.form.stages[stageIdx].ops.splice(opIdx, 1) },
    isMixedRW(rw) { return rw === 'rw' || rw === 'randrw' },
    onMdtestKindChange(stage) {
      stage.onlyFiles = stage._kind === 'files'
      stage.onlyDirs = stage._kind === 'dirs'
    },
    loadS3Preset() {
      this.form.stages = s3StandardPreset()
    },
    loadFioPreset() {
      this.form.fioStages = fioStandardPreset()
      this.form.fioDefaults.ioengine = this.form.fioDefaults.ioengine || 'libaio'
      if (this.form.fioDefaults.direct == null) this.form.fioDefaults.direct = 1
    },
    loadMdtestPreset() {
      this.form.mdtestStages = mdtestStandardPreset()
      if (!this.form.mdtestDefaults.numTasks) this.form.mdtestDefaults.numTasks = 4
    },
    s3StageSummary(s) {
      const ops = (s.ops || []).map(o => `${o.type}×${o.weight}`).join('+') || '无 op'
      const sz = s.objectSize?.mode === 'fixed'
        ? humanSize(Number(s.objectSize.fixedValue || 0) * Number(s.objectSize.fixedUnit || 1))
        : `${humanSize(Number(s.objectSize?.minValue || 0) * Number(s.objectSize?.minUnit || 1))}~${humanSize(Number(s.objectSize?.maxValue || 0) * Number(s.objectSize?.maxUnit || 1))}`
      return `${ops} · ${sz} · njobs=${s.numjobs} · runtime=${s.runtime}s`
    },
    fioStageSummary(s) {
      const parts = [s.rw || '?']
      if (s.bs) parts.push(`bs=${s.bs}`)
      if (s.iodepth) parts.push(`iod=${s.iodepth}`)
      if (s.numjobs) parts.push(`nj=${s.numjobs}`)
      if (s.size) parts.push(`size=${s.size}`)
      if (s.runtime) parts.push(`runtime=${s.runtime}s`)
      return parts.join(' · ')
    },
    mdtestStageSummary(s) {
      const parts = []
      parts.push(s._kind === 'files' ? '仅文件' : (s._kind === 'dirs' ? '仅目录' : '文件+目录'))
      if (s.iterations) parts.push(`i=${s.iterations}`)
      if (s.numItems) parts.push(`n=${s.numItems}`)
      if (s.depth) parts.push(`z=${s.depth}`)
      if (s.branching) parts.push(`b=${s.branching}`)
      if (s.writeBytes) parts.push(`w=${s.writeBytes}`)
      if (s.numTasks) parts.push(`tasks=${s.numTasks}`)
      return parts.join(' · ')
    },
    validateStages() {
      if (this.form.storageType === 's3') {
        if (this.form.stages.length === 0) { this.$message.error('至少配置一个 S3 阶段'); return false }
        for (const [i, s] of this.form.stages.entries()) {
          if (!s.name) { this.$message.error(`阶段 #${i + 1} 缺少名称`); s._expanded = true; return false }
          if (!s.ops || s.ops.length === 0) { this.$message.error(`阶段 #${i + 1} 至少需要一个操作`); s._expanded = true; return false }
          if (!packObjSize(s.objectSize)) { this.$message.error(`阶段 #${i + 1} 对象大小未配置`); s._expanded = true; return false }
        }
      } else if (this.form.storageType === 'posix') {
        if (this.form.fioStages.length === 0) { this.$message.error('至少配置一个 FIO 阶段'); return false }
        for (const [i, s] of this.form.fioStages.entries()) {
          if (!s.name) { this.$message.error(`阶段 #${i + 1} 缺少名称`); s._expanded = true; return false }
          if (!s.rw) { this.$message.error(`阶段 #${i + 1} 缺少 IO 模式`); s._expanded = true; return false }
        }
      } else if (this.form.storageType === 'mdtest') {
        if (this.form.mdtestStages.length === 0) { this.$message.error('至少配置一个 mdtest 阶段'); return false }
        for (const [i, s] of this.form.mdtestStages.entries()) {
          if (!s.name) { this.$message.error(`阶段 #${i + 1} 缺少名称`); s._expanded = true; return false }
        }
      }
      return true
    },
    buildS3Stages() {
      return this.form.stages.map(s => ({
        name: s.name,
        ops: s.ops.map(o => ({ type: o.type, weight: Number(o.weight || 1) })),
        numjobs: Number(s.numjobs || 0),
        runtime: Number(s.runtime || 0),
        numObjects: Number(s.numObjects || 0),
        objectSize: packObjSize(s.objectSize),
        deleteAll: !!s.deleteAll,
      }))
    },
    buildFioStages() {
      return this.form.fioStages.map(s => {
        // Final guard: sourceStage non-empty implies the user wants to
        // reuse those files, even if the toggle didn't get flipped (old
        // form state, race, edit-existing-rule path, …). See the
        // comment on onSourceStageChange for the failure mode this
        // prevents.
        const reuse = !!s.reuseFiles || !!s.sourceStage
        const out = { name: s.name, rw: s.rw, reuseFiles: reuse, skip: !!s.skip }
        if (s.bs) out.bs = s.bs
        if (this.isMixedRW(s.rw) && s.rwmixread != null) out.rwmixread = Number(s.rwmixread)
        if (s.sourceStage) out.sourceStage = s.sourceStage
        if (s.iodepth) out.iodepth = Number(s.iodepth)
        if (s.numjobs) out.numjobs = Number(s.numjobs)
        if (s.size) out.size = s.size
        if (s.runtime) out.runtime = Number(s.runtime)
        if (s.direct != null) out.direct = Number(s.direct)
        return out
      })
    },
    buildMdtestStages() {
      return this.form.mdtestStages.map(s => {
        const out = { name: s.name, skip: !!s.skip }
        if (s.iterations) out.iterations = Number(s.iterations)
        if (s.numItems) out.numItems = Number(s.numItems)
        if (s.itemsPerDir) out.itemsPerDir = Number(s.itemsPerDir)
        if (s.depth) out.depth = Number(s.depth)
        if (s.branching) out.branching = Number(s.branching)
        if (s.writeBytes) out.writeBytes = Number(s.writeBytes)
        if (s.readBytes) out.readBytes = Number(s.readBytes)
        if (s.onlyFiles) out.onlyFiles = true
        if (s.onlyDirs) out.onlyDirs = true
        if (s.uniqueDir) out.uniqueDir = true
        if (s.numTasks) out.numTasks = Number(s.numTasks)
        if (s.extraArgs) out.extraArgs = s.extraArgs
        return out
      })
    },
    async handleSubmit() {
      const valid = await this.$refs.form.validate().catch(() => false)
      if (!valid) return
      if (!this.validateStages()) return

      const ruleId = this.isEditMode ? this.form.id : this.generateRuleId()
      const payload = {
        cluster_name: this.clusterName,
        id: ruleId,
        name: this.form.name,
        storageType: this.form.storageType,
        parallelism: this.form.parallelism,
      }

      if (this.form.storageType === 's3') {
        // master/spec.BenchRule.BackendID is `string`; el-select :value pulls
        // the numeric mysql id from sync_storage_backend. Stringify so master
        // unmarshal doesn't fail with "cannot unmarshal number into string".
        payload.backendID = this.form.backendID == null ? '' : String(this.form.backendID)
        payload.keyPrefix = this.form.keyPrefix
        payload.stages = this.buildS3Stages()
      } else if (this.form.storageType === 'posix') {
        payload.mountPath = this.form.mountPath
        payload.fioDefaults = { ...this.form.fioDefaults }
        payload.fioStages = this.buildFioStages()
      } else if (this.form.storageType === 'mdtest') {
        payload.mountPath = this.form.mountPath
        payload.mdtestDefaults = { ...this.form.mdtestDefaults }
        payload.mdtestStages = this.buildMdtestStages()
      }

      this.submitting = true
      try {
        if (this.isEditMode) {
          await updateBenchRule(payload)
          this.$message.success(this.$t('common.edit') + this.$t('common.xxsuc'))
        } else {
          await createBenchRule(payload)
          this.$message.success(this.$t('common.create') + this.$t('common.xxsuc'))
        }
        this.$emit('confirm')
        this.innerVisible = false
      } finally {
        this.submitting = false
      }
    },
  },
}
</script>

<style lang="scss" scoped>
.bench-dialog {
  ::v-deep .el-dialog__body {
    padding: 8px 24px 16px;
    max-height: 78vh;
    overflow-y: auto;
  }
  ::v-deep .el-form-item {
    margin-bottom: 12px;
  }
  ::v-deep .el-form-item__label {
    padding-bottom: 2px;
    line-height: 1.3;
    font-size: 12px;
    color: #606266;
  }
}

.section {
  background: #fafbfc;
  border: 1px solid #ebeef5;
  border-radius: 6px;
  padding: 12px 16px;
  margin-bottom: 14px;
}

.section-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  font-size: 14px;
  color: #303133;
  margin-bottom: 10px;
  small {
    margin-left: 6px;
    color: #909399;
    font-size: 12px;
    font-weight: normal;
  }
}

.form-hint {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 4px;
}

.empty-tip {
  color: #909399;
  font-size: 13px;
  padding: 18px 0;
  text-align: center;
  border: 1px dashed #dcdfe6;
  border-radius: 4px;
  background: #fff;
}

.stage {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  margin-bottom: 8px;
  transition: border-color 0.2s;
  &.is-expanded {
    border-color: #409eff;
  }
  &.is-skip {
    opacity: 0.55;
  }
}

.stage-head {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  cursor: pointer;
  user-select: none;
  i {
    margin-right: 8px;
    color: #909399;
    transition: transform 0.15s;
  }
  .stage-name {
    font-weight: 600;
    color: #303133;
    min-width: 180px;
  }
  .stage-summary {
    color: #606266;
    font-size: 12px;
    flex: 1;
    margin-left: 12px;
  }
  .stage-skip {
    margin-right: 12px;
  }
  .stage-del {
    color: #ed4014;
    padding: 0;
  }
  &:hover {
    background: #f5f7fa;
  }
}

.stage-body {
  padding: 4px 16px 8px;
  border-top: 1px solid #ebeef5;
}

.op-row {
  display: flex;
  align-items: center;
  margin-bottom: 6px;
  .op-sep {
    margin: 0 8px;
    color: #909399;
    font-size: 12px;
  }
  .op-del {
    color: #ed4014;
    margin-left: 8px;
  }
}
</style>
