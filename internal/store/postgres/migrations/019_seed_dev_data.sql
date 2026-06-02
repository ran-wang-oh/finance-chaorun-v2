-- Seed default dev data
-- Uses ON CONFLICT DO NOTHING so it's safe to re-run on every migration pass

-- Default accounting book for entity "default"
INSERT INTO accounting_books (id, entity_id, code, name, accounting_standard, base_currency, start_period, is_default, status)
VALUES ('book-default', 'default', 'default', '默认账套', 'small_business_gaap_cn', 'CNY', '2025-01', true, 'active')
ON CONFLICT (entity_id, code) DO NOTHING;

-- Standard chart of accounts (Chinese GAAP, small business)
INSERT INTO chart_of_accounts (id, entity_id, book_id, code, name, category, balance_type, is_system, keywords)
VALUES
  (gen_random_uuid(), 'default', 'book-default', '1001', '库存现金', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1002', '银行存款', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1012', '其他货币资金', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1122', '应收账款', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1123', '预付账款', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1131', '应收股利', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1132', '应收利息', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1221', '其他应收款', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1403', '原材料', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1405', '库存商品', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1411', '周转材料', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1601', '固定资产', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1602', '累计折旧', 'asset', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1701', '无形资产', 'asset', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '1702', '累计摊销', 'asset', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2001', '短期借款', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2202', '应付账款', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2203', '预收账款', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2211', '应付职工薪酬', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2221', '应交税费', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2221-01', '应交税费-应交增值税-进项税额', 'liability', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2221-02', '应交税费-应交增值税-销项税额', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2221-03', '应交税费-应交增值税-已交税金', 'liability', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2221-06', '应交税费-应交城市维护建设税', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2221-07', '应交税费-应交教育费附加', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2221-08', '应交税费-应交地方教育附加', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2221-09', '应交税费-应交企业所得税', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2221-12', '应交税费-应交印花税', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2241', '其他应付款', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '2501', '长期借款', 'liability', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '4001', '实收资本', 'equity', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '4002', '资本公积', 'equity', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '4101', '盈余公积', 'equity', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '4104', '利润分配-未分配利润', 'equity', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '5401', '主营业务成本', 'cost', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '5402', '其他业务成本', 'cost', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '5403', '税金及附加', 'cost', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '5601', '销售费用', 'expense', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '5602', '管理费用', 'expense', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '5603', '财务费用', 'expense', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '6001', '主营业务收入', 'revenue', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '6051', '其他业务收入', 'revenue', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '6111', '投资收益', 'revenue', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '6301', '营业外收入', 'revenue', 'credit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '6711', '营业外支出', 'expense', 'debit', true, '{}'),
  (gen_random_uuid(), 'default', 'book-default', '6801', '所得税费用', 'expense', 'debit', true, '{}')
ON CONFLICT (entity_id, book_id, code) DO NOTHING;
