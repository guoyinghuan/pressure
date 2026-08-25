package benchmark

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

// fixedCost 返回一个耗时固定、永不出错的被压函数。
func fixedCost(d time.Duration) Call {
	return func(times int) (time.Duration, error) { return d, nil }
}

func TestStress(t *testing.T) {
	Stress(1, 1, baidu)
}

func baidu(i int) (time.Duration, error) {
	start := time.Now()
	resp, err := http.Get("https://www.baidu.com/")
	if err != nil {
		return time.Since(start), err
	}
	defer resp.Body.Close()
	d, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(string(d))
	return time.Since(start), nil
}

// 全部成功：计数、成功率、耗时统计应精确（每个请求恰好统计一次）。
func TestStressAllSuccess(t *testing.T) {
	r := Stress(100, 10, fixedCost(10*time.Millisecond))
	if r.TotalNum != 100 || r.SuccessNum != 100 || r.ErrorNum != 0 {
		t.Fatalf("计数错误: %+v", r)
	}
	if r.SuccessRate != "1.0000" {
		t.Fatalf("成功率 = %s, want 1.0000", r.SuccessRate)
	}
	if r.AvgTime != 10 || r.MaxTime != 10 || r.MinTime != 10 {
		t.Fatalf("耗时统计错误: avg=%d max=%d min=%d", r.AvgTime, r.MaxTime, r.MinTime)
	}
	if r.Qps <= 0 {
		t.Fatalf("QPS 错误: %+v", r)
	}
	if len(r.Err) != 0 {
		t.Fatalf("不应有错误: %v", r.Err)
	}
}

// 真实耗时函数：总耗时与 QPS 应基于墙钟时间统计。
func TestStressTotalTimeAndQps(t *testing.T) {
	r := Stress(100, 10, func(times int) (time.Duration, error) {
		start := time.Now()
		time.Sleep(time.Millisecond)
		return time.Since(start), nil
	})
	if r.TotalTime <= 0 {
		t.Fatalf("总耗时应为正: %+v", r)
	}
	if r.Qps <= 0 {
		t.Fatalf("QPS 应为正: %+v", r)
	}
	if r.ErrorNum != 0 || r.SuccessNum != 100 || r.SuccessRate != "1.0000" {
		t.Fatalf("计数错误: %+v", r)
	}
	// 理论耗时约 100*1ms/10 = 10ms，留足调度抖动余量
	if r.TotalTime < 5 || r.TotalTime > 100 {
		t.Fatalf("总耗时异常: %d ms", r.TotalTime)
	}
}

// 全部失败：错误计数必须精确，不能因并发丢失更新而少计。
func TestStressAllErrors(t *testing.T) {
	r := Stress(2000, 50, func(times int) (time.Duration, error) {
		return time.Microsecond, errBoom
	})
	if r.ErrorNum != 2000 || r.SuccessNum != 0 {
		t.Fatalf("错误计数漂移: %+v", r)
	}
	if r.SuccessRate != "0.0000" {
		t.Fatalf("成功率 = %s, want 0.0000", r.SuccessRate)
	}
	// 错误列表有上限，避免内存无限增长（与实现中 maxStoredErrors 一致）
	if len(r.Err) != 100 {
		t.Fatalf("保留错误数 = %d, want 100", len(r.Err))
	}
}

// 一半失败：错误数、成功数、成功率都要对半分。
func TestStressHalfErrors(t *testing.T) {
	r := Stress(100, 10, func(times int) (time.Duration, error) {
		if times%2 == 0 {
			return time.Millisecond, errBoom
		}
		return time.Millisecond, nil
	})
	if r.ErrorNum != 50 || r.SuccessNum != 50 {
		t.Fatalf("计数错误: %+v", r)
	}
	if r.SuccessRate != "0.5000" {
		t.Fatalf("成功率 = %s, want 0.5000", r.SuccessRate)
	}
	if len(r.Err) != 50 {
		t.Fatalf("保留错误数 = %d, want 50", len(r.Err))
	}
}

// 每次请求耗时不同（1..100ms）：若丢请求，max/min/avg 必然偏离精确值。
func TestStressVaryingTimes(t *testing.T) {
	r := Stress(100, 10, func(times int) (time.Duration, error) {
		return time.Duration(times%100+1) * time.Millisecond, nil
	})
	if r.MaxTime != 100 || r.MinTime != 1 || r.AvgTime != 50 {
		t.Fatalf("耗时统计错误: avg=%d max=%d min=%d", r.AvgTime, r.MaxTime, r.MinTime)
	}
}

// 零请求：不应 panic，返回零值结果。
func TestStressZeroTotal(t *testing.T) {
	r := Stress(0, 1, fixedCost(time.Millisecond))
	if r.TotalNum != 0 || r.Qps != 0 {
		t.Fatalf("意外结果: %+v", r)
	}
}

// 零并发：不应死锁或 panic，返回零值结果。
func TestStressZeroGoroutine(t *testing.T) {
	r := Stress(10, 0, fixedCost(time.Millisecond))
	if r.TotalNum != 0 || r.Qps != 0 {
		t.Fatalf("意外结果: %+v", r)
	}
}

// 零耗时函数：不应除零溢出（旧实现 Qps 会变成 -9223372036854775808）。
func TestStressZeroDuration(t *testing.T) {
	r := Stress(1, 1, fixedCost(0))
	if r.Qps < 0 || r.MaxTime != 0 || r.MinTime != 0 || r.AvgTime != 0 {
		t.Fatalf("意外结果: %+v", r)
	}
}

// 空函数：不应 panic，返回零值结果。
func TestStressNilFunc(t *testing.T) {
	r := Stress(10, 1, nil)
	if r.TotalNum != 0 {
		t.Fatalf("意外结果: %+v", r)
	}
}
