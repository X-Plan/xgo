// tree.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-01
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-10

package xrouter

import (
	"sort"
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
	children nodes
	handle   XHandle
}

// Register a new handle with the given path. If the path conflicts with
// a existing path, a error will be returned.
func (n *node) add(path string, handle XHandle) error {
	var (
		i             int
		err           error
		tail          string
		full          = path
		parent, child *node
	)

outer:
	for len(path) > 0 {
		switch n.nt {
		case static:
			i = lcp(path, n.path)
			path, tail = path[i:], n.path[i:]
			if len(tail) > 0 {
				// Split the static node.
				child := &node{
					path:     tail,
					index:    tail[0],
					priority: n.priority,
					children: n.children,
					handle:   n.handle,
				}
				n.path = n.path[:i]
				n.priority++
				parent.resort()

				n.children = append(n.children, child)
				if len(path) > 0 {
					child = &node{}
					n.children = append(n.children, child)
					parent, n = n, child
				}
				break outer
			} else if len(path) == 0 && !n.tsr {
				return fmt.Errorf("path '%s' has already been registered", path)
			}
		case param:
			i = lcp(path, n.path)
			if i == len(n.path) && i < len(path) {
				path = path[i:]
			} else if !n.tsr {
				return fmt.Errorf("'%s' in path '%s': conflict with the existing param wildcard '%s' in prefix '%s'", path, full, n.path, full[:strings.Index(full, path)]+n.path)
			}
		case all:
			return fmt.Errorf("'%s' in path '%s': conflict with the existing catch-all wildcard '%s' in prefix '%s' ", path, full, n.path, full[:strings.Index(full, path)]+n.path)
		}

		if !n.tsr {
			if child = n.child(path[0]); child != nil {
				parent, n = n, child
				continue
			} else {
				child := &node{}
				if err = child.construct(path, full, handle); err != nil {
					return err
				}
				n.priority++
				parent.resort()
				// We don't need to invoke n.resort, because the priority
				// of the 'child' node is minimum (equal to 1).
				n.children = append(n.children, child)
				break outer
			}
		} else {
			n.handle, n.tsr = handle, false
			break outer
		}
	}

	return nil
}

// Init a empty node ('path' field is empty) from path parameter.
func (n *node) construct(path, full string, handle XHandle) error {
	var (
		i int
	)

	// The initial path parameter must be not empty, it means the
	// for-loop will be executed once at least.
	for len(path) > 0 {
		// The priority is always equal to 1, because all of the nodes
		// in 'construct' function grow on the new branch of a trie.
		n.priority = 1

		switch path[0] {
		// If the node type of the current node is static or param,
		// We need add a extra node called tsr node, which is used
		// in 'get' function when the 'tsr' parameter is true.
		case ':':
			n.nt = param
			if i = strings.IndexAny(path[1:], ":*/"); i != -1 {
				if path[i] != '/' {
					return fmt.Errorf("'%s' in path '%s': only one wildcard per path segment is allowed", path, full)
				}
				n.path, path = path[:i], path[i:]
				n.children = make([]*node, 1)
				// Reach the end of the path, we need to consider how to
				// expand a tsr node. In fact, I can redirect the rest path
				// to the default branch, but it makes handling tsr node more
				// complicated (at least for me), so I handle it at this.
				if path == "/" {
					n.handle = handle
					n.tsr = true
					n.children = &node{path: "/", index: '/', priority: 1, handle: handle}
					path = ""
				} else {
					n = n.children[0]
				}

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
				// We only need to set the 'index' field of a static node,
				// there is no use for param node and all node.
				n.path, n.index, path = path[:i], path[0], path[i:]
				n.children = []*node{&node{}}
				n = n.children[0]
			} else if len(path) > 1 {
				n.handle, n.path, path = handle, path, ""
				if path[len(path)-1] == '/' {
					n.path = n.path[:len(n.path)-1]
				}
				n.children = []*node{&node{path: "/", tsr: true, index: '/', priority: 1, handle: handle}}
			} else {
				// Root path ("/") is a special case, we won't
				// expand a tsr node of it, which means it's never
				// be redirected.
				n.handle, n.path, n.index, path = handle, "/", '/', ""
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
