# v2 Finance Provider 开发检查清单

## 1. 阶段计划

### M0: 项目初始化

目标：建立干净的 v2-native provider 工程。

检查项：

- [ ] 新项目不依赖旧 `agents.chao.run` runtime。
- [ ] 新项目不使用 `tenant_id`。
- [ ] 新项目所有 public/internal contract 使用 `entity_id`。
- [ ] 新项目定义 Provider Contract。
- [ ] 新项目有本地 fake v2 connector 测试入口。

### M1: Capability Catalog

目标：定义 finance provider 可暴露的能力。

检查项：

- [ ] 每个 capability 有稳定 name。
- [ ] 每个 capability 有 input schema。
- [ ] 每个 capability 有 output schema。
- [ ] 每个 capability 有 side effect。
- [ ] 写操作声明 idempotency requirement。
- [ ] destructive 操作单独标记。
- [ ] capability catalog 可被 v2 adapter 读取。

最低能力：

- [ ] `finance.invoice.create_draft`
- [ ] `finance.invoice.approve`
- [ ] `finance.journal.create_draft`
- [ ] `finance.journal.post`
- [ ] `finance.report.trial_balance`
- [ ] `finance.period.close_check`

### M2: Data Model MVP

目标：建立 `entity_id + book_id` 存储模型。

检查项：

- [ ] accounting books 表。
- [ ] accounting periods 表。
- [ ] invoices 表。
- [ ] invoice lines 表。
- [ ] chart of accounts 表。
- [ ] journal entries 表。
- [ ] journal lines 表。
- [ ] finance audit log 表。
- [ ] idempotency records 表。
- [ ] 所有查询包含 `entity_id`。
- [ ] 所有 `book_id` 查询校验 ownership。

### M3: Deterministic Engine

目标：实现财务确定性规则。

检查项：

- [ ] 发票金额校验。
- [ ] 发票重复校验。
- [ ] 凭证借贷平衡校验。
- [ ] 凭证科目归属校验。
- [ ] 期间状态校验。
- [ ] 试算平衡计算。
- [ ] 过账前 validation。
- [ ] 关账前 close check。

### M4: Provider Execution

目标：实现 preview、validate、execute。

检查项：

- [ ] read capability 可执行。
- [ ] draft_write capability 可执行。
- [ ] committed_write capability 只在 v2 approval 后由 adapter 调用。
- [ ] 写操作必须检查 idempotency key。
- [ ] 写操作写入 finance domain audit。
- [ ] 返回 external request ID。
- [ ] 大对象返回 resource refs。

### M5: v2 Integration

目标：接入 chao.run v2 Capability Bus。

检查项：

- [ ] v2 能 list finance capabilities。
- [ ] v2 Registry 中有 published finance artifact。
- [ ] 未发布 artifact 不能执行。
- [ ] deprecated artifact 不能执行。
- [ ] hash drift 不能执行。
- [ ] read capability 无需 approval。
- [ ] committed_write 无 approval 时失败。
- [ ] committed_write 有 approval 时成功。
- [ ] v2 audit 和 finance audit 可互查。

### M6: Workflow MVP

目标：支持 v2 finance workflow。

检查项：

- [ ] 发票到凭证 workflow。
- [ ] workflow 可暂停等待审批。
- [ ] workflow 恢复后不会重复写。
- [ ] workflow timeline 有 finance external request ID。
- [ ] 月结 close check 可作为 read capability 执行。

## 2. 安全检查

每次发布前检查：

- [ ] 没有 `tenant_id` 出现在新 provider contract。
- [ ] 没有跨 entity 查询。
- [ ] 没有 raw SQL error 泄露。
- [ ] 没有 credential 泄露。
- [ ] 没有未脱敏敏感字段进入 audit summary。
- [ ] 写操作都有 idempotency。
- [ ] committed write 都通过 v2 approval。
- [ ] destructive capability 单独声明。
- [ ] connector timeout 已设置。
- [ ] output size limit 已设置。

## 3. 测试清单

### Unit Tests

- [ ] capability catalog schema。
- [ ] input validation。
- [ ] output validation。
- [ ] book ownership check。
- [ ] period locked check。
- [ ] duplicate invoice check。
- [ ] journal balance check。
- [ ] idempotency replay。
- [ ] idempotency conflict。
- [ ] domain audit write。

### Integration Tests

- [ ] Provider list capabilities。
- [ ] Provider context facts。
- [ ] Provider preview。
- [ ] Provider validate。
- [ ] Provider execute read。
- [ ] Provider execute draft_write。
- [ ] Provider execute committed_write。
- [ ] Provider error mapping。
- [ ] Provider resource read。

### v2 E2E Tests

- [ ] Registry unpublished denial。
- [ ] Registry deprecated denial。
- [ ] Policy denial。
- [ ] Approval required。
- [ ] Approval grant success。
- [ ] Cross entity denied。
- [ ] Workflow pause/resume。
- [ ] Audit chain.

## 4. 首个端到端验收场景

输入：

```text
entity_id = entity_cn_main
book_id = book_default
period = 2026-05
```

流程：

```text
1. create invoice draft
2. approve invoice
3. create journal draft
4. request v2 approval
5. post journal
6. query trial balance
```

验收：

- [ ] 发票草稿创建成功。
- [ ] 发票审核成功。
- [ ] 凭证草稿创建成功。
- [ ] 无 approval 不能过账。
- [ ] 有 approval 可以过账。
- [ ] 重放 idempotency key 不重复过账。
- [ ] 试算平衡包含该凭证。
- [ ] v2 audit 有 capability 调用记录。
- [ ] finance audit 有领域执行记录。
- [ ] 两边 audit 通过 trace ID 可关联。

## 5. 不建议做的事

- 不建议先做 UI。
- 不建议先迁移旧 Agent Session。
- 不建议复用旧 `tenant_id` 数据模型。
- 不建议把旧 finance REST API 当作 v2 public API。
- 不建议让 LLM 先直接生成并过账凭证。
- 不建议没有 approval 就实现 committed write。
- 不建议跳过 idempotency 后补。

## 6. 推荐下一步

第一周目标：

```text
Provider Contract + Capability Catalog + Data Model MVP
```

第二周目标：

```text
invoice draft + journal draft + journal post + trial balance
```

第三周目标：

```text
v2 Capability Bus integration + approval + audit + e2e
```
