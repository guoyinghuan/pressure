package benchmark

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// maxStoredErrors 结果中最多保留的错误条数，防止大失败量下内存无限增长。
const maxStoredErrors = 100

// Call 是被压测函数的签名。
// times 为请求序号（从 0 开始），返回值依次为本次请求耗时与错误。
type Call func(times int) (time.Duration, error)

// StressResult 压测统计结果，时间字段单位均为毫秒。
type StressResult struct {
	TotalTime    int64   `json:"total_time"` // 毫秒
	AvgTime      int64   `json:"avg_time"`
	MaxTime      int64   `json:"max_time"`
	MinTime      int64   `json:"min_time"`
	Qps          int64   `json:"qps"`
	TotalNum     int64   `json:"total_num"`
	ErrorNum     int64   `json:"error_num"`
	SuccessNum   int64   `json:"success_num"`
	SuccessRate  string  `json:"success_rate"`
	Err          []error `json:"err"`
	FunctionName string  `json:"function_name"`
}

// Stress 对 f 进行压测：共 total 次请求，由 goroutine 个 worker 协程并发执行。
// QPS 为实测吞吐（总请求数 / 实际总耗时）。
// total、goroutine <= 0 或 f 为 nil 时不执行压测，直接返回零值结果。
func Stress(total, goroutine int, f Call) StressResult {
	if f == nil {
		return StressResult{}
	}
	result := StressResult{FunctionName: FunctionName(f)}
	if total <= 0 || goroutine <= 0 {
		return result
	}

	// 每个请求一个槽位，各 worker 只写自己的槽位，避免并发写共享切片。
	per := make([]int64, total)
	var errNum int64
	var errArr []error
	var mu sync.Mutex

	jobs := make(chan int, goroutine)
	var wg sync.WaitGroup
	workers := goroutine
	if workers > total {
		workers = total
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for times := range jobs {
				exec, err := f(times)
				per[times] = int64(exec)
				if err != nil {
					atomic.AddInt64(&errNum, 1)
					mu.Lock()
					if len(errArr) < maxStoredErrors {
						errArr = append(errArr, err)
					}
					mu.Unlock()
				}
			}
		}()
	}

	start := time.Now()
	for i := 0; i < total; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait() // WaitGroup 建立 happens-before，此后读取 per/errNum 无竞态
	elapsed := time.Since(start)

	// 汇总统计（total > 0 已在上方保证）
	var sum int64
	max, min := per[0], per[0]
	for _, v := range per {
		sum += v
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
	}
	avg := sum / int64(total)

	result.TotalTime = elapsed.Milliseconds()
	result.AvgTime = avg / int64(time.Millisecond)
	result.MaxTime = max / int64(time.Millisecond)
	result.MinTime = min / int64(time.Millisecond)
	if elapsed > 0 {
		result.Qps = int64(float64(total) * float64(time.Second) / float64(elapsed))
	}
	result.TotalNum = int64(total)
	result.ErrorNum = errNum
	result.SuccessNum = result.TotalNum - result.ErrorNum
	result.SuccessRate = fmt.Sprintf("%.4f", float64(result.SuccessNum)/float64(result.TotalNum))
	result.Err = errArr

	fmt.Println(fmt.Sprintf("%s 压测结果", result.FunctionName))
	fmt.Println(fmt.Sprintf("总请求数：%d", result.TotalNum))
	fmt.Println(fmt.Sprintf("总耗时：%d ms", result.TotalTime))
	fmt.Println(fmt.Sprintf("错误请求数：%d", result.ErrorNum))
	fmt.Println(fmt.Sprintf("成功率：%s", result.SuccessRate))
	fmt.Println(fmt.Sprintf("qps：%d", result.Qps))
	fmt.Println(fmt.Sprintf("平均耗时：%d ms", result.AvgTime))
	fmt.Println(fmt.Sprintf("最大耗时：%.2f ms", float64(max)/float64(time.Millisecond)))
	fmt.Println(fmt.Sprintf("最小耗时：%.2f ms \n", float64(min)/float64(time.Millisecond)))

	return result
}

func FunctionName(i interface{}) string {
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	fnArr := strings.Split(fn, ".")
	return fnArr[len(fnArr)-1]
}
