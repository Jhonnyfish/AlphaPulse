## 1. 配置和连接

- [ ] 1.1 在 `internal/config/config.go` 添加 Alpha300 数据库配置字段
- [ ] 1.2 在 `.env.example` 添加 Alpha300 数据库配置示例
- [ ] 1.3 创建 `internal/services/alpha300_db.go` 实现数据库连接池
- [ ] 1.4 在 `cmd/server/main.go` 初始化 Alpha300 数据库连接

## 2. 日K线数据

- [ ] 2.1 实现 `FetchKline(ctx, code, days)` 查询日K线
- [ ] 2.2 实现日期格式转换 (YYYYMMDD → YYYY-MM-DD)
- [ ] 2.3 添加 K 线数据缓存 (5分钟 TTL)
- [ ] 2.4 实现批量查询 `FetchKlineBatch(ctx, codes, days)`

## 3. 估值数据

- [ ] 3.1 实现 `FetchDailyBasic(ctx, code)` 查询估值数据
- [ ] 3.2 添加估值数据缓存 (10分钟 TTL)
- [ ] 3.3 实现历史估值查询 `FetchDailyBasicHistory(ctx, code, days)`

## 4. 资金流向

- [ ] 4.1 实现 `FetchMoneyFlow(ctx, code, days)` 查询资金流向
- [ ] 4.2 添加资金流向缓存 (3分钟 TTL)
- [ ] 4.3 实现批量查询 `FetchMoneyFlowBatch(ctx, codes, days)`

## 5. 行业分类

- [ ] 5.1 实现 `FetchIndustry(ctx, code)` 查询行业分类
- [ ] 5.2 添加行业分类缓存 (1小时 TTL)
- [ ] 5.3 实现 `FetchAllIndustries()` 查询所有行业

## 6. 龙虎榜

- [ ] 6.1 实现 `FetchLhb(ctx, code, days)` 查询龙虎榜
- [ ] 6.2 添加龙虎榜缓存 (30分钟 TTL)
- [ ] 6.3 实现按日期查询 `FetchLhbByDate(ctx, date)`

## 7. 融资融券

- [ ] 7.1 实现 `FetchMargin(ctx, code, days)` 查询融资融券
- [ ] 7.2 添加融资融券缓存 (10分钟 TTL)

## 8. 排名因子

- [ ] 8.1 实现 `FetchRankSnapshot(ctx, date)` 查询排名快照
- [ ] 8.2 实现 `FetchRankFactors(ctx, code)` 查询排名因子
- [ ] 8.3 添加排名数据缓存 (5分钟 TTL)

## 9. 集成到分析引擎

- [ ] 9.1 修改 `AnalyzeTechnical` 优先使用 Alpha300 K线数据
- [ ] 9.2 修改 `AnalyzeValuation` 使用 Alpha300 估值数据
- [ ] 9.3 修改 `AnalyzeMoneyFlow` 使用 Alpha300 资金流向
- [ ] 9.4 添加降级逻辑：Alpha300 不可用时使用 EastMoney API

## 10. 集成到报告系统

- [ ] 10.1 修改 `DailyReportGenerate` 使用 Alpha300 排名数据
- [ ] 10.2 修改 `GenerateDailyReportAuto` 使用 Alpha300 排名数据
- [ ] 10.3 在报告中显示数据来源

## 11. 测试

- [ ] 11.1 编写 Alpha300 数据库连接测试
- [ ] 11.2 编写数据查询测试
- [ ] 11.3 编写降级逻辑测试
- [ ] 11.4 集成测试：使用 Alpha300 数据生成报告

## Verification

- [ ] Alpha300 数据库连接成功
- [ ] 日K线查询返回正确数据
- [ ] 估值数据查询返回正确数据
- [ ] 资金流向查询返回正确数据
- [ ] 缓存工作正常
- [ ] 降级逻辑工作正常
- [ ] 每日报告使用 Alpha300 数据
- [ ] 八维分析使用 Alpha300 数据
