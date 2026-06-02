-- Seed default report mappings for small_business_gaap_cn

-- Profit Statement (利润表)
INSERT INTO report_mappings (id, report_type, line_code, line_label, display_order, accounting_standard, account_selector, is_subtotal, parent_line_code, formula)
VALUES
  ('ps_revenue',       'profit_statement', 'revenue',           '一、营业收入',           1, 'small_business_gaap_cn', '{"prefixes": ["6"]}', false, '', ''),
  ('ps_cost',          'profit_statement', 'cost',              '减：营业成本',           2, 'small_business_gaap_cn', '{"prefixes": ["5401", "5402"]}', false, '', ''),
  ('ps_tax_surcharge', 'profit_statement', 'tax_and_surcharge', '税金及附加',            3, 'small_business_gaap_cn', '{"prefixes": ["5403"]}', false, '', ''),
  ('ps_selling',       'profit_statement', 'selling_expense',   '销售费用',              4, 'small_business_gaap_cn', '{"prefixes": ["5601"]}', false, '', ''),
  ('ps_admin',         'profit_statement', 'admin_expense',     '管理费用',              5, 'small_business_gaap_cn', '{"prefixes": ["5602"]}', false, '', ''),
  ('ps_finance',       'profit_statement', 'finance_expense',   '财务费用',              6, 'small_business_gaap_cn', '{"prefixes": ["5603"]}', false, '', ''),
  ('ps_invest_income', 'profit_statement', 'investment_income', '加：投资收益',           7, 'small_business_gaap_cn', '{"prefixes": ["6111"]}', false, '', ''),
  ('ps_operating',     'profit_statement', 'operating_profit',  '二、营业利润',           8, 'small_business_gaap_cn', '{}', true, '', 'revenue - cost - tax_and_surcharge - selling_expense - admin_expense - finance_expense + investment_income'),
  ('ps_nonop_income',  'profit_statement', 'non_op_income',     '加：营业外收入',         9, 'small_business_gaap_cn', '{"prefixes": ["6301"]}', false, '', ''),
  ('ps_nonop_expense', 'profit_statement', 'non_op_expense',    '减：营业外支出',        10, 'small_business_gaap_cn', '{"prefixes": ["6711"]}', false, '', ''),
  ('ps_total_profit',  'profit_statement', 'total_profit',      '三、利润总额',          11, 'small_business_gaap_cn', '{}', true, '', 'operating_profit + non_op_income - non_op_expense'),
  ('ps_income_tax',    'profit_statement', 'income_tax',        '减：所得税费用',        12, 'small_business_gaap_cn', '{"prefixes": ["6801"]}', false, '', ''),
  ('ps_net_profit',    'profit_statement', 'net_profit',        '四、净利润',            13, 'small_business_gaap_cn', '{}', true, '', 'total_profit - income_tax')
ON CONFLICT (id) DO NOTHING;

-- Balance Sheet (资产负债表)
INSERT INTO report_mappings (id, report_type, line_code, line_label, display_order, accounting_standard, account_selector, is_subtotal, parent_line_code, formula)
VALUES
  -- Current Assets
  ('bs_monetary',        'balance_sheet', 'monetary_funds',     '货币资金',              1, 'small_business_gaap_cn', '{"prefixes": ["1001", "1002"]}', false, '', ''),
  ('bs_ar',              'balance_sheet', 'accounts_receivable', '应收账款',              2, 'small_business_gaap_cn', '{"prefixes": ["1122", "1131", "1132"]}', false, '', ''),
  ('bs_prepay',          'balance_sheet', 'prepayments',        '预付账款',              3, 'small_business_gaap_cn', '{"prefixes": ["1123"]}', false, '', ''),
  ('bs_other_recv',      'balance_sheet', 'other_receivables',  '其他应收款',            4, 'small_business_gaap_cn', '{"prefixes": ["1221"]}', false, '', ''),
  ('bs_inventory',       'balance_sheet', 'inventory',          '存货',                  5, 'small_business_gaap_cn', '{"prefixes": ["1403", "1405", "1411"]}', false, '', ''),
  ('bs_current_assets',  'balance_sheet', 'current_assets',     '流动资产合计',          6, 'small_business_gaap_cn', '{}', true, '', 'monetary_funds + accounts_receivable + prepayments + other_receivables + inventory'),
  -- Non-Current Assets
  ('bs_fixed_assets',    'balance_sheet', 'fixed_assets',       '固定资产',              7, 'small_business_gaap_cn', '{"prefixes": ["1601"]}', false, '', ''),
  ('bs_intangible',      'balance_sheet', 'intangible_assets',  '无形资产',              8, 'small_business_gaap_cn', '{"prefixes": ["1701"]}', false, '', ''),
  ('bs_noncurr_assets',  'balance_sheet', 'non_current_assets', '非流动资产合计',        9, 'small_business_gaap_cn', '{}', true, '', 'fixed_assets + intangible_assets'),
  ('bs_total_assets',    'balance_sheet', 'total_assets',       '资产总计',             10, 'small_business_gaap_cn', '{}', true, '', 'current_assets + non_current_assets'),
  -- Current Liabilities
  ('bs_st_borrow',       'balance_sheet', 'short_term_borrow',  '短期借款',             11, 'small_business_gaap_cn', '{"prefixes": ["2001"]}', false, '', ''),
  ('bs_ap',              'balance_sheet', 'accounts_payable',   '应付账款',             12, 'small_business_gaap_cn', '{"prefixes": ["2202"]}', false, '', ''),
  ('bs_advance',         'balance_sheet', 'advance_receipts',   '预收账款',             13, 'small_business_gaap_cn', '{"prefixes": ["2203"]}', false, '', ''),
  ('bs_payroll',         'balance_sheet', 'payroll_payable',    '应付职工薪酬',         14, 'small_business_gaap_cn', '{"prefixes": ["2211"]}', false, '', ''),
  ('bs_tax_payable',     'balance_sheet', 'taxes_payable',      '应交税费',             15, 'small_business_gaap_cn', '{"prefixes": ["2221"]}', false, '', ''),
  ('bs_current_liab',    'balance_sheet', 'current_liabilities', '流动负债合计',         16, 'small_business_gaap_cn', '{}', true, '', 'short_term_borrow + accounts_payable + advance_receipts + payroll_payable + taxes_payable'),
  -- Non-Current Liabilities
  ('bs_lt_borrow',       'balance_sheet', 'long_term_borrow',   '长期借款',             17, 'small_business_gaap_cn', '{"prefixes": ["2501"]}', false, '', ''),
  ('bs_noncurr_liab',    'balance_sheet', 'non_current_liab',   '非流动负债合计',       18, 'small_business_gaap_cn', '{}', true, '', 'long_term_borrow'),
  ('bs_total_liab',      'balance_sheet', 'total_liabilities',  '负债合计',             19, 'small_business_gaap_cn', '{}', true, '', 'current_liabilities + non_current_liab'),
  -- Equity
  ('bs_paid_in',         'balance_sheet', 'paid_in_capital',    '实收资本',             20, 'small_business_gaap_cn', '{"prefixes": ["4001"]}', false, '', ''),
  ('bs_cap_reserve',     'balance_sheet', 'capital_reserve',    '资本公积',             21, 'small_business_gaap_cn', '{"prefixes": ["4002"]}', false, '', ''),
  ('bs_surplus',         'balance_sheet', 'surplus_reserve',    '盈余公积',             22, 'small_business_gaap_cn', '{"prefixes": ["4101"]}', false, '', ''),
  ('bs_retained',        'balance_sheet', 'retained_earnings',  '未分配利润',           23, 'small_business_gaap_cn', '{"prefixes": ["4104"]}', false, '', ''),
  ('bs_total_equity',    'balance_sheet', 'total_equity',       '所有者权益合计',       24, 'small_business_gaap_cn', '{}', true, '', 'paid_in_capital + capital_reserve + surplus_reserve + retained_earnings'),
  ('bs_total_liab_eq',   'balance_sheet', 'total_liab_equity',  '负债及所有者权益总计', 25, 'small_business_gaap_cn', '{}', true, '', 'total_liabilities + total_equity')
ON CONFLICT (id) DO NOTHING;
