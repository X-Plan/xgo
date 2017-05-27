// tree.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-05-26
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-05-27

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

// This auxiliary type is used for trailing slash redirect.
type tsrType uint8

const (
	notRedirect tsrType = iota // Can't be redirected.
	removeSlash                // Can be redirected by removing trailing slash.
	addSlash                   // Can be redirected by adding trailing slash.
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

// Register a new handle with the given path. If the path conflicts with
// a existing path, a error will be returned. path need to be noempty,
// otherwise anything won't happen, include error. I implement this function
// by recursively calling, because this will make implement recovery strategy
// more easy and code more readable. Of course, it will sacrifice performance,
// but the frequency of calling this function is lower comparing with 'get',
// so it doesn't matter.
func (n *node) add(path string, full string, handle XHandle) (err error) {
	if len(n.path) == 0 {
		// New node.
		return n.construct(path, full, handle)
	}

	switch n.nt {
	case static:
		i := lcp(path, n.path)
		if i < len(path) {
			if i < len(n.path) {
				err = n.split(i, nil)
			}
			if child := n.child(path[i]); child != nil {
				err = child.add(path[i:], full, handle)
			} else {
				n.children = append(n.children, &node{})
				err = n.children[len(n.children)-1].construct(path[i:], full, handle)
			}
		} else if i < len(n.path) && i == len(path) {
			err = n.split(i, handle)
		} else { // i == len(n.path) == len(path)
			err = fmt.Errorf("path '%s' has already been registered", full)
		}
	case param:
		i := lcp(path, n.path)
	case all:
	}

	return
}

func (n *node) construct(path string, full string, handle XHandle) error {
}

func (n *node) split(i int, handle XHandle) error {
}

// Returns the handle registered with the given path. The values of wildcards
// are saved to a xps parameter which are ordered. enableTSR control whether
// executes a TSR (trailing slash redirect) recommendation statement.
func (n *node) get(path string, enableTSR bool) (h XHandle, xps XParams, tsr tsrType) {
	var (
		i      int
		parent *node
	)

outer:
	for len(path) > 0 {
		switch n.nt {
		case static:
			for i = 0; i < len(n.path) && i < len(path) && path[i] == n.path[i]; {
				i++
			}
		case param:
			for i = 0; i < len(path); i++ {
				if path[i] == '/' {
					break
				}
			}

			// Because the value of XParam can't be empty, so the 'i' must
			// be greater than zero.
			if i > 0 && n.path[len(n.path)-1] == '/' {
				if i == len(path) {
					break outer
				}

				if xps == nil {
					xps = make(XParams, 0, n.maxParams)
				}
				xps = append(xps, XParam{Key: n.path[1 : len(n.path)-1], Value: path[:i]})
				i++
			} else if i > 0 {
				xps = append(xps, XParam{Key: n.path[1:], Value: path[:i]})
			}
		case all:
			xps = append(xps, XParam{Key: n.path[1:], Value: path})
			i = len(path)
		}

		if i < len(path) {
			if child := n.child(path[i]); child != nil {
				parent, n, path = n, child, path[i:]
				continue
			}
		} else if n.handle != nil {
			h = n.handle
		}
		break outer
	}

	if h == nil && enableTSR {
		tsr = n.canTSR(parent, path, i)
	}

	return
}

func (n *node) canTSR(parent *node, path string, i int) tsrType {
	if len(path) == 0 || path[len(path)-1] != '/' {
		switch n.nt {
		case static:
			if n.handle != nil && i == len(path) && i == len(n.path)-1 && n.path[i] == '/' {
				return addSlash
			}
		case param:
			if n.handle != nil && n.path[len(n.path)-1] == '/' {
				return addSlash
			}
		}
	} else { // len(path) > 0 && path[len(path)-1] == '/'
		switch n.nt {
		case static:
			if i == len(path)-1 && i == len(n.path) {
				if len(n.path) > 1 && n.handle != nil ||
					len(n.path) == 1 && parent.handle != nil {
					return removeSlash
				}
			}
		case param:
			if i == len(path)-1 && n.n.handle != nil {
				return removeSlash
			}
		}
	}
	return notRedirect
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

// Resort the children by the priority.
func (n *node) resort() {
	if n != nil && !sort.IsSorted(n.children) {
		sort.Sort(n.children)
	}
}

type nodes []*node

// Impelment sort.Interface.
func (ns nodes) Len() int {
	return len(ns)
}

func (ns nodes) Less(i, j int) bool {
	return ns[i].priority > ns[j].priority
}

func (ns nodes) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
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
