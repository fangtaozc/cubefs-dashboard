# Sync 子系统管理设计方案

## 1. 文档定位

本文档是 `cubefs-dashboard` 仓库内的 SDD 设计文档，目标是在现有 CFS 集群管理面中增加 Sync 子系统管理能力。

本次设计基于 2026-05 的 P2 cutover 规格，不再沿用“syncnode 本地管理 rule/task”的旧模型。

本文档只定义 dashboard 侧：

- 页面入口与交互
- backend 代理与协议归一化
- 数据模型扩展
- 权限、日志、测试与验收标准

不定义 syncnode 或 master 本身的实现。

## 2. 规格输入

### 2.1 上游规格

- syncnode 总体设计：
  `/Users/tao.fang/codes/cubefs/docs/plan/syncnode/design.md`
- syncnode 本地诊断面 OpenAPI：
  `/Users/tao.fang/codes/cubefs/docs/plan/syncnode/openapi.yaml`
- master Sync 子系统 OpenAPI：
  `/Users/tao.fang/codes/cubefs/docs/plan/master/sync_openapi.yaml`

### 2.2 当前 dashboard 现状

- 前端只接受 dashboard 自己的统一响应格式：`HTTP 200 + { code: 200, msg, data }`
- backend 当前统一挂在 `/api/cubefs/console/cfs/:cluster/...`
- 现有 `Cluster` 模型尚无 sync 管理配置字段
- 现有权限、操作日志、i18n 都需要显式扩展

## 3. 上游新事实

### 3.1 控制面分层已经变化

P2 后，上游接口已经明确拆成两层：

#### master 是权威控制面

由 master 提供：

- `syncRule/*`
- `syncTask/*`
- `syncNode/add|list|decommission|drain|restore|tasks`

含义：

- rule 存在 raft
- cron 在 master leader 上跑
- task ledger 在 master 内存维护
- 节点生命周期也由 master 管

#### syncnode 只保留本地诊断面

syncnode 本地只保留：

- `/admin/syncnode/version`
- `/admin/syncnode/stat`
- `/admin/syncnode/reload`

并且：

- `scheduledRules` 在 `stat` 中恒为 `0`
- `reload` 只重载并应用非 rule 配置
- `sync.json` 中 `rules:[...]` 已被忽略

### 3.2 dashboard 不应再设计为“按节点 CRUD 规则和任务”

旧方案中这些假设已经失效：

- rule 需要选择某个 syncnode 节点创建
- task 需要打到某个 syncnode 节点触发
- rule/task 数据归属于节点本地 BoltDB

P2 后正确语义是：

- rule 是 cluster-global，走 master
- task 是 cluster-global，走 master
- syncnode 是 worker，只有本地诊断信息走节点

## 4. 目标与非目标

### 4.1 目标

- 在 cluster detail 中增加 Sync 管理入口
- 支持通过 master 管理 sync rules、sync tasks、sync nodes
- 支持对单个 syncnode 做本地诊断：version、stat、reload
- 对 master / syncnode 两套协议做 dashboard 统一封装
- 在不暴露 admin token 给浏览器的前提下完成集成

### 4.2 非目标

- 不在 dashboard 内实现 syncnode 或 master 的协议逻辑
- 不为浏览器提供直连 master/syncnode 的能力
- 不在本阶段实现历史兼容的“节点本地 rule/task 管理”
- 不在本阶段实现自定义审计存储或长历史 task 落库

## 5. 关键约束

### 5.1 双来源约束

dashboard 必须同时对接两类上游：

- master：主控制面
- syncnode：本地诊断面

二者职责不能混用。

### 5.2 协议约束

上游 master/syncnode 都返回：

- `HTTP 200 + { code: 0, msg, data }` 成功
- `HTTP 4xx/5xx + { code: 1/2/4/1014... , msg, data? }` 失败

dashboard 前端仍只接受：

- `HTTP 200 + { code: 200, msg, data }` 成功

结论：

- backend 必须做协议归一化
- 前端不能直接消费上游 OpenAPI

### 5.3 leader 约束

master 的 rule/node 写操作依赖 raft 提交，写请求应尽量发送到 leader。

结论：

- backend 需要先解析 cluster leader
- 写接口优先打 leader
- 遇到 `code=4` / raft unavailable 时返回可识别的“可重试”错误

### 5.4 节点地址约束

master 的 `SyncNodeInfo.addr` 表示 syncnode TCP listen 地址 `host:port`，不是本地 admin HTTP 地址。

syncnode 本地诊断面需要访问其 `httpListen` 端口。

结论：

- dashboard 必须从 `addr` 中提取 host
- 再拼接 cluster 级配置的 `SyncNodeHTTPPort`
- v1 不支持每节点不同的 admin port

## 6. 设计决议

| ID | 决议 | 说明 |
|---|---|---|
| D1 | 保留单入口 `syncManage` | 避免把 sync 管理散落到资源页和事件页 |
| D2 | 页面主线只保留 2 个一级 tab：`Rules / Tasks` | 信息架构围绕 cluster-global 对象，而不是 worker |
| D3 | `Rules`、`Tasks` 一律走 master | 不再要求用户先选节点 |
| D4 | `Workers` 降为辅助维护入口 | 仅用于诊断、drain、decommission、restore，不承担主线导航 |
| D5 | cluster 模型只新增最小字段：`SyncNodeHTTPPort`、`SyncAdminToken` | 删除旧方案中的静态节点列表设计 |
| D6 | browser 永不持有 sync admin token | token 只保存在 backend |
| D7 | rule create/update 使用 flat `SyncRuleConfig` JSON | 与 master OpenAPI 一致，不再包装 `config` |
| D8 | task 触发入口放到 rule action `trigger` | `syncTask/trigger/save/load` 已不存在 |
| D9 | 不把 syncnode `state` 当成信息架构的依据 | `state/isActive` 只作为高级诊断元数据保留 |
| D10 | 从 worker 进入任务只做“带 owner 过滤跳转” | 不再设计 node-scoped task page |
| D11 | 节点维护分为 `drain / decommission / restore` 三类 | 明确区分临时维护下线、永久下线、恢复服务 |
| D12 | 节点任务处置优先复用 master 语义，不在 dashboard 侧循环取消任务 | 避免前端自编排造成竞态和语义漂移 |

## 7. 用户视角设计

### 7.1 入口

在 `clusterDetailChildren` 新增：

- 路由名：`syncManage`
- 菜单标题：`同步管理`

### 7.2 页面结构

页面内部只保留 2 个一级 tab：

1. `Sync Rules`
2. `Sync Tasks`

默认打开 `Sync Rules`。

页面头部提供一个次级入口：

- `执行器维护`
  - 打开全局 `Workers` 维护面板
  - 用于查看 worker 运行负载、做诊断和维护操作

### 7.3 页面关系

页面关系按对象职责组织，而不是按节点组织：

```text
Rules  ->  Tasks
             |
             v
          Worker Diagnostics
```

含义：

- `Rules` 是配置入口，也是主操作入口
- `Tasks` 是执行观测面，承接规则触发后的结果
- `Workers` 只是辅助维护面，不是规则和任务的归属容器
- 不存在“先选 worker，再进入 rules/tasks”的页面链路

具体跳转关系：

- 从 `Rules` 页点击某条规则的 `trigger` 或 `view tasks`
  - 跳到 `Tasks` 页
  - 自动带上 `ruleID` 过滤
- 从 `Tasks` 页点击某条任务的 `owner`
  - 若 owner 非空，打开 `Workers` 维护面板并定位到对应诊断抽屉
- 从页面头部点击 `执行器维护`
  - 打开全局 `Workers` 维护面板
- 从 `Workers` 维护面板点击“查看任务”
  - 只是跳到 `Tasks` 页并预填 `owner`
  - 不形成 node-scoped task page

### 7.4 Sync Rules

数据来源：master `/syncRule/*`

展示字段：

- `config.id`
- `config.type`
- `state`
- `config.schedule`
- `config.shardingStrategy`
- `config.parallelism`
- `createdAt`
- `updatedAt`
- `lastRunAt`
- `lastRunStatus`
- `lastRunError`

操作：

- `create`
- `update`
- `delete`
- `pause`
- `resume`
- `trigger`

设计要求：

- 不要求选择节点
- create/update 直接提交 flat `SyncRuleConfig`
- 显示冲突错误 `1014/1015/1016`

补充操作：

- `view tasks`

`view tasks` 的行为：

- 切换到 `Tasks` tab
- 预填 `ruleID=<config.id>`

### 7.5 Sync Tasks

数据来源：master `/syncTask/*`

顶部过滤：

- `status`
- `ruleID`
- `owner`

展示字段：

- `taskID`
- `ruleID`
- `type`
- `status`
- `owner`
- `shardIdx`
- `shardTotal`
- `startedAt`
- `doneAt`
- `error`

操作：

- `detail`
- `cancel`
- `retry`
- `export`

说明：

- task 创建入口不在本页
- 触发任务通过 `Rules` 页上的 `trigger` 完成
- 对 fan-out 父任务，详情页要展示 `owner` 为空、`shardTotal > 0` 的语义
- `export` 只导出 master 当前内存 ledger，不等价于长期审计历史
- 当 `owner` 非空时，支持跳到对应 worker 诊断

### 7.6 Workers 维护面板

进入方式：

- 页面头部 `执行器维护`
- `Tasks` 页中点击任务 `owner`

主列表数据来源：master `/syncNode/list`

列表只展示运行与负载信息，不把 `state` 当成核心业务字段：

- `addr`
- `version`
- `uptimeSeconds`
- `runningTasks`
- `queuedTasks`
- `bandwidthMBps`
- `bandwidthMBpsLimit`
- `cpuPercent`
- `memPercent`
- `boltDBHealthy`
- `loadScore`

操作：

- `diagnostics`
- `进入维护`
- `永久下线`
- `恢复服务`
- `view tasks`

不提供 `add` 按钮：

- 节点注册由 syncnode 启动时自动完成
- dashboard 只管理已注册节点

worker 详情抽屉分 3 块：

- `Runtime`
  - 运行指标、负载、boltDB 健康度
  - 当前节点任务摘要
- `Diagnostics`
  - `version`
  - `stat`
  - `reload`
  - 明示 `scheduledRules` 恒为 `0`
  - 明示 `reload` 不会重载 rule
- `Advanced`
  - 原始 `isActive`
  - 原始 `state`
  - 说明：仅用于排障，不作为主页面关系的一部分

### 7.7 节点上下线与任务处理

节点不是规则和任务的主视图，但 worker 维护面必须覆盖生命周期操作。

#### 上线

dashboard 不提供手工 `add node`。

原因：

- `syncNode/add` 是 syncnode 启动后的注册动作
- 对用户可见的“上线”语义只有一种：`restore`

具体规则：

- 当节点仍存在且 `state=draining` 时
  - 可执行 `restore`
  - 结果是节点回到 `active`
- 当节点已经被 `decommission` 移除后
  - dashboard 不提供“重新上线”按钮
  - 必须由 syncnode 重新启动并向 master 重新注册

#### 下线

下线分三种语义，UI 必须明确区分，不能只给一个“下线”按钮：

1. `进入维护`
   - 对应 `syncNode/drain`
   - 语义：清空当前节点任务，但保留节点注册记录
   - 场景：临时维护、短时摘流
2. `永久下线`
   - 入口是同一个 `永久下线` 按钮
   - 确认框中必须提供两种模式：
     - `平滑下线` -> `syncNode/decommission?force=false`
     - `强制下线` -> `syncNode/decommission?force=true`
   - 语义：
     - 平滑下线：先标记 `draining`，停止派发新任务；已有任务自然结束后由集群移除
     - 强制下线：先执行 drain 子流程，再删除节点记录
   - 场景：
     - 平滑下线：计划退役
     - 强制下线：节点故障、需要立即摘除

#### 节点任务停止语义

任务停止分两层：

- 单个任务停止
  - 走 `syncTask/cancel`
  - 是 task-scoped 异步操作
- 因节点维护引发的任务处置
  - 走 `syncNode/drain` 或 `syncNode/decommission?force=true`
  - 由 master 负责把节点上的任务清空/中断

dashboard 不应在执行 `drain` 或强制 `decommission` 时，对节点任务逐条调用 `syncTask/cancel`。

原因：

- 上游已经提供 node-scoped 维护语义
- 前端循环 cancel 会引入重复取消、竞态和结果不一致

#### 维护操作前的提示与确认

worker 维护面板在执行 `drain/decommission/restore` 前，应展示：

- 当前 `runningTasks`
- 当前 `queuedTasks`
- 受影响任务列表摘要

任务列表摘要的读取方式：

- 优先使用 master `/syncNode/tasks?addr=...`
- 仅用于 worker 维护面板中的只读影响面展示
- 不单独展开为新的节点任务页面

确认文案要求：

- `drain`
  - 明示“会清空当前节点任务，但保留节点，可后续 restore”
- `decommission(force=false)`
  - 明示“不会主动停止运行中任务，只会停止新派发，任务结束后自动移除节点”
- `decommission(force=true)`
  - 明示“会中断/清空当前节点任务，并删除节点记录”

#### 操作后的观察方式

节点动作返回后，前端不应假设任务已立即终态。

需要通过以下方式观察收敛：

- worker 面板刷新 `runningTasks/queuedTasks/state`
- `Tasks` 页按 `owner` 过滤观察任务状态变化
- 对单任务可继续轮询 `syncTask/get`

## 8. 数据模型设计

### 8.1 Cluster 扩展字段

v1 只扩展 `backend/model/cluster.go` 两个字段：

- `SyncNodeHTTPPort int`
  - 默认 `17911`
  - 用于由 syncnode TCP addr 推导本地 admin addr
- `SyncAdminToken types.EncryptStr`
  - 可空
  - 同时用于访问 master sync API 与 syncnode 本地诊断面

### 8.2 不再保留旧方案字段

以下旧方案字段不应落地：

- `SyncDiscoveryMode`
- `SyncNodeAddrs`

原因：

- 节点发现已由 master `/syncNode/list` 提供
- dashboard 不应再维护独立静态节点清单

### 8.3 迁移要求

新增一条 migration：

- migration ID 示例：`202605140_add_sync_fields_to_cluster`

要求：

- `SyncNodeHTTPPort` 有默认值
- `SyncAdminToken` 使用加密类型
- 老记录平滑升级

### 8.4 集群配置入口

扩展 cluster create/update 表单：

- `syncNode admin port`
- `sync admin token`

说明：

- token 不回显明文
- 不在 cluster list 表格展示

## 9. 后端设计

### 9.1 模块划分

新增模块：

- `backend/handler/syncnode`
- `backend/handler/syncrule`
- `backend/handler/synctask`
- `backend/service/sync`

建议在 `backend/service/sync` 内拆分：

- `master_client.go`
- `node_client.go`
- `resolver.go`
- `adapter.go`

### 9.2 resolver 设计

#### ResolveSyncMaster

职责：

1. 读取 cluster 的 `master_addr`
2. 通过现有 `cluster.Get` 获取 `LeaderAddr`
3. 写请求优先返回 leader
4. 读请求默认也走 leader，减少状态漂移

返回：

- `leaderAddr`
- `masterToken` 由 cluster 配置提供

#### ResolveSyncNodeAdminAddr

输入：

- `syncNodeTCPAddr`，例如 `10.0.1.10:17910`

输出：

- `http://10.0.1.10:<cluster.SyncNodeHTTPPort>`

### 9.3 dashboard 对外 API

为保持现有风格，仍挂在 cluster-scoped 路径下。

#### master-backed

- `GET /api/cubefs/console/cfs/:cluster/syncNode/list`
- `GET /api/cubefs/console/cfs/:cluster/syncNode/tasks?addr=...&status=...`
- `POST /api/cubefs/console/cfs/:cluster/syncNode/decommission`
- `POST /api/cubefs/console/cfs/:cluster/syncNode/drain`
- `POST /api/cubefs/console/cfs/:cluster/syncNode/restore`

- `GET /api/cubefs/console/cfs/:cluster/syncRule/list`
- `GET /api/cubefs/console/cfs/:cluster/syncRule/get?id=...`
- `POST /api/cubefs/console/cfs/:cluster/syncRule/create`
- `POST /api/cubefs/console/cfs/:cluster/syncRule/update`
- `POST /api/cubefs/console/cfs/:cluster/syncRule/delete`
- `POST /api/cubefs/console/cfs/:cluster/syncRule/pause`
- `POST /api/cubefs/console/cfs/:cluster/syncRule/resume`
- `POST /api/cubefs/console/cfs/:cluster/syncRule/trigger`

- `GET /api/cubefs/console/cfs/:cluster/syncTask/list`
- `GET /api/cubefs/console/cfs/:cluster/syncTask/get?id=...`
- `POST /api/cubefs/console/cfs/:cluster/syncTask/cancel`
- `POST /api/cubefs/console/cfs/:cluster/syncTask/retry`
- `GET /api/cubefs/console/cfs/:cluster/syncTask/export?since=...`

#### node-local diagnostics

- `GET /api/cubefs/console/cfs/:cluster/syncNode/version?addr=...`
- `GET /api/cubefs/console/cfs/:cluster/syncNode/stat?addr=...`
- `POST /api/cubefs/console/cfs/:cluster/syncNode/reload`

说明：

- 这里的 `addr` 一律使用 master 视角的 syncnode TCP listen addr
- backend 再转换成 node local admin addr
- `syncNode/tasks` 只服务于 worker 维护面板的影响面展示
- 用户从 worker 回看任务主线时，仍统一跳转 `syncTask/list?owner=`

### 9.4 上游接口映射

#### master

| dashboard API | master API |
|---|---|
| `syncNode/list` | `GET /syncNode/list` |
| `syncNode/tasks` | `GET /syncNode/tasks?addr=&status=` |
| `syncNode/decommission` | `POST /syncNode/decommission?addr=&force=` |
| `syncNode/drain` | `POST /syncNode/drain?addr=` |
| `syncNode/restore` | `POST /syncNode/restore?addr=` |
| `syncRule/list` | `GET /syncRule/list?state=` |
| `syncRule/get` | `GET /syncRule/get?id=` |
| `syncRule/create` | `POST /syncRule/create` |
| `syncRule/update` | `POST /syncRule/update` |
| `syncRule/delete` | `POST /syncRule/delete?id=` |
| `syncRule/pause` | `POST /syncRule/pause?id=` |
| `syncRule/resume` | `POST /syncRule/resume?id=` |
| `syncRule/trigger` | `POST /syncRule/trigger?id=` |
| `syncTask/list` | `GET /syncTask/list?status=&ruleID=&owner=` |
| `syncTask/get` | `GET /syncTask/get?id=` |
| `syncTask/cancel` | `POST /syncTask/cancel?id=` |
| `syncTask/retry` | `POST /syncTask/retry?id=` |
| `syncTask/export` | `GET /syncTask/export?since=` |

#### syncnode local

| dashboard API | syncnode API |
|---|---|
| `syncNode/version` | `GET /admin/syncnode/version` |
| `syncNode/stat` | `GET /admin/syncnode/stat` |
| `syncNode/reload` | `POST /admin/syncnode/reload` |

### 9.5 请求体设计

#### dashboard POST

推荐 body：

```json
{
  "addr": "10.0.1.10:17910",
  "id": "rule-or-task-id",
  "force": false,
  "config": {}
}
```

具体约定：

- `syncNode/decommission`: `{ "addr": "...", "force": false }`
- `syncNode/drain|restore`: `{ "addr": "..." }`
- `syncNode/reload`: `{ "addr": "..." }`
- `syncRule/create|update`: body 直接是 flat `SyncRuleConfig`
- `syncRule/delete|pause|resume|trigger`: `{ "id": "rule-id" }`
- `syncTask/cancel|retry`: `{ "id": "task-id" }`

注意：

- `syncRule/create|update` 不允许再包装成 `{ config: ... }`
- 这点必须和旧方案区分清楚

### 9.6 协议归一化

统一成功标准：

- 上游成功：`body.code == 0`
- dashboard 成功：`body.code == 200`

统一失败映射：

| 上游来源 | 上游 code / http | dashboard code |
|---|---|---|
| ParamError | `code=2` | `400` |
| Not found | `code=1` 且 `msg` 含 `not found` | `404` |
| Conflict | `code=1014/1015/1016` | `409` |
| Raft unavailable | `code=4` / HTTP 503 | `503` |
| Unauthorized | HTTP 401/403 | `401/403` |
| Node local reload failed | HTTP 500 | `510` |
| 网络失败 / 超时 | N/A | `510` |
| 其他上游内部错误 | `code=1` | `510` |

保留原始上下文：

```json
{
  "code": 503,
  "msg": "raft submit failed: leader unavailable",
  "data": {
    "upstream": "master",
    "sync_code": 4
  }
}
```

### 9.7 认证头注入

对 master 与 syncnode local 都采用同一注入策略：

1. cluster 上未配置 `SyncAdminToken` 时不注入
2. 默认注入 `X-Sync-Token: <token>`
3. 后续若上游强制 Bearer，再切到 `Authorization: Bearer <token>`

### 9.8 导出透传

`syncTask/export` 特殊处理：

- backend 不包 JSON
- 透传 `Content-Type: application/x-ndjson`
- 透传 `Content-Disposition`
- 前端用下载方式触发

## 10. 前端设计

### 10.1 路由

新增：

```js
{
  path: 'syncManage',
  name: 'syncManage',
  meta: { title: 'router.syncmanage' },
  component: () => import('@/pages/cfs/clusterOverview/clusterInfo/syncManage/index.vue'),
}
```

### 10.2 目录建议

```text
frontend/src/pages/cfs/clusterOverview/clusterInfo/syncManage/
  index.vue
  components/
    SyncRuleTab.vue
    SyncRuleEditorDialog.vue
    SyncTaskTab.vue
    SyncTaskDrawer.vue
    SyncWorkerPanel.vue
    SyncWorkerDrawer.vue
```

### 10.3 SyncWorkerPanel

该面板由页面头部 `执行器维护` 按钮打开，不作为一级 tab。

列表只调用 master `/syncNode/list`。

行操作：

- `诊断`
- `查看任务`
- `进入维护`
- `永久下线`
- `恢复服务`

`永久下线` 的确认弹窗必须提供：

- `平滑下线`
  - 调用 `syncNode/decommission` with `force=false`
- `强制下线`
  - 调用 `syncNode/decommission` with `force=true`

当用户点开某个 worker 的操作确认框或诊断抽屉时，可额外请求：

- `syncNode/tasks`

用途：

- 展示当前节点受影响任务摘要
- 支撑 `drain / decommission` 的确认提示
- 不作为独立 tab 或独立页面

`诊断` 抽屉打开时再按需请求：

- `syncNode/version`
- `syncNode/stat`

`reload` 按钮仅放在诊断抽屉中。

### 10.4 SyncRuleTab

列表和详情完全 cluster-global。

创建/编辑首版仍用 JSON 编辑器，但必须遵循 flat shape：

```json
{
  "id": "r1",
  "type": "sync",
  "schedule": "*/30 * * * * *",
  "src": { "kind": "local", "path": "/srv/data/" },
  "dst": { "kind": "s3", "endpoint": "...", "bucket": "backup" }
}
```

操作：

- `create`
- `update`
- `delete`
- `pause`
- `resume`
- `trigger`

### 10.5 SyncTaskTab

列表完全 cluster-global。

支持过滤：

- `status`
- `ruleID`
- `owner`

支持操作：

- `detail`
- `cancel`
- `retry`
- `export`

不提供：

- `save`
- `load`
- `trigger`

因为这些触发语义已收敛到 `syncRule/trigger`。

## 11. 权限、日志与国际化

### 11.1 新增权限码

#### sync node

- `CFS_SYNCNODE_LIST`
- `CFS_SYNCNODE_DECOMMISSION`
- `CFS_SYNCNODE_DRAIN`
- `CFS_SYNCNODE_RESTORE`
- `CFS_SYNCNODE_VERSION`
- `CFS_SYNCNODE_STAT`
- `CFS_SYNCNODE_RELOAD`

#### sync rule

- `CFS_SYNCRULE_LIST`
- `CFS_SYNCRULE_GET`
- `CFS_SYNCRULE_CREATE`
- `CFS_SYNCRULE_UPDATE`
- `CFS_SYNCRULE_DELETE`
- `CFS_SYNCRULE_PAUSE`
- `CFS_SYNCRULE_RESUME`
- `CFS_SYNCRULE_TRIGGER`

#### sync task

- `CFS_SYNCTASK_LIST`
- `CFS_SYNCTASK_GET`
- `CFS_SYNCTASK_CANCEL`
- `CFS_SYNCTASK_RETRY`
- `CFS_SYNCTASK_EXPORT`

### 11.2 默认角色建议

- `Admin`：全部
- `Operator`：全部
- `Viewer`：
  - `CFS_SYNCNODE_LIST`
  - `CFS_SYNCNODE_VERSION`
  - `CFS_SYNCNODE_STAT`
  - `CFS_SYNCRULE_LIST`
  - `CFS_SYNCRULE_GET`
  - `CFS_SYNCTASK_LIST`
  - `CFS_SYNCTASK_GET`
  - `CFS_SYNCTASK_EXPORT`

### 11.3 操作日志

建议记录：

- syncnode `decommission/drain/restore/reload`
- syncrule `create/update/delete/pause/resume/trigger`
- synctask `cancel/retry`

查询类不记日志。

### 11.4 i18n

需要新增：

- `router.syncmanage`
- `common.sync`
- `common.syncnode`
- `common.syncrule`
- `common.synctask`
- `common.diagnostics`
- `common.decommission`
- `common.drain`
- `common.restore`
- `common.trigger`
- rule state / task status / worker 诊断文案

## 12. 实施分期与验收标准

### Phase A: 数据模型与后端骨架

范围：

- cluster 字段扩展
- migration
- sync master/node client
- 路由挂载

AC：

- migration 成功
- cluster create/update 可保存 `SyncNodeHTTPPort` 与 `SyncAdminToken`
- 新 API 路由可访问

### Phase B: master-backed node/rule/task

范围：

- `syncNode/list/tasks/decommission/drain/restore`
- `syncRule/*`
- `syncTask/*`

AC：

- 列表、详情、变更请求都通过 master API 完成
- `code=4` 能被识别为可重试错误
- `1014/1015/1016` 能以冲突错误返回前端
- `Workers -> view tasks` 只表现为 `Tasks` 页 owner 过滤跳转
- `drain/decommission(force=false|true)/restore` 的确认文案和后置刷新行为与上游语义一致

### Phase C: node-local diagnostics

范围：

- `syncNode/version/stat/reload`
- 节点诊断抽屉

AC：

- 通过 `addr + SyncNodeHTTPPort` 能正确访问本地诊断接口
- reload 失败时返回明确错误
- 诊断页明确展示“rules ignored / scheduledRules=0”

### Phase D: 权限、日志、打磨

范围：

- default permissions
- op log
- i18n
- 前端交互细节

AC：

- `v-auth` 正常生效
- 关键写操作进入 op log
- 中英文切换无缺词

## 13. 测试策略

### 13.1 后端单元测试

- leader resolver
- node admin addr resolver
- master error -> dashboard code mapping
- syncnode local error -> dashboard code mapping

### 13.2 后端集成测试

- mock master server 覆盖 `/syncRule/*`、`/syncTask/*`、`/syncNode/*`
- mock syncnode server 覆盖 `/admin/syncnode/*`
- 验证 `export` NDJSON 透传
- 验证 `drain / decommission(force=false|true) / restore` 的请求参数与返回映射

### 13.3 前端回归测试

- `Rules` 页面 create/update/delete/trigger/view tasks
- `Tasks` 页面 filter/cancel/retry/export/owner->worker 跳转
- `Workers` 维护面板 diagnostics/decommission/drain/restore/view tasks
- `Workers` 维护面板中的任务影响面展示与确认弹窗
- 无权限场景

## 14. 风险与未决项

### 14.1 风险

#### R1: master `SyncNodeInfo` 不含 admin addr

影响：

- dashboard 只能用 `host + SyncNodeHTTPPort` 推导本地诊断地址

应对：

- v1 明确要求 cluster 内 syncnode admin port 一致
- 后续若 master schema 增加 `adminAddr`，优先使用它

#### R2: sync admin token 与 dashboard 会话体系割裂

影响：

- 需要额外保存一个 cluster 级 token

应对：

- v1 存在 `Cluster` 模型中并加密保存
- 后续若 master/syncnode 能复用 dashboard session，再收敛

#### R3: raft leader 切换导致写操作短时失败

影响：

- create/update/delete/pause/resume/trigger/decommission 等写操作会收到 `code=4`

应对：

- backend 返回 `503`
- 前端对特定操作提供“可重试”提示

### 14.2 未决项

1. sync admin token 是否保证 master 与 syncnode 复用同一份
2. `syncNodeInfo.version` 是否足够，是否还需要在 node list 上补一次本地 `version` fan-out
3. 是否需要在 cluster overview 顶部增加 Sync 总览卡片
4. 后续是否要为 `syncRuleConfig` 提供表单式编辑器，替代 JSON 编辑器

## 15. 最终边界

本设计落地后，dashboard 的职责是：

- 用 master 作为 Sync 子系统的主控制面
- 用 syncnode 作为节点本地诊断面
- 统一封装两类上游协议
- 提供 cluster-global 的规则与任务管理能力

本设计明确不再支持：

- 节点本地 rule CRUD
- 节点本地 task CRUD
- 以 syncnode 为权威存储的旧控制模型
