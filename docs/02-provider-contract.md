# v2 Finance Provider 合约

## 1. 目标

本文定义 Finance Provider 与 `chao.run v2` Connector Adapter / Capability Bus 之间的接口合约。

合约目标：

- v2 只通过 capability 调用 Finance Provider。
- Finance Provider 不暴露旧 `tenant_id`。
- Finance Provider 对所有调用执行 entity scope 校验。
- 写操作支持幂等。
- 输出可审计、可追踪、可被 v2 schema validation 验证。

## 2. 基础原则

### 2.1 必填字段

每个 Provider 请求必须包含：

```text
entity_id
capability_id
actor
trace_id
```

写操作还必须包含：

```text
idempotency_key
```

`committed_write` 和 `destructive` 必须由 v2 在调用前完成 approval grant 校验。

### 2.2 Provider 不做平台决策

Provider 可以做领域校验，例如：

- 账本是否存在
- 期间是否锁定
- 凭证是否平衡
- 发票是否重复
- 税务规则是否满足

Provider 不做平台决策，例如：

- 用户是否有权限
- capability 是否已发布
- capability hash 是否可信
- 是否需要审批
- approval grant 是否有效
- 是否允许跨 entity

这些由 v2 负责。

## 3. Endpoint 设计

如果 Finance Provider 以 HTTP 服务实现，建议接口：

```text
GET  /healthz
GET  /readyz
GET  /v1/capabilities
POST /v1/context
POST /v1/capabilities/{capability_id}/preview
POST /v1/capabilities/{capability_id}/validate
POST /v1/capabilities/{capability_id}/execute
GET  /v1/resources/{resource_uri}
```

这些 endpoint 是 Provider 内部接口，不是 v2 public API。

v2 public API 仍然是：

```text
/api/v1/entities/{entity_id}/capabilities/{capability_id}/invoke
```

## 4. Request Envelope

```json
{
  "entity_id": "entity_cn_main",
  "capability_id": "finance.journal.post",
  "actor": {
    "type": "user",
    "id": "user_123"
  },
  "input": {
    "book_id": "book_default",
    "journal_entry_id": "je_123"
  },
  "idempotency_key": "idem_123",
  "trace_id": "trace_123",
  "dry_run": false,
  "metadata": {
    "v2_capability_id": "connector.finance.journal.post",
    "workflow_run_id": "wf_123",
    "approval_grant_id": "grant_123"
  }
}
```

字段说明：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `entity_id` | 是 | v2 entity scope |
| `capability_id` | 是 | Finance Provider 内部 action name |
| `actor` | 是 | v2 传入的执行主体 |
| `input` | 是 | capability 输入 |
| `idempotency_key` | 写操作必填 | 防止重复写 |
| `trace_id` | 是 | v2 trace / run correlation |
| `dry_run` | 否 | 只预览不写入 |
| `metadata` | 否 | v2 上下文引用 |

## 5. Response Envelope

```json
{
  "status": "ok",
  "data": {
    "journal_entry_id": "je_123",
    "status": "posted",
    "voucher_no": "记-2026-05-001"
  },
  "resource_refs": [
    "finance://journal-entry/je_123"
  ],
  "external_request_id": "fin_req_123",
  "domain_audit_ref": "finance-audit://audit_123",
  "warnings": []
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `status` | `ok` / `failed` |
| `data` | 小型结构化结果 |
| `resource_refs` | 大对象或可追溯对象引用 |
| `external_request_id` | Provider 请求 ID |
| `domain_audit_ref` | Finance 领域审计引用 |
| `warnings` | 非阻断性提示 |

## 6. Capability Descriptor

Provider capability 示例：

```yaml
name: finance.journal.post
domain: finance
version: 0.1.0
side_effect: committed_write
requires_idempotency_key: true
supports_dry_run: false
input_schema:
  type: object
  required:
    - book_id
    - journal_entry_id
  properties:
    book_id:
      type: string
    journal_entry_id:
      type: string
output_schema:
  type: object
  required:
    - journal_entry_id
    - status
  properties:
    journal_entry_id:
      type: string
    status:
      type: string
    voucher_no:
      type: string
```

v2 Registry 中对应 artifact：

```yaml
id: connector.finance.journal.post
kind: connector
name: finance.journal.post
domain: finance
version: 0.1.0
side_effect: committed_write
status: published
```

## 7. 初始 Capability 清单

| Capability | Side Effect | 说明 |
| --- | --- | --- |
| `finance.invoice.create_draft` | `draft_write` | 创建发票草稿 |
| `finance.invoice.approve` | `committed_write` | 审核发票 |
| `finance.invoice.reject` | `committed_write` | 驳回发票 |
| `finance.invoice.extract_preview` | `read` | 只抽取不落库 |
| `finance.journal.create_draft` | `draft_write` | 创建凭证草稿 |
| `finance.journal.post` | `committed_write` | 凭证过账 |
| `finance.journal.batch_post` | `committed_write` | 批量过账 |
| `finance.journal.void` | `destructive` | 凭证作废 |
| `finance.report.trial_balance` | `read` | 试算平衡 |
| `finance.report.account_balance` | `read` | 科目余额 |
| `finance.tax.vat_summary` | `read` | VAT 汇总 |
| `finance.tax.risk_scan` | `draft_write` | 生成风险扫描记录 |
| `finance.consistency.check` | `draft_write` | 生成一致性检查记录 |
| `finance.period.close_check` | `read` | 关账前检查 |
| `finance.period.close` | `committed_write` | 关账 |
| `finance.period.lock` | `committed_write` | 锁账 |
| `finance.period.reopen` | `destructive` | 重开期间 |

## 8. Error Contract

Provider 返回错误：

```json
{
  "status": "failed",
  "error": {
    "code": "finance_period_locked",
    "message": "Accounting period is locked.",
    "details": {
      "book_id": "book_default",
      "period": "2026-05"
    }
  },
  "external_request_id": "fin_req_123",
  "domain_audit_ref": "finance-audit://audit_123"
}
```

推荐错误码：

| Code | 含义 |
| --- | --- |
| `finance_book_not_found` | 账本不存在 |
| `finance_book_scope_mismatch` | 账本不属于当前 entity |
| `finance_period_locked` | 期间已锁定 |
| `finance_period_closed` | 期间已关闭 |
| `finance_duplicate_invoice` | 发票重复 |
| `finance_journal_unbalanced` | 凭证借贷不平 |
| `finance_invalid_tax_rule` | 税务规则不满足 |
| `finance_idempotency_conflict` | 幂等冲突 |
| `finance_resource_not_found` | 资源不存在 |
| `finance_validation_failed` | 领域校验失败 |

不得返回：

- SQL 原始错误。
- credential。
- 内部 token。
- 旧 tenant ID。
- 未脱敏敏感字段。

## 9. Resource URI

推荐 URI：

```text
finance://book/{book_id}
finance://period/{book_id}/{period}
finance://invoice/{invoice_id}
finance://journal-entry/{journal_entry_id}
finance://report/trial-balance/{book_id}/{period}
finance://tax-return/{tax_return_id}
finance://risk-scan/{scan_id}
finance://consistency-check/{check_id}
```

Resource 读取必须再次校验 `entity_id`。

## 10. Idempotency

写操作幂等键范围：

```text
entity_id + capability_id + idempotency_key
```

如果同一 key 重放且 input hash 相同，返回同一结果。

如果同一 key 重放但 input hash 不同，返回：

```text
finance_idempotency_conflict
```

## 11. Audit Link

Provider 领域审计必须记录：

```text
entity_id
capability_id
v2_capability_id
actor_type
actor_id
trace_id
workflow_run_id
approval_grant_id
idempotency_key
finance_object_type
finance_object_id
outcome
```
