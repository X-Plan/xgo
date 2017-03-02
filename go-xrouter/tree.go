// tree.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-01
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-02

package xrouter

import (
	"strings"
)

type nodeType uint8

const (
	static nodeType = iota // default node type
	param                  // ':name' wildcard node type
	all                    // '*name' wildcard node type
)

type node struct {
	path     string
	index    byte
	nt       nodeType
	priority uint32
	children []*node
	handle   XHandle
}

// Returns the handle registered with the given path. The values of wildcards
// are saved to a xps parameter which are ordered. tsr (trailing slash redirect)
// parameter is used to control whether get function returns a handle exists
// with an extra (without the) trailing slash for given path when it hasn't
// been registered.
func (n *node) get(path string, xps XParams, tsr bool) XHandle {
	var (
		i    int
		tail string
	)

outer:
	for len(path) > 0 {
		switch n.nt {
		case static:
			i = lcp(path, n.path)
			path, tail = path[i:], n.path[i:]

			if tail == "" {
				if len(path) > 0 {
					if n = n.child(path[0]); n == nil {
						break outer
					}
					continue
				} else {
					return n.handle
				}
			} else {
				break outer
			}

		case param:
			if i = strings.IndexByte(path, '/'); i == -1 {
				i = len(path)
			}
			xps = append(xps, XParam{Key: n.path[1:], Value: path[:i]})
			path = path[i:]

			if len(path) > 0 {
				if n = n.child(path[0]); n == nil {
					break outer
				}
				continue
			} else {
				return n.handle
			}

		case all:
			xps = append(xps, XParam{Key: n.path[1:], Value: path})
			return n.handle

		default:
			// Unless I make a mistake, this statement will never be executed.
			panic(fmt.Sprintf("invalid node type (%d)", n.nt))
		}
	}

	return nil
}

// Locate the approriate child node by index parameter.
func (n *node) child(index byte) *node {
	for _, c := range n.children {
		if c.index == index {
			return c
		}
	}
	return nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

// Find the longest common prefix.
func lcp(a, b string) int {
	var i, max = 0, min(len(a), len(b))
	for i < max && a[i] == b[i] {
		i++
	}
	return i
}
