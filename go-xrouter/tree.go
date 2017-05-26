// tree.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-05-26
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-05-26

package xrouter

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type nodeType uint8

var nodeTypeStr = [3]string{"static", "param", "catch-all"}

func (nt nodeType) String() string {
	return nodeTypeStr[int(nt)]
}

const (
	static nodeType = iota // default node type
	param                  // ':name' wildcard node type
	all                    // '*name' wildcard node type
)

type node struct {
	path      string
	maxParams uint8
	index     byte
	nt        nodeType
	priority  uint32
	children  nodes
	handle    XHandle
}

// Returns the handle registered with the given path. The values of wildcards
// are saved to a xps parameter which are ordered. enableTSR control whether
// executes a TSR (trailing slash redirect) recommendation statement.
func (n *node) get(path string, enableTSR bool) (h XHandle, xps XParams, tsr bool) {
	var (
		i      int
		parent *node
	)

outer:
	for len(path) > 0 {
		if n.nt != all {
			if n.nt == static {
				for i = 0; i < len(n.path) && i < len(path) && path[i] == n.path[i]; {
					i++
				}
			} else {
				var (
					slash bool
					val   string
				)
				// 'param' node type.
				for i = 0; i < len(path); i++ {
					if path[i] == '/' {
						i++
						slash = true
						break
					}
				}

				if slash {
					val = path[:i-1]
				} else {
					val = path[:i]
				}

				if len(val) == 0 {
					break outer
				}

				if xps == nil {
					xps = make(XParams, 0, n.maxParams)
				}
				xps = append(xps, XParam{Key: n.path[1 : len(n.path)-1], Value: val})
			}

			if n.nt == param || i == len(n.path) {
				if i < len(path) {
					if child := n.child(path[i]); child != nil {
						parent, n, path = n, child, path[i:]
						continue
					}
				} else if n.handle != nil {
					h = n.handle
				}
			}

			break outer
		}

		// 'all' node type
		h, xps = n.handle, append(xps, XParam{Key: n.path[1:], Value: path})
		break outer
	}

	if h == nil && enableTSR {
		tsr = n.canTSR(parent, path, i)
	}

	return
}

func (n *node) canTSR(parent *node, path string) bool {
}

// Locate the approriate child node by index parameter.
func (n *node) child(index byte) *node {
	for _, c := range n.children {
		// If the node tyoe of a child node is either param or all,
		// the number of the children of a node must be equal to 1, we
		// can directly return it. Otherwise have to compare the index.
		if c.nt != static || c.index == index {
			return c
		}
	}
	return nil
}
