# go-xkwfilter

![Building](https://img.shields.io/badge/building-passing-green.svg)
![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xkwfilter**是一个关键字过滤器, 它将输入字节流中的关键字替换  
为**mask**(屏蔽字节流). 该过滤器的实现基于[Aho-Corasick][ac-wiki]算法.


## 过滤规则

如果输入流中的关键字单独出现, 毫无疑问会被替换为mask.  
当输入流中的关键字以如下三种方式组合出现时只会被替换  
成一个mask.

* 一个关键字是另外一个关键字的字串.
* 两个关键字交叠.
* 两个关键字衔接.

![FilterRule](img/filter_rule.png)


## 例子

```go
    var xkwf = New("***", "hello", "world", "who", "are", "you")
    if xkwf == nil {
        // 处理错误
    }

    if n, err := xkwf.Filter(os.Stdout, os.Stdin); err != nil {
        // 处理错误
    }
```



[ac-wiki]: https://zh.wikipedia.org/wiki/AC%E8%87%AA%E5%8A%A8%E6%9C%BA%E7%AE%97%E6%B3%95
