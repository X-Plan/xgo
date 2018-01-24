# go-xconnpool

![Building](https://img.shields.io/badge/building-passing-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xconnpool** package implements a concurrent safe connection pool, it can be used to     
manage and reuse connections. The main elements of this package as follows.

- **XConn**: XConn is an implementation of `net.Conn` interface, it wraps the underlying      
connection. Each XConn is concern with a **XConnPool**, and closing a XConn will optionally    
return it to the connection pool instead of releasing it directly.    
- **XConnPool**: XConnPool is a connection pool of XConn. For concurrent safe, its implementation    
is based on [Go Channel](https://tour.golang.org/concurrency/2)
- **XConnPools**: XConnPools is also a connection pool type based on XConnPool type. The objective     
of designing it is to solve the redistribution problem of backend addresses, this can't be detected    
by XConnPool. I recommend you use this instead of XConnPool when backend addresses can be      
changed dynamically.

## XConnPool Usage

``` go
var (
    factory = func() (net.Conn, error) {
        return net.Dial("tcp", "127.0.0.1:8000")
    }

    xcp *xconnpool.XConnPool
    conn net.Conn
    err error
)

xcp = xconnpool.New(10, factory)
if xcp != nil {
  // Handle error.
}

conn, err = xcp.Get()
if err != nil {
  // Handle error.
}

io.WriteString(conn, "Hello World!")

if err = conn.Close(); err != nil {
  // Handle error.
}

fmt.Printf("Size of XConnPool: %d", xcp.Size())

if err = xcp.Close(); err != nil {
  // Handle error.
}

```
The more detail example you can find in **test** directory :smirk:.



## XConnPools Usage

```go
addrs := []string{
		"192.168.1.100:80:10",
		"192.168.1.101:80:10",
		"192.168.1.102:80:10",
		"192.168.1.103:80:10",
		"192.168.1.104:80:10",
}

scheduler, _ := xsched.New(addrs)
xcps := xconnpool.NewXConnPools(32, scheduler, net.Dial)
if xcps != nil {
  // Handle error.
}

conn, err := xcps.Get()
if err != nil {
  // Handle error.
}

io.WriteString(conn, "Hello World!")

if err = conn.Close(); err != nil {
  // Handle error.
}

if err = xcps.Close(); err != nil {
  // Handle error.
}
```

It's similar to the usage of **XConnPool**, but maybe you need to know more detail     
about [go-xsched](../go-xsched) package.
