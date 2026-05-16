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
    <div class="toolbar">
      <el-button size="small" @click="loadData">{{ $t('button.refresh') }}</el-button>
      <el-button
        v-auth="'CFS_SYNCBACKEND_CREATE'"
        type="primary"
        size="small"
        @click="openCreate"
      >{{ $t('sync.createbackend') }}</el-button>
    </div>

    <el-table :data="dataList" border>
      <el-table-column :label="$t('sync.backendname')" min-width="140" prop="name"></el-table-column>
      <el-table-column :label="$t('sync.backendkind')" min-width="80" prop="kind"></el-table-column>
      <el-table-column :label="$t('sync.backendendpoint')" min-width="200" prop="endpoint"></el-table-column>
      <el-table-column :label="$t('sync.backendbucket')" min-width="120" prop="bucket"></el-table-column>
      <el-table-column :label="$t('sync.backendregion')" min-width="100" prop="region">
        <template slot-scope="scope">{{ scope.row.region || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('sync.akmasked')" min-width="160" prop="access_key_masked"></el-table-column>
      <el-table-column :label="$t('sync.backendremark')" min-width="140" prop="remark">
        <template slot-scope="scope">{{ scope.row.remark || '-' }}</template>
      </el-table-column>
      <el-table-column :label="$t('common.action')" min-width="140">
        <template slot-scope="scope">
          <el-button v-auth="'CFS_SYNCBACKEND_UPDATE'" type="text" @click="openEdit(scope.row)">{{ $t('common.edit') }}</el-button>
          <el-button v-auth="'CFS_SYNCBACKEND_DELETE'" type="text" style="color: #ed4014" @click="handleDelete(scope.row)">{{ $t('sync.deletebackend') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      :visible.sync="dialogVisible"
      :title="dialogMode === 'create' ? $t('sync.createbackend') : $t('sync.editbackend')"
      width="560px"
      append-to-body
      @close="handleDialogClose"
    >
      <el-form ref="backendForm" :model="form" :rules="rules" label-width="120px" size="small">
        <el-form-item :label="$t('sync.backendname')" prop="name">
          <el-input v-model.trim="form.name" :placeholder="$t('sync.backendname')" clearable></el-input>
        </el-form-item>
        <el-form-item :label="$t('sync.backendkind')" prop="kind">
          <el-select v-model="form.kind" style="width: 100%;">
            <el-option label="TOS" value="tos"></el-option>
            <el-option label="BOS" value="bos"></el-option>
            <el-option label="S3" value="s3"></el-option>
          </el-select>
        </el-form-item>
        <el-form-item :label="$t('sync.backendendpoint')" prop="endpoint">
          <el-input v-model.trim="form.endpoint" placeholder="https://..." clearable></el-input>
        </el-form-item>
        <el-form-item :label="$t('sync.backendbucket')" prop="bucket">
          <el-input v-model.trim="form.bucket" clearable></el-input>
        </el-form-item>
        <el-form-item :label="$t('sync.backendregion')">
          <el-input v-model.trim="form.region" clearable></el-input>
        </el-form-item>
        <el-form-item :label="$t('sync.backendaccesskey')" :prop="dialogMode === 'create' ? 'access_key' : undefined">
          <el-input v-model.trim="form.access_key" :placeholder="dialogMode === 'edit' ? $t('sync.keepaccesskey') : ''" clearable></el-input>
        </el-form-item>
        <el-form-item :label="$t('sync.backendsecretkey')" :prop="dialogMode === 'create' ? 'secret_key' : undefined">
          <el-input v-model.trim="form.secret_key" type="password" show-password :placeholder="dialogMode === 'edit' ? $t('sync.keepsecretkey') : ''" clearable></el-input>
        </el-form-item>
        <el-form-item :label="$t('sync.backendusepathstyle')">
          <el-switch v-model="form.use_path_style"></el-switch>
        </el-form-item>
        <el-form-item :label="$t('sync.backendinsecuretls')">
          <el-switch v-model="form.insecure_tls"></el-switch>
        </el-form-item>
        <el-form-item :label="$t('sync.backendremark')">
          <el-input v-model.trim="form.remark" type="textarea" :rows="2" clearable></el-input>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="handleDialogClose">{{ $t('button.cancel') }}</el-button>
        <el-button type="primary" @click="handleSubmit">{{ $t('button.submit') }}</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import {
  createSyncStorageBackend,
  deleteSyncStorageBackend,
  getSyncStorageBackendList,
  updateSyncStorageBackend,
} from '@/api/cfs/cluster'

const emptyForm = () => ({
  name: '',
  kind: 'tos',
  endpoint: '',
  bucket: '',
  region: '',
  access_key: '',
  secret_key: '',
  use_path_style: false,
  insecure_tls: false,
  remark: '',
})

export default {
  name: 'SyncStorageBackendTab',
  props: {
    clusterName: {
      type: String,
      required: true,
    },
  },
  data() {
    return {
      dataList: [],
      dialogVisible: false,
      dialogMode: 'create',
      editId: null,
      form: emptyForm(),
    }
  },
  computed: {
    rules() {
      const required = [{ required: true, message: this.$t('common.required'), trigger: 'blur' }]
      return {
        name: required,
        kind: required,
        endpoint: required,
        bucket: required,
        access_key: this.dialogMode === 'create' ? required : [],
        secret_key: this.dialogMode === 'create' ? required : [],
      }
    },
  },
  mounted() {
    this.loadData()
  },
  methods: {
    async loadData() {
      const { data } = await getSyncStorageBackendList({ cluster_name: this.clusterName })
      this.dataList = data || []
    },
    openCreate() {
      this.dialogMode = 'create'
      this.editId = null
      this.form = emptyForm()
      this.dialogVisible = true
    },
    openEdit(row) {
      this.dialogMode = 'edit'
      this.editId = row.id
      this.form = {
        name: row.name,
        kind: row.kind,
        endpoint: row.endpoint,
        bucket: row.bucket,
        region: row.region || '',
        access_key: '',
        secret_key: '',
        use_path_style: !!row.use_path_style,
        insecure_tls: !!row.insecure_tls,
        remark: row.remark || '',
      }
      this.dialogVisible = true
    },
    handleDialogClose() {
      this.dialogVisible = false
      this.$refs.backendForm && this.$refs.backendForm.resetFields()
    },
    async handleSubmit() {
      const valid = await this.$refs.backendForm.validate().catch(() => false)
      if (!valid) return
      if (this.dialogMode === 'create') {
        await createSyncStorageBackend({ cluster_name: this.clusterName, ...this.form })
        this.$message.success(this.$t('sync.createbackend') + this.$t('common.xxsuc'))
      } else {
        await updateSyncStorageBackend({ cluster_name: this.clusterName, id: this.editId, ...this.form })
        this.$message.success(this.$t('sync.editbackend') + this.$t('common.xxsuc'))
      }
      this.handleDialogClose()
      this.loadData()
    },
    async handleDelete(row) {
      await this.$confirm(this.$t('sync.confirmdeletebackend'), this.$t('common.notice'), {
        confirmButtonText: this.$t('common.yes'),
        cancelButtonText: this.$t('common.no'),
        type: 'warning',
      })
      await deleteSyncStorageBackend({ cluster_name: this.clusterName, id: row.id })
      this.$message.success(this.$t('sync.deletebackend') + this.$t('common.xxsuc'))
      this.loadData()
    },
  },
}
</script>

<style lang="scss" scoped>
.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
</style>
