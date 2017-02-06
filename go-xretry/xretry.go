// xretry.go
//
// 创建人: blinklv <blinklv@icloud.com>
// 创建日期: 2017-02-02
// 修订人: blinklv <blinklv@icloud.com>
// 修订日期: 2017-02-06

// go-xretry可以对某一操作以指数回退的方式进行重试.
package xretry

import "time"

const Version = "1.1.0"

// 这是一个辅助类型, op函数可以用其包装
// 返回的错误, 当Retry碰见该错误类型时
// 不会进行后面的重试逻辑, 而是直接退出.
// NOTE: Retry因为该错误类型返回的error是
// 原始error, 而不是FatalError包装后的error.
type FatalError struct {
	Err error
}

func (fe FatalError) Error() string {
	return fe.Err.Error()
}

// 当op返回非空的error时, Retry回对其进行重试, 重试的次数上限为
// n, 重试的时间间隔采取指数回退的方式, 基础的间隔为internval, 之后
// 的间隔为2*interval, 4*interval, 8*interval. 该函数返回最后一次
// 执行的错误信息以及重试的次数, 正常情况下返回的重试次数应该为0,
// 表示没有重试.
// NOTE:
// 1.n代表重试次数, 如果设置3. 当每次执行都失败时, 实际执行了4次, 因为初次执行不算重试.
// 2.如果n小于0则op不会被执行.
// 3.如果interval等于0则之后的所有重试间隔都为0.
func Retry(op func() error, n int, interval time.Duration) (error, int) {
	var (
		i   int
		err error
	)

	// 希望这个循环的条件不会让你感到奇怪 :)
	// 执行成功和没有重试次数都会让该循环退出.
	for i = 0; n >= 0; i++ {
		if err = op(); err == nil || i >= n {
			break
		}

		if fatalError, ok := err.(FatalError); ok {
			err = fatalError.Err
			break
		}

		time.Sleep(interval)
		interval = 2 * interval
	}

	return err, i
}
