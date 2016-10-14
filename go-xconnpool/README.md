#go-xconnpool

![Building](https://img.shields.io/badge/building-passing-green.svg)
![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)

**go-xconnpool**包实现了一个并发安全的连接池, 它可以用来管理和重用连接.  
该包有两个主要的结构体: 

* **XConn**: XConn一个net.Conn接口的具体实现, 它将已有的连接进行了一层简单的封装.  
每个XConn都与一个特定的XConnPool关联, 其Close操作不是简单的释放连接, 而是有  
选择的将连接归还到对应的连接池.(**注意**: XConn不是并发安全对象)
* **XConnPool**: XConnPool是一个XConn的连接池. 为了实现并发安全, 它在实现上用到了  
Go的原生Channel.


## 例子

```
var (
    // 创建一个连接生产函数用于产生基础的连接, 这里选择对net.Dial进行一层简单封装
    factory = func() (net.Conn, error) {
        return net.Dial("tcp", "127.0.0.1:8000")
    }

    xcp *xconnpool.XConnPool
    conn net.Conn
    err error
)

// 创建一个容量为10的连接池
xcp = xconnpool.New(10, factory)
if xcp != nil {
    // 处理错误
}

// 从连接池中获取连接.
conn, err = xcp.Get()
if err != nil {
    // 处理错误
}

// 操作连接.
io.WriteString(conn, "Hello World!")

// 关闭连接. 这个操作不是简单的释放连接,
// 而是将连接返回到连接池中.
if err = conn.Close(); err != nil {
    // 处理错误
}

// 可以观测下连接池现在的大小. 
fmt.Printf("Size of XConnPool: %d", xcp.Size())

// 关闭连接池, 这个操作会关闭所有已经在连接池中的
// 连接.
if err = xcp.Close(); err != nil {
    // 处理错误.
}

```
更详细的例子可以参考**test**目录下的文件.


