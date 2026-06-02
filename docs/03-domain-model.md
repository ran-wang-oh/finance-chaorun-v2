# v2 Finance Provider 领域模型

## 1. 设计原则

Finance Provider 的数据模型从第一天开始使用：

```text
entity_id + book_id
```

不使用旧 `tenant_id`。

基本关系：

```text
entity
  -> accounting book
    -> accounting period
    -> invoice
    -> journal entry
    -> tax profile
    -> tax return
    -> report snapshot
```

## 2. 核心对象

### 2.1 Accounting Book

账本是 finance 领域内的核算边界。

建议字段：

```sql
CREATE TABLE accounting_books (
  id                  TEXT PRIMARY KEY,
  entity_id           TEXT NOT NULL,
  code                TEXT NOT NULL,
  name                TEXT NOT NULL,
  accounting_standard TEXT NOT NULL,
  base_currency       TEXT NOT NULL DEFAULT 'CNY',
  start_period        TEXT NOT NULL,
  is_default          BOOLEAN NOT NULL DEFAULT false,
  status              TEXT NOT NULL DEFAULT 'active',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_id, code)
);
```

规则：

- 每个 entity 可以有多本账。
- 每个 entity 最多一本 default book。
- 所有下游对象必须归属于 `entity_id + book_id`。

### 2.2 Accounting Period

```sql
CREATE TABLE accounting_periods (
  id          TEXT PRIMARY KEY,
  entity_id   TEXT NOT NULL,
  book_id     TEXT NOT NULL,
  period      TEXT NOT NULL,
  status      TEXT NOT NULL,
  opened_at   TIMESTAMPTZ,
  closing_at  TIMESTAMPTZ,
  closed_at   TIMESTAMPTZ,
  locked_at   TIMESTAMPTZ,
  closed_by   TEXT,
  metadata    JSONB NOT NULL DEFAULT '{}',
  UNIQUE (entity_id, book_id, period)
);
```

状态建议：

```text
open
closing
closed
locked
```

规则：

- `locked` 后禁止影响该期间的写操作。
- `closed` 后只允许受控重开或调整流程。
- `close` 和 `lock` 是 `committed_write`。
- `reopen` 是 `destructive`。

### 2.3 Invoice

```sql
CREATE TABLE invoices (
  id             TEXT PRIMARY KEY,
  entity_id      TEXT NOT NULL,
  book_id        TEXT NOT NULL,
  invoice_no     TEXT NOT NULL,
  invoice_type   TEXT NOT NULL,
  direction      TEXT NOT NULL,
  issue_date     DATE NOT NULL,
  seller_name    TEXT,
  seller_tax_no  TEXT,
  buyer_name     TEXT,
  buyer_tax_no   TEXT,
  amount         NUMERIC(18,2) NOT NULL,
  tax_amount     NUMERIC(18,2) NOT NULL,
  total_amount   NUMERIC(18,2) NOT NULL,
  currency       TEXT NOT NULL DEFAULT 'CNY',
  status         TEXT NOT NULL,
  source         TEXT NOT NULL,
  extraction     JSONB NOT NULL DEFAULT '{}',
  evidence_refs  JSONB NOT NULL DEFAULT '[]',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_id, book_id, invoice_no)
);
```

状态建议：

```text
draft
pending_review
approved
rejected
posted
voided
red_lettered
```

规则：

- 创建草稿是 `draft_write`。
- 审核、驳回、验真结果落库是 `committed_write`。
- 作废、红冲可能是 `destructive` 或高风险 `committed_write`。

### 2.4 Invoice Line

```sql
CREATE TABLE invoice_lines (
  id             TEXT PRIMARY KEY,
  entity_id      TEXT NOT NULL,
  invoice_id     TEXT NOT NULL,
  line_no        INTEGER NOT NULL,
  item_name      TEXT NOT NULL,
  item_code      TEXT,
  quantity       NUMERIC(18,4),
  unit_price     NUMERIC(18,4),
  amount         NUMERIC(18,2) NOT NULL,
  tax_rate       NUMERIC(8,4) NOT NULL,
  tax_amount     NUMERIC(18,2) NOT NULL,
  metadata       JSONB NOT NULL DEFAULT '{}',
  UNIQUE (entity_id, invoice_id, line_no)
);
```

发票明细用于：

- 税率校验。
- 商品服务编码校验。
- 进销项品类一致性。
- 金税四期风险扫描。

### 2.5 Chart of Accounts

```sql
CREATE TABLE chart_of_accounts (
  id             TEXT PRIMARY KEY,
  entity_id      TEXT NOT NULL,
  book_id        TEXT NOT NULL,
  code           TEXT NOT NULL,
  name           TEXT NOT NULL,
  category       TEXT NOT NULL,
  balance_type   TEXT NOT NULL,
  parent_id      TEXT,
  is_system      BOOLEAN NOT NULL DEFAULT false,
  tax_relevant   BOOLEAN NOT NULL DEFAULT false,
  keywords       TEXT[] NOT NULL DEFAULT '{}',
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_id, book_id, code)
);
```

### 2.6 Journal Entry

```sql
CREATE TABLE journal_entries (
  id              TEXT PRIMARY KEY,
  entity_id       TEXT NOT NULL,
  book_id         TEXT NOT NULL,
  period          TEXT NOT NULL,
  voucher_no      TEXT,
  voucher_word    TEXT NOT NULL DEFAULT '记',
  entry_date      DATE NOT NULL,
  summary         TEXT NOT NULL,
  source_type     TEXT,
  source_id       TEXT,
  status          TEXT NOT NULL,
  created_by      TEXT,
  posted_by       TEXT,
  posted_at       TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (entity_id, book_id, period, voucher_no)
);
```

状态建议：

```text
draft
pending_review
posted
voided
```

### 2.7 Journal Line

```sql
CREATE TABLE journal_lines (
  id                TEXT PRIMARY KEY,
  entity_id         TEXT NOT NULL,
  journal_entry_id  TEXT NOT NULL,
  account_id        TEXT NOT NULL,
  debit_amount      NUMERIC(18,2) NOT NULL DEFAULT 0,
  credit_amount     NUMERIC(18,2) NOT NULL DEFAULT 0,
  currency          TEXT NOT NULL DEFAULT 'CNY',
  line_no           INTEGER NOT NULL,
  auxiliary         JSONB NOT NULL DEFAULT '{}',
  UNIQUE (entity_id, journal_entry_id, line_no)
);
```

规则：

- 每张凭证借贷必须平衡。
- 过账前必须校验期间状态。
- 过账前必须校验账户归属同一 `entity_id + book_id`。

### 2.8 Tax Profile

```sql
CREATE TABLE tax_profiles (
  id                 TEXT PRIMARY KEY,
  entity_id          TEXT NOT NULL,
  book_id            TEXT NOT NULL,
  taxpayer_type      TEXT NOT NULL,
  vat_taxpayer_type  TEXT NOT NULL,
  cit_rate_type      TEXT NOT NULL,
  effective_from     DATE NOT NULL,
  effective_to       DATE,
  metadata           JSONB NOT NULL DEFAULT '{}',
  UNIQUE (entity_id, book_id)
);
```

### 2.9 Risk Scan

```sql
CREATE TABLE risk_scans (
  id            TEXT PRIMARY KEY,
  entity_id     TEXT NOT NULL,
  book_id       TEXT NOT NULL,
  period        TEXT NOT NULL,
  status        TEXT NOT NULL,
  summary       JSONB NOT NULL DEFAULT '{}',
  findings      JSONB NOT NULL DEFAULT '[]',
  evidence_refs JSONB NOT NULL DEFAULT '[]',
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 2.10 Domain Audit

```sql
CREATE TABLE finance_audit_log (
  id                  TEXT PRIMARY KEY,
  entity_id           TEXT NOT NULL,
  book_id             TEXT,
  capability_id       TEXT NOT NULL,
  v2_capability_id    TEXT,
  actor_type          TEXT NOT NULL,
  actor_id            TEXT NOT NULL,
  trace_id            TEXT NOT NULL,
  workflow_run_id     TEXT,
  approval_grant_id   TEXT,
  idempotency_key     TEXT,
  object_type         TEXT,
  object_id           TEXT,
  action              TEXT NOT NULL,
  outcome             TEXT NOT NULL,
  payload             JSONB NOT NULL DEFAULT '{}',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

## 3. 幂等模型

```sql
CREATE TABLE idempotency_records (
  entity_id        TEXT NOT NULL,
  capability_id    TEXT NOT NULL,
  idempotency_key  TEXT NOT NULL,
  input_hash       TEXT NOT NULL,
  result           JSONB NOT NULL,
  status           TEXT NOT NULL,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (entity_id, capability_id, idempotency_key)
);
```

规则：

- 相同 key + 相同 input hash：返回同一结果。
- 相同 key + 不同 input hash：返回幂等冲突。
- 幂等记录必须和 finance domain audit 可关联。

## 4. 领域不变量

必须始终成立：

- 任意 finance object 只属于一个 `entity_id`。
- 任意 `book_id` 只属于一个 `entity_id`。
- 任意凭证行的账户必须属于同一账本。
- 凭证过账时借贷必须平衡。
- 锁定期间不可写。
- 已关闭期间只能走受控调整或重开。
- 税务计算使用确定性规则。
- LLM 输出不能直接成为最终税额或过账依据。

## 5. MVP 数据范围

首版建议只实现：

- accounting_books
- accounting_periods
- invoices
- invoice_lines
- chart_of_accounts
- journal_entries
- journal_lines
- finance_audit_log
- idempotency_records

报表和税务可先从凭证、科目和发票实时计算，再逐步增加 snapshot 表。
