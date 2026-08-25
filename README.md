# pressure

简单易用的 Go 语言压测库，通过协程并发调用目标函数，并统计请求耗时、QPS、成功率等指标。

## 功能特性

- 🚀 并发压测：指定请求总数与并发协程数，自动调度执行
- 📊 结果统计：总耗时、平均/最大/最小耗时、QPS、成功率、错误列表
- 🏷️ JSON 序列化：`StressResult` 已声明 JSON 标签，便于采集上报

## 安装

```bash
go get github.com/guoyinghuan/pressure/src
```

## 快速开始

```go
package main

import (
	"net/http"
	"time"

	"github.com/guoyinghuan/pressure/src"
)

func main() {
	// 压测 100 次请求，10 个协程并发
	result := benchmark.Stress(100, 10, func(times int) (time.Duration, error) {
		start := time.Now()
		resp, err := http.Get("https://www.baidu.com/")
		if err != nil {
			return time.Since(start), err
		}
		defer resp.Body.Close()
		return time.Since(start), nil
	})

	_ = result // result 即压测统计结果
}
```

执行后会直接在控制台打印统计结果，例如：

```
baidu 压测结果
总请求数：100
总耗时：856 ms
错误请求数：0
成功率：1.0000
qps：116
平均耗时：86 ms
最大耗时：320.00 ms
最小耗时：51.00 ms
```

## API

### Stress

```go
func Stress(total, goroutine int, f Call) StressResult
```

| 参数        | 类型   | 说明                     |
| ----------- | ------ | ------------------------ |
| `total`     | int    | 总请求数                 |
| `goroutine` | int    | 并发协程数               |
| `f`         | `Call` | 被压测的函数             |

### Call

被压测函数的签名：

```go
type Call func(times int) (time.Duration, error)
```

| 参数/返回值     | 类型            | 说明                             |
| --------------- | --------------- | -------------------------------- |
| `times`         | int             | 请求序号，从 0 开始              |
| 返回值 1        | `time.Duration` | 本次请求的执行耗时               |
| 返回值 2        | error           | 请求是否出错                     |

> 压测函数内部需自行记录耗时（一般取 `time.Now()` 到执行结束的差值）。

### StressResult

| 字段           | JSON 标签        | 说明                 |
| -------------- | ---------------- | -------------------- |
| `TotalTime`    | `total_time`     | 总耗时（毫秒）       |
| `AvgTime`      | `avg_time`       | 平均耗时（毫秒）     |
| `MaxTime`      | `max_time`       | 最大耗时（毫秒）     |
| `MinTime`      | `min_time`       | 最小耗时（毫秒）     |
| `Qps`          | `qps`            | 每秒请求数           |
| `TotalNum`     | `total_num`      | 总请求数             |
| `ErrorNum`     | `error_num`      | 失败请求数           |
| `SuccessNum`   | `success_num`    | 成功请求数           |
| `SuccessRate`  | `success_rate`   | 成功率（保留 4 位小数） |
| `Err`          | `err`            | 错误列表             |
| `FunctionName` | `function_name`  | 被压测的函数名       |

## 注意事项

- 结果中的耗时字段单位均为**毫秒**，`Call` 返回值为 `time.Duration`（纳秒），由库统一换算。
- `SuccessRate` 为字符串类型，格式形如 `1.0000`。
