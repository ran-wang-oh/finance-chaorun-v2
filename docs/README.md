# v2 Finance Provider 开发文档包

本目录用于辅助从零开发 `chao.run v2` 原生 Finance Provider。

设计结论：

```text
参考旧 finance 的业务模型和规则。
不要沿用旧 finance 的 Agent Runtime、tenant_id、多平台依赖和旧 control plane。
```

Finance Provider 在 v2 中的定位：

```text
Finance Domain Provider
  + Finance Rule Engine
  + Finance Executor
  + Finance Domain Audit
```

它只负责财务领域能力，不拥有平台运行时。

## 文档清单

- [01-architecture.md](01-architecture.md)：v2 Finance Provider 架构定位、边界和模块拆分。
- [02-provider-contract.md](02-provider-contract.md)：Provider 对 v2 Capability Bus 暴露的接口合约。
- [03-domain-model.md](03-domain-model.md)：`entity_id + book_id` 财务领域数据模型。
- [04-development-checklist.md](04-development-checklist.md)：开发阶段、验收标准和测试清单。

## 核心边界

### v2 平台负责

- Router
- Registry
- Policy
- Approval
- Context Engine
- Capability Bus
- Workflow Runner
- Platform Audit
- Model Provider Governance

### Finance Provider 负责

- Finance capability discovery
- Finance context facts
- Preview
- Validate
- Execute
- Deterministic finance calculation
- Finance domain audit
- Finance resource refs

### 禁止事项

- 不在 Finance Provider 内实现 Agent Runtime。
- 不在 Finance Provider 内实现 Planner。
- 不在 Finance Provider 内实现 Proposal lifecycle。
- 不在 v2 API 或 Provider public contract 中暴露 `tenant_id`。
- 不让 LLM 直接决定税额、申报栏次、过账、关账、审批或 destructive 操作。
- 不让 Agent 直接调用数据库、REST route、MCP server 或 provider internals。

## 推荐开发顺序

1. 实现最小 Provider Contract。
2. 实现 capability catalog。
3. 实现 `entity_id + book_id` 数据模型。
4. 实现 invoice draft 和 journal draft。
5. 实现 approval-gated journal post。
6. 实现 report read capability。
7. 实现 month-end close workflow 所需能力。
8. 接入 v2 audit、policy、idempotency 和 resource refs。
