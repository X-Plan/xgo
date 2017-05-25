// tree.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-01
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-05-25

package xrouter

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// 'tree' contains the root node of the tree, and it also
// has a read-write lock to make it safe in concurrent scenario.
type tree struct {
	rwmtx *sync.RWMutex
	n     *node
}

func (t *tree) add(path string, handle XHandle) error {
	t.rwmtx.Lock()
	err := t.n.add(path, handle)
	t.rwmtx.Unlock()
	return err
}

func (t *tree) get(path string, xps *XParams, tsr bool) XHandle {
	var handle XHandle
	t.rwmtx.RLock()
	if len(t.n.path) > 0 {
		handle = t.n.get(path, xps, tsr)
	}
	t.rwmtx.RUnlock()
	return handle
}

// If the path length of the root node is zero, which
// represent this tree is empty.
func (t *tree) isempty() bool {
	t.rwmtx.RLock()
	result := len(t.n.path) == 0
	t.rwmtx.RUnlock()
	return result
}

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
	path     string
	tsr      bool
	index    byte
	nt       nodeType
	priority uint32
	children nodes
	handle   XHandle
}

// Register a new handle with the given path. If the path conflicts with
// a existing path, a error will be returned. path need to be noempty,
// otherwise anything won't happen, include error. The first byte of the
// path should be slash ('/').
func (n *node) add(path string, handle XHandle) error {
	var (
		i             int
		err           error
		full          = path
		parent, child *node
	)

	// NOTE: Because the value of static is equal to zero,
	// so I never explicitly set it.
outer:
	for len(path) > 0 {
		switch n.nt {
		case static:
			i = lcp(path, n.path)
			if i < len(n.path) && i < len(path) && (path[i] == ':' || path[i] == '*') {
				return fmt.Errorf("'%s' in path '%s': wildcard confilicts with the existing path segment '%s' in prefix '%s'", path[i:], full, n.path[i:], full[:strings.Index(full, path)]+n.path)
			} else if i > 0 && i < len(n.path) {
				if path = n.split(parent, i, path, handle); len(path) == 0 {
					break outer
				}
			} else if i < len(path) {
				path = path[i:]
			} else if i == len(path) && n.handle != nil && !n.tsr {
				return fmt.Errorf("path '%s' has already been registered", path)
			}
		case param:
			i = lcp(path, n.path)

			var ok bool
			if i == len(n.path) {
				if i < len(path) && path[i] == '/' {
					path = path[i:]
					ok = true
				} else if i == len(path) {
					if n.handle == nil || n.tsr {
						path = ""
						ok = true
						if !n.tsr && len(n.children) == 1 {
							// param node has only child one node at most, and the
							// first character of child.path must be '/'.
							child = n.children[0]
							if len(child.path) > 1 {
								subChild := *child
								subChild.path, subChild.index = child.path[1:], child.path[1]
								child.path, child.children = "/", []*node{&subChild}
							}
							child.handle, child.tsr = handle, true
						}
					}
				}
			}

			if !ok {
				return fmt.Errorf("'%s' in path '%s': conflict with the existing param wildcard '%s' in prefix '%s'", path, full, n.path, full[:strings.Index(full, path)]+n.path)
			}

		case all:
			return fmt.Errorf("'%s' in path '%s': conflict with the existing catch-all wildcard '%s' in prefix '%s' ", path, full, n.path, full[:strings.Index(full, path)]+n.path)
		}

		if len(path) > 0 {
			if child = n.child(path[0]); child != nil {
				n.priority++
				parent.resort()
				parent, n = n, child
				continue
			} else if len(n.path) > 0 {
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
			} else {
				// Root node.
				if err = n.construct(path, full, handle); err != nil {
					return err
				}
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
		// NOTE: This don't affect tsr node, the contribution of the
		// tsr node to priority is zero.
		n.priority = 1

		switch path[0] {
		// If the node type of the current node is static or param,
		// We need add a extra node called tsr node, which is used
		// in 'get' function when the 'tsr' parameter is true.
		case ':':
			n.nt = param
			if i = strings.IndexAny(path[1:], ":*/"); i > 0 {
				// NOTE: 'i' is based on 'path[1:]', not 'path', so
				// we have got to add 1 to it.
				i++
				if path[i] != '/' {
					return fmt.Errorf("'%s' in path '%s': only one wildcard per path segment is allowed", path, full)
				}
				n.path, path = path[:i], path[i:]
				if path == "/" {
					n.tsr, n.handle = true, handle
				}

				n.children = []*node{&node{}}
				n = n.children[0]

			} else if i == -1 && len(path) > 1 {
				// Reach the end of the path, the last byte is not '/'.
				// index field doesn't make sense to param node.
				n.handle, n.path, path = handle, path, ""
				n.children = []*node{&node{path: "/", tsr: true, index: '/', handle: handle}}
			} else {
				return fmt.Errorf("'%s' in path '%s': param wildcard can't be empty", path, full)
			}

		case '*':
			n.nt = all
			if i = strings.IndexAny(path[1:], ":*/"); i != -1 {
				return fmt.Errorf("'%s' in path '%s': catch-all routes are only allowed at the end of the path", path[:i+1], full)
			} else if len(path) == 1 {
				return fmt.Errorf("'%s' in path '%s': catch-all wildcard can't be empty", path, full)
			}
			n.handle, n.path, path = handle, path, ""

		default:
			if i = strings.IndexAny(path, ":*"); i != -1 {
				// We only need to set the 'index' field of a static node,
				// there is no use for param node and all node.
				n.path, n.index, n.children, path = path[:i], path[0], []*node{&node{}}, path[i:]
				n = n.children[0]
			} else if path[len(path)-1] == '/' {
				// Reach the end of the path, the last byte is '/'.
				if len(path) > 1 {
					n.handle, n.path, n.index, n.tsr = handle, path[:len(path)-1], path[0], true
					n.children = []*node{&node{priority: n.priority}}
					n = n.children[0]
				}
				n.handle, n.path, n.index, path = handle, "/", '/', ""
			} else {
				// Reach the end of the path, the last byte is not '/'.
				n.handle, n.path, n.index, path = handle, path, path[0], ""
				n.children = []*node{&node{path: "/", tsr: true, index: '/', handle: handle}}
			}
		}
	}

	return nil
}

// Split the static node.
func (n *node) split(parent *node, i int, path string, handle XHandle) string {
	// 'i' must greater than zero and less than the length of 'n.path'.
	child := *n
	n.handle, n.tsr, path, child.path, child.index = nil, false, path[i:], n.path[i:], n.path[i]

	if len(path) > 0 {
		// The priority needn't be increased, it will be
		// handled by the parent function.
		n.path, n.children = n.path[:i], []*node{&child}
		if path == "/" {
			n.tsr, n.handle = true, handle
		}
	} else {
		n.priority, n.handle = n.priority+1, handle

		if n.path[i-1] == '/' {
			if i > 1 {
				n.path, n.tsr = n.path[:i-1], true
				n.children = []*node{&node{path: "/", index: '/', handle: handle, priority: n.priority, children: []*node{&child}}}
			} else if parent != nil {
				// In fact, the handle of parent must be nil, unless
				// I make a mistake (In this case, path must be equal
				// to "/").
				n.path, n.children = "/", []*node{&child}
				parent.tsr, parent.handle = true, handle
			}
		} else if n.path[i] == '/' {
			n.path = n.path[:i]
			if len(child.path) > 1 {
				child.path, child.index = child.path[1:], child.path[1]
				n.children = []*node{&node{path: "/", index: '/', tsr: true, handle: handle, priority: child.priority, children: []*node{&child}}}
			} else {
				// In this case, 'child.path' is equal to "/".
				child.tsr, child.handle = true, handle
				n.children = []*node{&child}
			}
		} else {
			n.path, n.children = n.path[:i], []*node{
				&child,
				&node{path: "/", tsr: true, index: '/', handle: handle},
			}
		}
	}

	return path
}

// Returns the handle registered with the given path. The values of wildcards
// are saved to a xps parameter which are ordered. tsr (trailing slash redirect)
// parameter is used to control whether get function returns a handle exists
// with an extra (without the) trailing slash for given path when it hasn't
// been registered.
func (n *node) get(path string, xps *XParams, tsr bool) XHandle {
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
				*xps = append(*xps, XParam{Key: n.path[1:], Value: path[:i]})
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
			} else {
				break outer
			}
		}

		// 'all' node type
		*xps = append(*xps, XParam{Key: n.path[1:], Value: path})
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
