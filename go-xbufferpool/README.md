# go-xbufferpool

![Building](https://img.shields.io/badge/building-passing-green.svg)
![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xbufferpool**包实现了一个并发安全的缓存池, 它可以用来管理和
重用缓存. 该包含有两个主要的结构体:

* **XBuffer**: XBuffer是bytes.Buffer的一个封装, 你可以把它当成bytes.Buffer  
使用, 但是它多了一个接口Close, 调用这个接口会将XBuffer归还到对应的缓存池.
* **XBufferPool**: XBufferPool是一个XBuffer的缓存池. 为了实现并发安全, 它  
在实现上用到了Go的原生Channel.

## 例子

```go
var (
    xbp *XBufferPool = xbufferpool.New(1000, 0)
)

if xbp == nil {
    // 处理错误.
}

// 获取缓存.
xb, err := xbp.Get()
if err != nil {
    // 处理错误.
}

// 使用缓存.

// 归还存储.
err = xb.Close()
if err != nil {
    // 处理错误.
}

// 关闭缓存池.
err = xbp.Close()
if err != nil {
    // 处理错误.
}

```

# 效率

这是将使用缓存池和未使用缓存池进行比对, 都采用100个
go-routine同时并发获取缓存, 进行写操作, 然后归还. 
数据块的大小是这次测试的变量.

>**使用缓存池**  
>BenchmarkBufferPool128-4           30000         46628 ns/op  
>BenchmarkBufferPool256-4           30000         48481 ns/op  
>BenchmarkBufferPool512-4           30000         49829 ns/op  
>BenchmarkBufferPool1024-4          30000         49434 ns/op  
>BenchmarkBufferPool2048-4          30000         50586 ns/op  
>BenchmarkBufferPool4096-4          30000         50973 ns/op  
>BenchmarkBufferPool8192-4          30000         55831 ns/op  
>
>**不使用缓存池**  
>BenchmarkDummyBufferPool128-4      50000         33444 ns/op  
>BenchmarkDummyBufferPool256-4      30000         36074 ns/op  
>BenchmarkDummyBufferPool512-4      30000         40642 ns/op  
>BenchmarkDummyBufferPool1024-4     20000         57955 ns/op  
>BenchmarkDummyBufferPool2048-4     20000         75907 ns/op  
>BenchmarkDummyBufferPool4096-4     10000        168982 ns/op  
>BenchmarkDummyBufferPool8192-4     10000        224390 ns/op  


在获取小块缓存的情况下, 缓存池的效率并不理想. 但是在获取中块和  
大块的情况下从缓存池获取缓存要更加经济实用. 还可以看出使用缓存池  
获取缓存所需要的时间正比于缓存的大小, 但是该比例系数增长的十分  
缓慢, 而不使用缓存池的情况则恰恰相反.
