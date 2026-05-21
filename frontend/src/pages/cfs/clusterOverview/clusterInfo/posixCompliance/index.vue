<!--
 Copyright 2026 The CubeFS Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0
-->

<template>
  <div>
    <el-card>
      <div class="toolbar">
        <div class="title">{{ clusterName }} / POSIX 测试</div>
        <div>
          <el-button v-auth="'CFS_POSIXCHECK_LIST'" size="small" plain @click="loadData">{{ $t('common.refresh') || '刷新' }}</el-button>
          <el-button v-auth="'CFS_POSIXCHECK_RUN'" size="small" type="primary" @click="openRun">新建测试</el-button>
        </div>
      </div>

      <div class="intro">
        基于第三方 <a href="https://github.com/pjd/pjdfstest" target="_blank">pjd-fstest</a> 测试套件，针对 CubeFS 文件系统做 POSIX 语义合规验证（mkdir / rename / chmod / chown / symlink 等系统调用）。每次运行启动一个 K8s Job 在自动创建的<b>临时 CubeFS 卷</b>上跑测试，结果按 pass / fail / skip 分类，失败用例可下钻查看。
      </div>

      <u-page-table :data="dataList" :page-size="page.per_page" border>
        <el-table-column label="ID" prop="id" min-width="60"></el-table-column>
        <el-table-column label="运行标签" prop="target_vol" min-width="160"></el-table-column>
        <el-table-column label="状态" min-width="100">
          <template slot-scope="scope">
            <el-tag :type="statusTagType(scope.row.status)" size="mini">{{ scope.row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="通过" prop="pass_count" min-width="70"></el-table-column>
        <el-table-column label="失败" min-width="70">
          <template slot-scope="scope">
            <span :style="{ color: scope.row.fail_count > 0 ? '#ed4014' : '#67c23a' }">{{ scope.row.fail_count }}</span>
          </template>
        </el-table-column>
        <el-table-column label="跳过" prop="skip_count" min-width="70"></el-table-column>
        <el-table-column label="耗时" min-width="80">
          <template slot-scope="scope">
            <span>{{ scope.row.duration_sec ? scope.row.duration_sec + 's' : '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="触发者" prop="trigger_user" min-width="100"></el-table-column>
        <el-table-column label="创建时间" min-width="160">
          <template slot-scope="scope">
            <span>{{ formatTime(scope.row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" min-width="180">
          <template slot-scope="scope">
            <el-button v-auth="'CFS_POSIXCHECK_GET'" type="text" @click="openDetail(scope.row)">详情</el-button>
            <el-button
              v-if="scope.row.status === 'running' || scope.row.status === 'pending'"
              v-auth="'CFS_POSIXCHECK_CANCEL'"
              type="text"
              style="color: #ed4014"
              @click="cancelRun(scope.row)"
            >取消</el-button>
            <el-button
              v-auth="'CFS_POSIXCHECK_DELETE'"
              type="text"
              style="color: #ed4014"
              @click="deleteRun(scope.row)"
            >删除</el-button>
          </template>
        </el-table-column>
      </u-page-table>
    </el-card>

    <RunDialog
      :visible.sync="runDialogVisible"
      :cluster-name="clusterName"
      @confirm="onRunCreated"
    />

    <el-drawer
      :visible.sync="detailDrawerVisible"
      :title="`运行 #${detail && detail.run && detail.run.id || ''}`"
      direction="rtl"
      size="50%"
      append-to-body
    >
      <div v-if="detail && detail.run" class="detail">
        <el-row :gutter="12">
          <el-col :span="14">
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item label="状态">{{ detail.run.status }}</el-descriptions-item>
              <el-descriptions-item label="运行标签">{{ detail.run.target_vol }}</el-descriptions-item>
              <el-descriptions-item label="套件镜像">{{ detail.run.suite_image }}</el-descriptions-item>
              <el-descriptions-item label="子目录">{{ detail.run.mount_subdir }}</el-descriptions-item>
              <el-descriptions-item label="测试过滤">{{ detail.run.test_filter || '(全部)' }}</el-descriptions-item>
              <el-descriptions-item label="耗时">{{ detail.run.duration_sec }}s</el-descriptions-item>
              <el-descriptions-item label="通过 / 失败 / 跳过">
                <span style="color:#67c23a;">{{ detail.run.pass_count }}</span>
                /
                <span style="color:#ed4014;">{{ detail.run.fail_count }}</span>
                /
                <span style="color:#909399;">{{ detail.run.skip_count }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="错误信息">{{ detail.run.error_msg || '-' }}</el-descriptions-item>
            </el-descriptions>
          </el-col>
          <el-col :span="10">
            <PosixResultPie
              :pass="detail.run.pass_count || 0"
              :fail="detail.run.fail_count || 0"
              :skip="detail.run.skip_count || 0"
            />
          </el-col>
        </el-row>

        <h3 v-if="failures.length > 0" style="margin-top: 20px;">失败用例 ({{ failures.length }})</h3>
        <el-collapse v-if="failures.length > 0">
          <el-collapse-item
            v-for="(group, syscall) in groupedFailures"
            :key="syscall"
            :name="syscall"
            :title="`${syscall} (${group.length})`"
          >
            <el-table :data="group" border size="mini">
              <el-table-column label="文件" prop="test_file" min-width="120"></el-table-column>
              <el-table-column label="#" prop="test_number" width="50"></el-table-column>
              <el-table-column label="描述" prop="description" min-width="200"></el-table-column>
              <el-table-column label="期望" prop="expected" min-width="120"></el-table-column>
              <el-table-column label="实际" prop="actual" min-width="120"></el-table-column>
            </el-table>
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-drawer>
  </div>
</template>

<script>
import mixin from '@/pages/cfs/clusterOverview/mixin'
import UPageTable from '@/pages/components/uPageTable'
import { listPosixCheckRuns, getPosixCheckRun, cancelPosixCheck, deletePosixCheck } from '@/api/cfs/cluster'
import RunDialog from './components/RunDialog.vue'
import PosixResultPie from './components/PosixResultPie.vue'

function fmtDate(s) {
  if (!s) return '-'
  const d = new Date(s)
  if (isNaN(d.getTime())) return s
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export default {
  name: 'PosixCompliance',
  components: { UPageTable, RunDialog, PosixResultPie },
  mixins: [mixin],
  data() {
    return {
      dataList: [],
      total: 0,
      page: { current: 1, per_page: 20 },
      runDialogVisible: false,
      detailDrawerVisible: false,
      detail: null,
      failures: [],
      pollTimer: null,
    }
  },
  computed: {
    groupedFailures() {
      const g = {}
      for (const f of this.failures) {
        const k = f.syscall || 'unknown'
        if (!g[k]) g[k] = []
        g[k].push(f)
      }
      return g
    },
  },
  mounted() {
    this.loadData()
    // 10s 轮询，只在有 running/pending 的 run 时拉数据；拉到后 patch in-place
    // 避免整张表 re-render 抖动。
    this.pollTimer = setInterval(() => {
      if (this.dataList.some(r => r.status === 'running' || r.status === 'pending')) {
        this.loadData(true)
      }
    }, 10000)
  },
  beforeDestroy() {
    if (this.pollTimer) clearInterval(this.pollTimer)
  },
  methods: {
    formatTime: fmtDate,
    statusTagType(s) {
      switch (s) {
        case 'done': return 'success'
        case 'failed': return 'danger'
        case 'cancelled': return 'info'
        case 'running': return 'warning'
        default: return ''
      }
    },
    // Silent refresh: patch rows in-place rather than replacing the whole
    // array, otherwise el-table re-renders on every poll and flickers.
    patchDataList(newList) {
      const sig = obj => JSON.stringify(obj)
      const newById = new Map(newList.map(r => [r.id, r]))
      const oldById = new Map(this.dataList.map((r, i) => [r.id, i]))
      for (const [id, item] of newById) {
        const idx = oldById.get(id)
        if (idx !== undefined) {
          const existing = this.dataList[idx]
          if (sig(existing) !== sig(item)) Object.assign(existing, item)
        }
      }
      // remove rows no longer present
      for (let i = this.dataList.length - 1; i >= 0; i--) {
        if (!newById.has(this.dataList[i].id)) this.dataList.splice(i, 1)
      }
      // append new rows
      for (const item of newList) {
        if (!oldById.has(item.id)) this.dataList.push(item)
      }
    },
    async loadData(silent = false) {
      try {
        const { data } = await listPosixCheckRuns({
          cluster_name: this.clusterName,
          page: this.page.current,
          page_size: this.page.per_page,
        })
        const next = data?.data || []
        if (silent) {
          this.patchDataList(next)
          // total 可能也变（新增/删除），但纯轮询大概率不变，单独更新避免无关刷新
          if (this.total !== (data?.total || 0)) this.total = data?.total || 0
        } else {
          this.dataList = next
          this.total = data?.total || 0
        }
      } catch (e) {
        if (!silent) this.$message.error('加载失败：' + (e.message || e))
      }
    },
    openRun() { this.runDialogVisible = true },
    onRunCreated() {
      this.runDialogVisible = false
      this.loadData()
    },
    async openDetail(row) {
      this.detail = null
      this.failures = []
      this.detailDrawerVisible = true
      try {
        const { data } = await getPosixCheckRun({ cluster_name: this.clusterName, id: row.id })
        this.detail = { run: data?.run || row }
        this.failures = data?.failures || []
      } catch (e) {
        this.$message.error('加载详情失败：' + (e.message || e))
      }
    },
    async cancelRun(row) {
      try {
        await this.$confirm(`确定取消运行 #${row.id} 吗？`, '提示', { type: 'warning' })
      } catch (_) { return }
      try {
        await cancelPosixCheck({ cluster_name: this.clusterName, id: row.id })
        this.$message.success('已取消')
        this.loadData()
      } catch (e) {
        this.$message.error('取消失败：' + (e.message || e))
      }
    },
    async deleteRun(row) {
      try {
        await this.$confirm(`确定删除运行 #${row.id} 吗？该操作会同时删除关联的失败用例记录，不可恢复。`, '提示', { type: 'warning' })
      } catch (_) { return }
      try {
        await deletePosixCheck({ cluster_name: this.clusterName, id: row.id })
        this.$message.success('已删除')
        // 关掉抽屉如果正打开同一条
        if (this.detail && this.detail.run && this.detail.run.id === row.id) {
          this.detailDrawerVisible = false
        }
        this.loadData()
      } catch (e) {
        this.$message.error('删除失败：' + (e.message || e))
      }
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
.intro {
  color: #606266;
  font-size: 13px;
  background: #f5f7fa;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  padding: 10px 14px;
  margin-bottom: 14px;
  line-height: 1.6;
  a { color: #409eff; }
}
.detail {
  padding: 0 18px 18px;
}
</style>
