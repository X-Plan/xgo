#go-xassert

![Building](https://img.shields.io/badge/building-passing-green.svg)
![Version](https://img.shields.io/badge/version-1.1.0-blue.svg)

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

    xassert.IsNil(t, a)         // 成功
    xassert.NotNil(t, b)        // 成功
    xassert.NotEqual(t, a, b)   // 成功
    xassert.Equal(t, a, b)      // 失败
}

```
