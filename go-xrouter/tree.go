// tree.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-01
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-07

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
	tsr      bool
	index    byte
	nt       nodeType
	priority uint32
	children []*node
	handle   XHandle
}

// Register a new handle with the given path. If the path conflicts with
// a existing path, a error will be returned.
func (n *node) add(path string, handle XHandle) error {
	var (
		i     int
		rest  string
		child *node
	)
	return nil
}

// Init a empty node (not nil) from path parameter.
func (n *node) construct(path, full string, handle XHandle) error {
	var (
		i int
	)

	for len(path) > 0 {
		// The priority is always equal to 1, because all of the nodes
		// in 'construct' function grow on a new branch of the trie.
		n.priority = 1

		switch path[0] {
		case ':':
			n.nt = param
			if i = strings.IndexAny(path[1:], ":*/"); i != -1 {
				if path[i] != '/' {
					return fmt.Errorf("'%s' in path '%s': only one wildcard per path segment is allowed", path, full)
				}
				n.path, path = path[:i], path[i:]
				n.children = make([]*node, 1)
				n = n.children[0]
			} else {
				n.handle, n.path, path = handle, path, ""
				// create a tsr node.
				n.children = []*node{&node{path: "/", tsr: true, index: '/', priority: 1, handle: handle}}
			}

		case '*':
			n.nt = all
			if i = strings.IndexAny(path[1:], ":*/"); i != -1 {
				return fmt.Errorf("'%s' in path '%s': catch-all routes are only allowed at the end of the path", path, full)
			}
			n.handle, n.path, path = handle, path, ""

		default:
			if i = strings.IndexAny(path, ":*"); i != -1 {
				n.path, path = path[:i], path[i:]
				n.children = make([]*node, 1)
				n = n.children[0]
			} else {
				n.handle, n.path, path = handle, path, ""
				// There are two cases when create the tsr node of a static node,
				// which is not like creating the tsr node of a param node, we only
				// need to consider adding slash, because a slash isn't allowed in the
				// path segment of a param node, but a static node allows it.
				if path[len(path)-1] == '/' {
					n.path = n.path[:len(n.path)-1]
				}
				n.children = []*node{&node{path: "/", tsr: true, index: '/', priority: 1, handle: handle}}
			}
		}
	}
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
	for {
		if n.nt != all {
			if n.nt == static {
				i = lcp(path, n.path)
				path, tail = path[i:], n.path[i:]
			} else {
				// 'param' node type
				if i = strings.IndexByte(path, '/'); i == -1 {
					i = len(path)
				}
				xps = append(xps, XParam{Key: n.path[1:], Value: path[:i]})
				path = path[i:]
			}

			if n.nt == param || tail == "" {
				if len(path) > 0 {
					if n = n.child(path[0]); n == nil {
						break outer
					}
					continue
				} else if !n.tsr || (tsr && n.tsr) {
					return n.handle
				}
			}

		}

		// 'all' node type
		xps = append(xps, XParam{Key: n.path[1:], Value: path})
		return n.handle
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
