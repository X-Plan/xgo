# go-xtcpapi

![Building](https://img.shields.io/badge/building-passing-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xtcpapi** provides some extended functions of TCP based on the standard net package.   

## Inherited Listen

[net.Listen](https://golang.org/pkg/net/#Listen) always creates a new listener, but **xtcpapi.Listen** can return a listener from the parent process.     
This feature will be better for the server when it restarts.


## TCP Server

**xtcpapi.Server** provides a simple TCP server framework, you only need to design how to handle connections.

