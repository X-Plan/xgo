# go-xassert

![Building](https://img.shields.io/badge/building-passing-green.svg)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

**go-xassert** is a assert package used to test.


## Usage

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

    xassert.IsTrue(t, true)     // success
    xassert.IsFalse(t, false)   // success
    xassert.IsNil(t, a)         // success
    xassert.NotNil(t, b)        // success
    xassert.NotEqual(t, a, b)   // success
    xassert.Equal(t, a, b)      // fail

    // regular expression match.
	xassert.Match(t, errors.New("Hello World"), `[Hh]ello\s+[Ww]orld`)
	xassert.NotMatch(t, errors.New("Are You OK?"), `You\s{2}`)
}

```
