# go-xp

![Building](https://img.shields.io/badge/building-passing-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xp** is an implementation of [X-Protocol][x-protocol].

## X-Protocol

*X-Protocol* is a [ProtoBuf][protobuf] protocol, which used to standardize application protocols based on **TCP** communication.     
The underlying protocol of *X-Protocol* is a simple binary protocol, the more detail you can see [go-xpacket](../go-xpacket) package.

## Client

The client end of *X-Protocol*. It has only one method (`RoundTrip`) currently, the format of this method is similar to [Transport.RoundTrip](https://golang.org/pkg/net/http/#Transport.RoundTrip).

```go
  rsp, err := client.RoundTrip(req)
  if err != nil {
    // Handle error.
  }
```

## Server & Router

In fact, `Server` is independent of *X-Protocol*. Because the `Handler` field of it is connection-oriented, so you      
can use it separately when you don't use *X-Protocol*. But I recommend you use `Router` if your application      
depends on *X-Protocol*. It provides a request-oriented interface for you, which makes development easier.

```go
  router := &Router{}
  router.Bind(1000, 1000, HandlerFunc(func(req *Request) (*Response, error) {
    return &Response{Body: []byte("Hello World")}, nil
  }), nil)

  s := &Server{ Handler: router}
  s.Serve(l)
```

[protobuf]: https://developers.google.com/protocol-buffers/
[x-protocol]: xp.proto
