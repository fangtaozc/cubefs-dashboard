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

<!--
  Reusable mount-path picker shared by 测试管理 (BenchRuleCreateDialog) and
  同步管理 (SyncRuleCreateDialog). Renders a grouped el-select listing
  fuse.cubefs mounts under "CubeFS 卷" and everything else (本地 / GPFS / 其他)
  under a separate group, while still allowing free-form input via
  allow-create — so an admin can type a path that's not yet on any syncnode.

  The parent owns the inventory (passed in via :options) so a dialog with
  multiple instances (e.g. src + dst) only fetches getSyncNodeList once.
-->
<template>
  <div class="mount-path-select">
    <el-select
      :value="value"
      filterable
      allow-create
      default-first-option
      clearable
      :disabled="disabled"
      :placeholder="placeholder"
      style="width: 100%"
      @input="$emit('input', $event)"
    >
      <el-option-group v-if="options.cubefs && options.cubefs.length" label="CubeFS 卷">
        <el-option
          v-for="m in options.cubefs"
          :key="'cfs-' + m.path"
          :label="m.path"
          :value="m.path"
        >
          <span>{{ m.path }}</span>
          <span class="fs-type">{{ m.fsType }}</span>
        </el-option>
      </el-option-group>
      <el-option-group v-if="options.external && options.external.length" label="本地 / GPFS / 其他">
        <el-option
          v-for="m in options.external"
          :key="'ext-' + m.path"
          :label="m.path"
          :value="m.path"
        >
          <span>{{ m.path }}</span>
          <span class="fs-type">{{ m.fsType || '未知' }}</span>
        </el-option>
      </el-option-group>
    </el-select>
    <div v-if="hint" class="mount-path-hint">{{ hint }}</div>
  </div>
</template>

<script>
export default {
  name: 'MountPathSelect',
  props: {
    value: {
      type: String,
      default: '',
    },
    options: {
      type: Object,
      required: true,
      validator(v) {
        return v && Array.isArray(v.cubefs) && Array.isArray(v.external)
      },
    },
    disabled: {
      type: Boolean,
      default: false,
    },
    placeholder: {
      type: String,
      default: '选择已挂载路径，或自行输入',
    },
    hint: {
      type: String,
      default: '',
    },
  },
}
</script>

<style lang="scss" scoped>
.mount-path-select {
  width: 100%;
}

.fs-type {
  float: right;
  color: #909399;
  font-size: 12px;
}

.mount-path-hint {
  margin-top: 4px;
  font-size: 12px;
  color: #9ca3af;
  line-height: 1.5;
}
</style>
