// tree.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-01
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-03-01

package xrouter

type nodeType uint8

const (
	static nodeType = iota // default node type
	param                  // ':name' wildcard node type
	all                    // '*name' wildcard node type
)

type node struct {
	path     string
	nt       nodeType
	priority uint32
	children []*node
	handle   XHandle
}

// Returns the handle registered with the given path. The values of wildcards
// are saved to a XParams variable which are ordered. tsr (trailing slash redirect)
// parameter is used to control whether get function returns a handle exists
// with an extra (without the) trailing slash for given path when it hasn't
// been registered.
func (n *node) get(path string, tsr bool) (handle XHandle, xps XParams) {
	switch n.nt {
	case static:
	case param:
	case all:
	default:
		// Unless I make a mistake, this statement can't be executed.
		panic(fmt.Sprintf("invalid node type (%d)", n.nt))
	}
	return nil, nil
}
