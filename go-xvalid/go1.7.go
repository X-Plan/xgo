// go1.7.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-27
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-02-27

// +build go1.7

package xvalid

import "reflect"

// go1.7引入了StructTag的Lookup方法, 可以参照: https://golang.org/doc/go1.7
func tagLookup(tag reflect.StructTag, key string) (string, bool) {
	return tag.Lookup(key)
}
