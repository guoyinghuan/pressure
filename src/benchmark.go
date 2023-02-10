package benchmark

import (
    "fmt"
    "math"
    "reflect"
    "runtime"
    "strings"
    "time"
)

// times 第几个请求
//@return time.Duration 执行时间
type Call func(times int) (time.Duration, error)

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

// total 总共多少次
// goroutine 协程数
func Stress(total, goroutine int, f Call) StressResult {
    var per []int64
    var errNum int64
    var errArr []error
    ch := make(chan int, goroutine)
    start := time.Now()
    for i := 0; i < total; i++ {
        ch <- 1
        go func(ch chan int, times int) {
            exec, err := f(times)
            if err != nil {
                errNum++
                errArr = append(errArr, err)
            }
            per = append(per, int64(exec))
            <-ch
        }(ch, i)
    }
    var result StressResult
    for {
        if len(ch) == 0 {
            end := time.Since(start)
            var sum int64
            for _, item := range per {
                sum += item
            }
            avg := sum / int64(len(per))

            max := float64(per[0])
            min := float64(per[0])
            for i := 1; i < len(per); i++ {
                max = math.Max(max, float64(per[i]))
                min = math.Min(min, float64(per[i]))
            }
            qps := float64(time.Second) / float64(avg) * float64(goroutine)

            result.TotalTime = end.Milliseconds()
            result.AvgTime = avg / 1e6
            result.MaxTime = int64(max / 1e6)
            result.MinTime = int64(min / 1e6)
            result.Qps = int64(qps)
            result.TotalNum = int64(total)
            result.ErrorNum = errNum
            result.SuccessNum = result.TotalNum - errNum
            result.SuccessRate = fmt.Sprintf("%.4f", float64(result.SuccessNum)/float64(result.TotalNum))
            result.Err = errArr
            result.FunctionName = FunctionName(f)

            fmt.Println(fmt.Sprintf("%s 压测结果", result.FunctionName))
            fmt.Println(fmt.Sprintf("总请求数：%d", result.TotalNum))
            fmt.Println(fmt.Sprintf("总耗时：%d ms", result.TotalTime))
            fmt.Println(fmt.Sprintf("错误请求数：%d", errNum))
            fmt.Println(fmt.Sprintf("成功率：%s", result.SuccessRate))
            fmt.Println(fmt.Sprintf("qps：%d", int(qps)))
            fmt.Println(fmt.Sprintf("平均耗时：%d ms", avg/1e6))
            fmt.Println(fmt.Sprintf("最大耗时：%.2f ms", max/1e6))
            fmt.Println(fmt.Sprintf("最小耗时：%.2f ms \n", min/1e6))

            break
        }
        time.Sleep(time.Millisecond * 10)
    }
    return result
}
func FunctionName(i interface{}) string {
    fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
    fnArr := strings.Split(fn, ".")
    return fnArr[len(fnArr)-1]
}
