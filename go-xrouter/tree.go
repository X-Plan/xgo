// tree.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-05-26
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-06-13

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
	err := t.n.add(path, path, handle)
	t.rwmtx.Unlock()
	return err
}

func (t *tree) remove(path string) error {
	t.rwmtx.Lock()
	ok := t.n.remove(path)
	t.rwmtx.Unlock()

	if ok {
		return nil
	} else {
		return fmt.Errorf("path (%s) hasn't been registered", path)
	}
}

func (t *tree) get(path string, enableTSR bool) (h XHandle, xps XParams, tsr tsrType) {
	t.rwmtx.RLock()
	if len(t.n.path) > 0 {
		h, xps, tsr = t.n.get(path, enableTSR)
	}
	t.rwmtx.RUnlock()
	return
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
	if len(path) == 0 {
		return fmt.Errorf("path argument is empty")
	}

	if len(n.path) == 0 {
		// New node, may be it's just only used for root node.
		if err = n.construct(path, full, handle); err != nil {
			*n = node{}
		}
		return
	}

	switch n.nt {
	case static:
		i := lcp(path, n.path)
		if i < len(path) {
			if i == 0 && (path[i] == ':' || path[i] == '*') {
				err = fmt.Errorf("'%s' in path '%s': wildcard conflicts with the existing path segment '%s' in prefix '%s'", path[i:], full, n.path[i:], full[:strings.Index(full, path)]+n.path)
				break
			}

			if i < len(n.path) {
				defer n.combine(&err)
				if err = n.split(i, nil); err != nil {
					break
				}
			}
			err = n.next(i, path, full, handle)
		} else if i < len(n.path) && i == len(path) {
			defer n.combine(&err)
			err = n.split(i, handle)
		} else if n.handle == nil {
			// i == len(n.path) == len(path)
			n.handle, n.priority = handle, n.priority+1
		} else {
			err = fmt.Errorf("path '%s' has already been registered", full)
		}
	case param:
		i := lcp(path, n.path)
		if i == len(n.path) && i < len(path) {
			err = n.next(i, path, full, handle)
		} else if i == len(n.path) && i == len(path) {
			if n.handle == nil {
				n.handle, n.priority = handle, n.priority+1
			} else {
				err = fmt.Errorf("path '%s' has already been registered", full)
			}
		} else if i == len(n.path)-1 && n.path[len(n.path)-1] == '/' {
			defer n.combine(&err)
			err = n.split(i, handle)
		} else {
			err = fmt.Errorf("'%s' in path '%s': conflict with existing param wildcard '%s' in prefix '%s'", path, full, n.path, full[:strings.Index(full, path)]+n.path)
		}

	case all:
		err = fmt.Errorf("'%s' in path '%s': conflict with the existing catch-all wildcard '%s' in prefix '%s' ", path, full, n.path, full[:strings.Index(full, path)]+n.path)
	}

	return
}

// Remove a register handle. I implement this function by recursively calling.
// The path parameter must match a existing path exactly, It won't consider
// wildcard and trailing slash. If the path '/foo/:bar/who/are/*you' has been
// registerd, we can't use the argument '/foo/:xxx/who/are/*ooo' to remove it.
func (n *node) remove(path string) (ok bool) {
	var i = lcp(n.path, path)
	if i == len(n.path) && i == len(path) {
		if n.handle != nil {
			n.priority, n.handle = n.priority-1, nil
			if len(n.children) == 1 {
				n.concat()
			}
			ok = true
		}
	} else if i == len(n.path) && i < len(path) {
		var k int
		for k = 0; k < len(n.children); k++ {
			if n.children[k].path[0] == path[i] {
				break
			}
		}

		if k < len(n.children) && n.children[k].priority == 0 {
			if ok = n.children[k].remove(path[i:]); ok {
				n.children = append(n.children[:i], n.children[i+1:]...)
				n.priority--
				n.setMaxParams()

				if len(n.children) == 1 && n.handle == nil {
					n.concat()
				}
			}
		}
	}

	return
}

// Concate the current node with the child node, the current node
// must only have one child node.
func (n *node) concat() {
	child := n.children[0]
	if child.nt == static {
		if n.nt == static || (n.nt == param && child.path == "/") {
			n.path += child.path
			n.children, n.handle = child.children, child.handle
		}
	}
}

// Move to next child node (If not exist, create it).
func (n *node) next(i int, path, full string, handle XHandle) (err error) {
	if i < len(path) {
		if len(n.children) > 0 && (path[i] == ':' || path[i] == '*') {
			// If the children of a node have one static node, all children
			// must be static. If a node exists one static node, will produce
			// a conflict.
			if n.children[0].nt == static {
				// The result must be non-nil.
				err = n.children[0].add(path[i:], full, handle)
				return
			}
		}

		if child := n.child(path[i]); child != nil {
			if err = child.add(path[i:], full, handle); err == nil {
				n.resort()
				n.maxParams = max(n.maxParams, child.maxParams)
				n.priority++
			}
		} else {
			child := &node{}
			if err = child.construct(path[i:], full, handle); err == nil {
				// We don't need to resort children, because the priority
				// of new node is equal to 1 (minimum).
				n.children = append(n.children, child)
				n.maxParams = max(n.maxParams, child.maxParams)
				n.priority++
			}
		}
	}
	return
}

// Init a empty node.
func (n *node) construct(path string, full string, handle XHandle) (err error) {
	var i int
	n.priority = 1
	switch path[0] {
	case ':':
		n.nt = param
		if i = strings.IndexAny(path[1:], ":*/"); i > 0 {
			// NOTE: 'i' is based on 'path[1:]', not 'path', so
			// we have got to add 1 to it.
			i++
			if path[i] != '/' {
				err = fmt.Errorf("'%s' in path '%s': only one wildcard per path segment is allowed", path, full)
				break
			}

			i++
			n.path, n.maxParams = path[:i], 1
			if i < len(path) {
				child := &node{}
				if err = child.construct(path[i:], full, handle); err == nil {
					n.children = []*node{child}
					n.maxParams += child.maxParams
				}
			} else {
				n.handle = handle
			}
		} else if i == -1 && len(path) > 1 {
			n.path, n.maxParams, n.handle = path, 1, handle
		} else {
			err = fmt.Errorf("'%s' in path '%s': param wildcard can't be empty", path, full)
		}
	case '*':
		n.nt = all
		if i = strings.IndexAny(path[1:], ":*/"); i != -1 {
			err = fmt.Errorf("'%s' in path '%s': catch-all routes are only allowed at the end of the path", path[:i+1], full)
		} else if len(path) == 1 {
			err = fmt.Errorf("'%s' in path '%s': catch-all wildcard can't be empty", path, full)
		} else {
			n.path, n.maxParams, n.handle = path, 1, handle
		}
	default:
		if i = strings.IndexAny(path, ":*"); i != -1 {
			// We only need to set the 'index' field of a static node,
			// there is no use for param node and all node.
			n.path, n.index = path[:i], path[0]
			child := &node{}
			if err = child.construct(path[i:], full, handle); err == nil {
				n.children, n.maxParams = []*node{child}, child.maxParams
			}
		} else {
			n.path, n.index, n.handle = path, path[0], handle
		}
	}

	return
}

func (n *node) split(i int, handle XHandle) error {
	if i > 0 {
		child := *n
		child.path, child.index = n.path[i:], n.path[i]
		n.path, n.children = n.path[:i], []*node{&child}

		if n.nt == param {
			child.nt, child.maxParams = static, child.maxParams-1
		}

		if handle != nil {
			n.handle, n.priority = handle, n.priority+1
		} else {
			n.handle = nil
		}
	}
	return nil
}

func (n *node) setMaxParams() {
	n.maxParams = 0
	for _, child := range n.children {
		if child.maxParams > n.maxParams {
			n.maxParams = child.maxParams
		}
	}

	if n.nt == param || n.nt == all {
		n.maxParams++
	}
}

// The inverse operator of split function, used in recovery.
func (n *node) combine(err *error) {
	if *err != nil {
		child := n.children[0]
		n.path += child.path
		n.children = child.children
		if n.handle != nil {
			n.priority--
		}
		n.handle = child.handle
	}
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
			if i == len(path)-1 && n.handle != nil {
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

func max(a, b uint8) uint8 {
	if a >= b {
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

// The following functions/methods are used to debug.

// Print a node recursively.
func (n *node) print(indent int) {
	// Format:
	// priority:[index] path (nt:maxParams:handle)
	fmt.Printf("%s%d:[%c] %s (%s:%d:%v)\n", strings.Repeat(" ", indent), n.priority, n.index, n.path, n.nt, n.maxParams, n.handle)

	for _, child := range n.children {
		child.print(indent + 1)
	}
}

// Check the priority of a node recursively.
func (n *node) checkPriority() error {
	var sum, last uint32
	if len(n.children) > 0 {
		last = n.children[0].priority
	}
	for _, child := range n.children {
		if child.priority > last {
			return fmt.Errorf("The children's order of node (%s:%s) is invalid", n.nt, n.path)
		}
		sum += child.priority
		last = child.priority
		if err := child.checkPriority(); err != nil {
			return err
		}
	}

	if n.handle != nil {
		sum++
	}

	if sum != n.priority {
		return fmt.Errorf("The total priority (%d) of children is not equal to the priority (%d) of node (%s:%s)", sum, n.priority, n.nt, n.path)
	}

	return nil
}

// Check the maxParams of a node recursively.
func (n *node) checkMaxParams() error {
	var max uint8
	for _, child := range n.children {
		if err := child.checkMaxParams(); err != nil {
			return err
		}

		if child.maxParams > max {
			max = child.maxParams
		}
	}

	if n.nt != static {
		max++
	}

	if n.maxParams != max {
		return fmt.Errorf("The maxParams of node (%s:%s) is invalid", n.nt, n.path)
	}

	return nil
}
