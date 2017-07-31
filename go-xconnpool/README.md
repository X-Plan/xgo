# go-xconnpool

![Building](https://img.shields.io/badge/building-passing-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xconnpool** package implement a concurrent safe connection pool, it can be used to     
manage and reuse connections. The main elements of this package as follows.

- **XConn**: XConn is a implementation of `net.Conn` interface, it wraps the underlying      
connection. Each XConn is concern with a **XConnPool**, and closing a XConn will optionaly    
return it to the connection pool instead of releasing it directly.    
- **XConnPool**: XConnPool is a connection pool of XConn. For concurrent safe, its implementation    
is based on [Go Channel](https://tour.golang.org/concurrency/2)

## Usage

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


