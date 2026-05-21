<!--
 Copyright 2026 The CubeFS Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0
-->

<template>
  <el-dialog
    :visible.sync="innerVisible"
    title="新建 POSIX 兼容性测试"
    width="640px"
    append-to-body
    @open="onOpen"
    @close="handleClose"
  >
    <el-form
      ref="form"
      :model="form"
      :rules="rules"
      label-width="120px"
      size="small"
    >
      <el-form-item label="运行标签" prop="target_vol">
        <el-input v-model.trim="form.target_vol" placeholder="如 release-v3.6 验收 / chmod-regress"></el-input>
        <div class="hint">本次运行的备注/标签。注意：测试在 dashboard 自动创建的<b>临时 CubeFS 卷</b>上跑，不会污染已有卷。</div>
      </el-form-item>
      <el-form-item label="测试套件镜像">
        <el-input v-model.trim="form.suite_image" :placeholder="DEFAULT_IMAGE"></el-input>
        <div class="hint">留空则使用默认（{{ DEFAULT_IMAGE }}）</div>
      </el-form-item>
      <el-form-item label="子目录">
        <el-input v-model.trim="form.mount_subdir" placeholder="留空自动生成"></el-input>
        <div class="hint">在临时卷下的工作子目录，运行结束随卷一起销毁</div>
      </el-form-item>
      <el-form-item label="测试过滤">
        <el-input v-model.trim="form.test_filter" placeholder="空格分隔，如：rename chmod/00.t"></el-input>
        <div class="hint">留空跑全部用例（约 8000 个，全跑几十分钟）。冒烟可填 chmod/00.t</div>
      </el-form-item>
    </el-form>
    <span slot="footer">
      <el-button size="small" @click="handleClose">{{ $t('common.cancel') }}</el-button>
      <el-button size="small" type="primary" :loading="submitting" @click="handleSubmit">{{ $t('common.confirm') }}</el-button>
    </span>
  </el-dialog>
</template>

<script>
import { runPosixCheck } from '@/api/cfs/cluster'

const DEFAULT_IMAGE = 'hub.shiyak-office.com/storage/pjd-fstest:20090130-rc2'

export default {
  name: 'PosixCheckRunDialog',
  props: {
    visible: { type: Boolean, default: false },
    clusterName: { type: String, required: true },
  },
  data() {
    return {
      DEFAULT_IMAGE,
      submitting: false,
      form: {
        target_vol: '',
        suite_image: '',
        mount_subdir: '',
        test_filter: '',
      },
      rules: {
        target_vol: [{ required: true, message: '请填写运行标签', trigger: 'blur' }],
      },
    }
  },
  computed: {
    innerVisible: {
      get() { return this.visible },
      set(val) { this.$emit('update:visible', val) },
    },
  },
  methods: {
    onOpen() {
      this.form = { target_vol: '', suite_image: '', mount_subdir: '', test_filter: '' }
      this.$nextTick(() => { this.$refs.form && this.$refs.form.clearValidate() })
    },
    handleClose() {
      this.innerVisible = false
      this.$emit('close')
    },
    async handleSubmit() {
      const valid = await this.$refs.form.validate().catch(() => false)
      if (!valid) return
      this.submitting = true
      try {
        await runPosixCheck({
          cluster_name: this.clusterName,
          target_vol: this.form.target_vol,
          suite_image: this.form.suite_image || undefined,
          mount_subdir: this.form.mount_subdir || undefined,
          test_filter: this.form.test_filter || undefined,
        })
        this.$message.success('已创建运行任务')
        this.$emit('confirm')
        this.innerVisible = false
      } catch (e) {
        this.$message.error('创建失败：' + (e.message || e))
      } finally {
        this.submitting = false
      }
    },
  },
}
</script>

<style lang="scss" scoped>
.hint {
  color: #909399;
  font-size: 12px;
  margin-top: 4px;
}
</style>
