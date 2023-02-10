package benchmark

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "testing"
    "time"
)

func TestStress(t *testing.T) {
    Stress(1, 1, baidu)
}

func baidu(i int) (time.Duration, error) {
    start := time.Now()
    url := "https://www.baidu.com/"
    resp, err := http.Get(url)
    defer resp.Body.Close()
    d, _ := ioutil.ReadAll(resp.Body)
    fmt.Println(string(d))
    if err != nil {
        return time.Since(start), err
    }
    return time.Since(start), err
}
