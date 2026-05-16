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
    :title="dialogTitle"
    :visible.sync="innerVisible"
    width="800px"
    @close="handleClose"
  >
    <el-alert
      :title="$t('sync.ruleconfig')"
      type="info"
      :closable="false"
      class="mg-bt-s"
    ></el-alert>
    <el-input
      v-model="editorValue"
      type="textarea"
      :rows="18"
      resize="none"
    ></el-input>
    <div slot="footer">
      <el-button @click="handleClose">{{ $t('button.cancel') }}</el-button>
      <el-button type="primary" @click="handleConfirm">{{ $t('button.submit') }}</el-button>
    </div>
  </el-dialog>
</template>

<script>
export default {
  name: 'SyncRuleEditorDialog',
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    mode: {
      type: String,
      default: 'create',
    },
    value: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      innerVisible: false,
      editorValue: '',
    }
  },
  computed: {
    dialogTitle() {
      return `${this.mode === 'edit' ? this.$t('common.edit') : this.$t('common.create')}${this.$t('sync.rules')}`
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
      },
    },
  },
  methods: {
    handleClose() {
      this.$emit('update:visible', false)
      this.$emit('close')
    },
    handleConfirm() {
      this.$emit('confirm', this.editorValue)
    },
  },
}
</script>
