# go-xpacket

![Building](https://img.shields.io/badge/building-passing-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xpacket** is a simple binary protocol encode/decode package.

## Protocol Format

- start of packet &nbsp;**3 octet**
- packet length   &nbsp;&nbsp;**4 octet, big endian**
- packet body     &nbsp;&nbsp;&nbsp;&nbsp;**n octet**
- end of packet   &nbsp;&nbsp;**3 octet**
