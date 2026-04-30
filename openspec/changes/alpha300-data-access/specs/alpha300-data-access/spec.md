## ADDED Requirements

### Requirement: Alpha300 数据库连接
系统 SHALL 建立与 Alpha300 数据库的只读连接

#### Scenario: 数据库连接成功
- Given: Alpha300 数据库配置正确
- When: 系统启动
- Then: MUST 成功建立只读连接池

#### Scenario: 连接失败降级
- Given: Alpha300 数据库不可用
- When: 查询数据
- Then: MUST 降级到 EastMoney API 并记录警告

### Requirement: 日K线数据查询
系统 SHALL 从 Alpha300 查询日K线数据

#### Scenario: 查询指定股票K线
- Given: 股票代码 600519.SH
- When: 查询最近 60 天 K 线
- Then: MUST 返回 OHLCV 数据，格式与现有 KlinePoint 兼容

#### Scenario: 批量查询K线
- Given: 多个股票代码
- When: 批量查询
- Then: MUST 并发查询并合并结果

### Requirement: 估值数据查询
系统 SHALL 从 Alpha300 查询估值数据

#### Scenario: 查询 PE/PB 数据
- Given: 股票代码
- When: 查询最新估值
- Then: MUST 返回 pe_ttm, pb, total_mv, circ_mv

#### Scenario: 历史估值查询
- Given: 股票代码和日期范围
- When: 查询历史估值
- Then: MUST 返回指定日期范围的估值数据

### Requirement: 资金流向查询
系统 SHALL 从 Alpha300 查询资金流向数据

#### Scenario: 查询主力资金
- Given: 股票代码
- When: 查询资金流向
- Then: MUST 返回主力净流入/流出金额和占比

#### Scenario: 批量查询资金流向
- Given: 多个股票代码
- When: 批量查询
- Then: MUST 并发查询并合并结果

### Requirement: 行业分类查询
系统 SHALL 从 Alpha300 查询行业分类数据

#### Scenario: 查询股票行业
- Given: 股票代码
- When: 查询行业分类
- Then: MUST 返回行业名称和分类标准

### Requirement: 龙虎榜数据查询
系统 SHALL 从 Alpha300 查询龙虎榜数据

#### Scenario: 查询龙虎榜
- Given: 股票代码或日期
- When: 查询龙虎榜
- Then: MUST 返回买卖金额、净买入、上榜原因

### Requirement: 融资融券查询
系统 SHALL 从 Alpha300 查询融资融券数据

#### Scenario: 查询融资融券
- Given: 股票代码
- When: 查询融资融券
- Then: MUST 返回融资余额、融券余额

### Requirement: 排名因子查询
系统 SHALL 从 Alpha300 查询排名因子数据

#### Scenario: 查询排名因子
- Given: 股票代码
- When: 查询排名因子
- Then: MUST 返回 momentum, trend, volatility, liquidity 等因子值

#### Scenario: 查询排名快照
- Given: 日期
- When: 查询排名
- Then: MUST 返回该日期的排名和评分

### Requirement: 数据缓存
系统 SHALL 缓存 Alpha300 查询结果

#### Scenario: K线数据缓存
- Given: 查询过的 K 线数据
- When: 再次查询相同参数
- Then: MUST 从缓存返回，不查询数据库

#### Scenario: 缓存过期
- Given: 缓存数据超过 TTL
- When: 查询数据
- Then: MUST 重新查询数据库并更新缓存
