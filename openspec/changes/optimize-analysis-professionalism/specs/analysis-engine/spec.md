## ADDED Requirements

### Requirement: Technical Indicators Engine
系统 SHALL 计算 MACD、KDJ、RSI、布林带等技术指标用于个股分析

#### Scenario: 计算 MACD 指标
- Given: 股票代码 600519，有 120 天 K 线数据
- When: 调用 ComputeIndicators()
- Then: 返回 DIF、DEA、柱状图值，MACD 信号（金叉/死叉/中性）

#### Scenario: K 线数据不足
- Given: 股票代码 600519，仅有 10 天 K 线数据
- When: 调用 ComputeIndicators()
- Then: 数据不足的指标 MUST 返回默认值，不报错

### Requirement: Fund Flow Analysis
系统 SHALL 分析主力资金、北向资金、融资融券流向

#### Scenario: 主力资金净流入评分
- Given: 近 5 日主力净流入数据
- When: 计算资金流向评分
- Then: MUST 返回主力净流入评分 0-100，5日趋势

#### Scenario: 资金数据不可用
- Given: EastMoney 资金流 API 返回错误
- When: 计算资金流向评分
- Then: MUST 返回中性评分 50，不阻断整体分析

### Requirement: Volume-Price Analysis
系统 SHALL 评估量价配合度

#### Scenario: 量价齐升
- Given: 近 5 日价格上涨且成交量递增
- When: 计算量价配合度
- Then: 量价配合度评分 SHALL 大于 70，信号为 "强势"

#### Scenario: 量价背离
- Given: 近 5 日价格创新高但成交量递减
- When: 计算量价配合度
- Then: 量价配合度评分 SHALL 小于 40，信号为 "背离警告"

### Requirement: Valuation Analysis
系统 SHALL 计算 PE/PB 历史分位数

#### Scenario: 低估值股票
- Given: 股票 PE=15，近 3 年 PE 分位数为 20%
- When: 计算估值评分
- Then: 估值评分 SHALL 大于 70（低估），信号为 "估值偏低"

### Requirement: Unified Scoring Engine
系统 MUST 提供统一的 8 维度加权评分

#### Scenario: 完整 8 维度评分
- Given: 股票代码 600519，所有数据源正常
- When: 调用 AnalyzeStock()
- Then: MUST 返回 8 个维度评分（技术面、资金面、量价、估值、趋势、波动率、板块、情绪），每个评分 0-100

#### Scenario: 部分维度数据缺失
- Given: 股票代码 600519，新闻 API 失败
- When: 调用 AnalyzeStock()
- Then: 消息面维度 MUST 使用默认评分 50，其他维度正常计算

## MODIFIED Requirements

### Requirement: Daily Report Generation
每日报告 SHALL 包含更专业的分析内容

#### Scenario: 生成完整每日报告
- Given: 自选股列表有 10 只股票，大盘数据正常
- When: 调用 GenerateDailyReport()
- Then: 报告 MUST 包含大盘概况、板块轮动、个股 8 维度分析、AI 综述

#### Scenario: 报告包含板块分析
- Given: 今日领涨板块为半导体、新能源
- When: 生成每日报告
- Then: 报告 MUST 包含板块轮动章节，列出 Top 5 领涨/领跌板块

### Requirement: Watchlist Ranking
综合排名 MUST 使用多因子评分

#### Scenario: 多因子排名
- Given: 自选股 10 只，所有数据正常
- When: 调用 Ranking()
- Then: 每只股票 MUST 返回动量、趋势、资金、波动率、估值 5 个因子得分

#### Scenario: 行业内相对排名
- Given: 自选股包含 3 只银行股
- When: 调用 Ranking()
- Then: 银行股之间 SHALL 有相对排名
