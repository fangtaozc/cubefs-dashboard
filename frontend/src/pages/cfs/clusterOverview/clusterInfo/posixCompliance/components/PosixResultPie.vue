<!--
 Copyright 2026 The CubeFS Authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0
-->

<template>
  <div ref="chart" class="posix-pie"></div>
</template>

<script>
import Echarts from 'echarts'

export default {
  name: 'PosixResultPie',
  props: {
    pass: { type: Number, default: 0 },
    fail: { type: Number, default: 0 },
    skip: { type: Number, default: 0 },
  },
  data() {
    return { inst: null, resizeHandler: null }
  },
  computed: {
    summary() {
      return [this.pass, this.fail, this.skip].join(',')
    },
  },
  watch: {
    summary() { this.$nextTick(this.render) },
  },
  mounted() {
    this.$nextTick(this.render)
    this.resizeHandler = () => this.inst && this.inst.resize()
    window.addEventListener('resize', this.resizeHandler)
  },
  beforeDestroy() {
    if (this.resizeHandler) window.removeEventListener('resize', this.resizeHandler)
    if (this.inst) { this.inst.dispose(); this.inst = null }
  },
  methods: {
    render() {
      if (!this.$refs.chart) return
      if (!this.inst) this.inst = Echarts.init(this.$refs.chart)
      const total = this.pass + this.fail + this.skip
      const passRate = total > 0 ? ((this.pass / total) * 100).toFixed(1) : '-'
      this.inst.setOption({
        tooltip: { trigger: 'item', formatter: '{b}: {c} ({d}%)' },
        title: {
          left: 'center', top: 'center',
          text: total > 0 ? `${passRate}%` : '无数据',
          subtext: total > 0 ? `通过率` : '',
          textStyle: { fontSize: 22, fontWeight: 600, color: this.fail > 0 ? '#e6a23c' : '#67c23a' },
          subtextStyle: { fontSize: 12, color: '#909399' },
        },
        legend: { bottom: 0, icon: 'circle' },
        series: [{
          type: 'pie',
          radius: ['52%', '78%'],
          avoidLabelOverlap: true,
          label: { show: false },
          labelLine: { show: false },
          data: [
            { name: '通过', value: this.pass, itemStyle: { color: '#67c23a' } },
            { name: '失败', value: this.fail, itemStyle: { color: '#f56c6c' } },
            { name: '跳过', value: this.skip, itemStyle: { color: '#909399' } },
          ],
        }],
      }, true)
    },
  },
}
</script>

<style lang="scss" scoped>
.posix-pie {
  width: 100%;
  height: 220px;
}
</style>
