# 核心压力测试

本目录用于课程项目答辩所需的核心性能实验，包含接口并发、调度算法规模和持续发单积压三组测试。

## 环境要求

- 后端使用 `config-dev.yml`，MySQL 和 Redis 可正常连接
- 已安装 Go、k6、jq、Python 3
- 默认测试地址为 `http://127.0.0.1:8088`

积压测试会向当前数据库写入模拟运单，请勿直接对生产数据库执行。

## 运行方式

先启动服务：

```bash
./loadtest/start_server.sh
```

运行全部测试：

```bash
API_DURATION=5s BACKLOG_DURATION=20s ./loadtest/run.sh all
```

也可以分别运行：

```bash
./loadtest/run.sh algorithm
API_DURATION=5s ./loadtest/run.sh api
BACKLOG_DURATION=20s ./loadtest/run.sh backlog
python3 ./loadtest/scripts/build_report_data.py
```

如服务地址不同，可设置：

```bash
BASE_URL=http://127.0.0.1:8088 ./loadtest/run.sh api
```

## 结果文件

- `results/api-summary.csv`：接口 QPS、P95、HTTP/业务失败率
- `results/algorithm-summary.csv`：NF 与 SA 在不同车辆规模下的耗时
- `results/backlog-summary.csv`：不同发单速率下的订单积压
- `results/raw/`：k6 原始 JSON 和时间序列 CSV
- `results/charts/`：可直接插入 PowerPoint 的 SVG 曲线图

运行 `build_report_data.py` 不依赖 matplotlib 等第三方 Python 包。
