# go-xassert

![Building](https://img.shields.io/badge/building-passing-green.svg)
![Version](https://img.shields.io/badge/version-1.2.2-blue.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xassert**实现了一个方便测试的断言包.


## 例子

```go
type foo struct {
    x int
    b string
}

type bar struct {
    a int
    b string
    c map[int]string
    d foo
}

func Test(t *testing.T) {
    var (
        a map[int]string
        b = bar {
            a: 1,
            b: "hello world",
            c: make(map[int]string),
            d: foo{
                x: 1,
                b: "who are you?",
            },
        }
    )

    xassert.IsTrue(t, true)     // 成功
    xassert.IsFalse(t, false)   // 成功
    xassert.IsNil(t, a)         // 成功
    xassert.NotNil(t, b)        // 成功
    xassert.NotEqual(t, a, b)   // 成功
    xassert.Equal(t, a, b)      // 失败

    // 正则匹配.
	xassert.Match(t, errors.New("Hello World"), `[Hh]ello\s+[Ww]orld`)
	xassert.NotMatch(t, errors.New("Are You OK?"), `You\s{2}`)
}

```
