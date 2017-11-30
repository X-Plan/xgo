// xsched_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-12
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-11-30

package xsched

import (
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"log"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEmpty(t *testing.T) {
	xs, err := New(nil)
	xassert.IsNil(t, err)
	xassert.NotNil(t, xs)

	address, err := xs.Get()
	xassert.Match(t, err, `all hosts are temporarily unavailable`)
	xassert.IsTrue(t, len(address) == 0)
}

func TestParseAddrUnit(t *testing.T) {
	var strs = []struct {
		str     string
		ok      bool
		address string
		weight  int
	}{
		{"192.168.1.1:333:-20", false, "", 0},
		{"192.168.1.1:aaa:10", false, "", 0},
		{"10.13.1.10:bbbb:ccc", false, "", 0},
		{"10.13.1.10:80:20:20", false, "", 0},
		{"127.0.0.1:80:10", true, "127.0.0.1:80", 10},
		{"  10.13.1.10:8080:0 ", true, "10.13.1.10:8080", 0},
		{"172.187.23.10:8192:27", true, "172.187.23.10:8192", 27},
	}

	for _, str := range strs {
		if str.ok {
			u := newAddrUnit(str.str)
			xassert.NotNil(t, u)
			xassert.Equal(t, u.address, str.address)
			xassert.Equal(t, u.weight, str.weight)
			xassert.Equal(t, u.available, true)
			xassert.Equal(t, u.total, 0)
			xassert.Equal(t, u.fail, 0)
			xassert.Equal(t, u.samplePeriod, maxSamplePeriod)
			xassert.Equal(t, u.waitInterval, minWaitInterval)
		} else {
			xassert.IsNil(t, newAddrUnit(str.str))
		}
	}
}

func TestAddrUnit(t *testing.T) {
	u := newAddrUnit("127.0.0.1:80:10")
	xassert.NotNil(t, u)

	var (
		seed                       = createResultSeed(0.2)
		last, current              = false, true
		samplePeriod, waitInterval time.Duration
		i                          = 0
		l                          = log.New(os.Stdout, "", log.Ldate|log.Ltime)
	)

	for {
		current = u.IsAvailable()
		if !last && current {
			l.Printf("Sample sample-interval: %s wait-interval: %s\n", u.samplePeriod, u.waitInterval)
		} else if last && !current {
			l.Printf("Wait sample-interval: %s wait-interval: %s\n", u.samplePeriod, u.waitInterval)
			i++
			if i == 7 {
				seed = createResultSeed(0.05)
				l.Printf("Change Seed")
			}
		} else if u.samplePeriod != samplePeriod || u.waitInterval != waitInterval {
			l.Printf("Sample sample-interval: %s wait-interval: %s\n", u.samplePeriod, u.waitInterval)
		}

		samplePeriod, waitInterval = u.samplePeriod, u.waitInterval
		if current {
			u.Feedback(seed())
		}

		last = current
		time.Sleep(10 * time.Millisecond)
	}
}

func TestErrorNew(t *testing.T) {
	var strs = []string{
		"192.168.1.10:80:10",
		"192.168.1.11:80:10",
		"192.168.1.12:8080:-20",
		"192.168.1.13:8192:10",
		"192.168.1.14:80:10",
	}

	xs, err := New(strs)
	xassert.IsNil(t, xs)
	xassert.Match(t, err, `invalid address \(192\.168\.1\.12:8080:-20\)`)

	strs[2], strs[3] = "192.168.1.12:8080:12", "192.168.1.13:AAA:10"
	xs, err = New(strs)
	xassert.IsNil(t, xs)
	xassert.Match(t, err, `invalid address \(192\.168\.1\.13:AAA:10\)`)

	strs[3], strs[4] = "192.168.1.13:8192:10", "192.168-1.8:80:10"
	xs, err = New(strs)
	xassert.IsNil(t, xs)
	xassert.Match(t, err, `invalid address \(192\.168-1\.8:80:10\)`)
}

func TestCorrectNew(t *testing.T) {
	var strs = []string{
		"192.168.1.10:80:10",
		"192.168.1.11:80:10",
		"192.168.1.12:80:20",
		"192.168.1.13:80:30",
		"192.168.1.14:80:30",
		"192.168.1.15:80:50",
		"192.168.1.16:80:20",
		"192.168.1.17:80:70",
		"192.168.1.18:80:35",
		"192.168.1.19:80:0",
		"192.168.1.20:80:45",
		"192.168.1.11:80:20",
	}

	var results = []struct {
		address string
		weight  int
	}{
		{"192.168.1.10:80", 2},
		{"192.168.1.12:80", 4},
		{"192.168.1.13:80", 6},
		{"192.168.1.14:80", 6},
		{"192.168.1.15:80", 10},
		{"192.168.1.16:80", 4},
		{"192.168.1.17:80", 14},
		{"192.168.1.18:80", 7},
		{"192.168.1.20:80", 9},
		{"192.168.1.11:80", 4},
	}

	xs, err := New(strs)
	xassert.NotNil(t, xs)
	xassert.IsNil(t, err)

	xassert.Equal(t, len(results), len(strs)-2)
	n := len(results)
	xassert.Equal(t, len(xs.addrs), n)
	xassert.Equal(t, len(xs.addrm), n)
	xassert.Equal(t, xs.n, n)

	for i, result := range results {
		xassert.Equal(t, xs.addrs[i].address, result.address)
		xassert.Equal(t, xs.addrs[i].weight, result.weight)
	}
}

type addrCounter struct {
	weight int
	seed   func() bool
	total  int64
}

type addrCounters map[string]*addrCounter

func (acs addrCounters) CreateStrs() []string {
	var strs []string
	for address, ac := range acs {
		strs = append(strs, address+":"+strconv.Itoa(ac.weight))
	}
	return strs
}

func TestXScheduler(t *testing.T) {
	acs := addrCounters(map[string]*addrCounter{
		"192.168.1.1:80":  &addrCounter{10, createResultSeed(0.05), 0},
		"192.168.1.2:80":  &addrCounter{10, createResultSeed(0.05), 0},
		"192.168.1.3:80":  &addrCounter{40, createResultSeed(0.05), 0},
		"192.168.1.4:80":  &addrCounter{20, createResultSeed(0.3), 0},
		"192.168.1.5:80":  &addrCounter{25, createResultSeed(0.05), 0},
		"192.168.1.6:80":  &addrCounter{5, createResultSeed(0.05), 0},
		"192.168.1.7:80":  &addrCounter{50, createResultSeed(0.2), 0},
		"192.168.1.8:80":  &addrCounter{30, createResultSeed(0.05), 0},
		"192.168.1.9:80":  &addrCounter{10, createResultSeed(0.05), 0},
		"192.168.1.10:80": &addrCounter{35, createResultSeed(0.05), 0},
	})

	xs, err := New(acs.CreateStrs())
	xassert.NotNil(t, xs)
	xassert.IsNil(t, err)

	wg := &sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < 5000; j++ {
				address, err := xs.Get()
				if err != nil {
					fmt.Println(err)
					return
				}
				ac := acs[address]
				xs.Feedback(address, ac.seed())
				atomic.AddInt64(&(ac.total), int64(1))
				time.Sleep(100 * time.Millisecond)
			}
		}()
	}

	wg.Wait()

	for address, ac := range acs {
		fmt.Println(address, ac.weight, ac.total)
	}
}

func createResultSeed(failRate float64) func() bool {
	var (
		i        = 0
		thresold = 100 - int(failRate*100.0)
	)
	return func() bool {
		i = (i + 1) % 100
		if i < thresold {
			return true
		} else {
			return false
		}
	}
}

func testGCD(t *testing.T) {
	xassert.Equal(t, gcd(12, 3), 3)
	xassert.Equal(t, gcd(10, 100), 10)
	xassert.Equal(t, gcd(101, 9), 1)
	xassert.Equal(t, gcd(100, 0), 100)
}

func TestUpdate(t *testing.T) {
	var strs = []string{
		"192.168.1.10:80:10",
		"192.168.1.11:80:10",
		"192.168.1.12:80:20",
		"192.168.1.13:80:30",
		"192.168.1.14:80:30",
		"192.168.1.15:80:50",
		"192.168.1.16:80:20",
		"192.168.1.17:80:70",
	}

	// Update the weight of an existing address.
	for i, str := range strs {
		xs1, _ := New(strs)
		xassert.NotNil(t, xs1)

		strs[i] = str[:len(str)-2] + randomEven()
		xs2, _ := New(strs)
		xassert.NotNil(t, xs2)

		xassert.IsNil(t, xs1.Update(strs[i]))
		xassert.IsNil(t, xs1.equal(xs2))
	}

	newStrs := strs
	// Add new addresses.
	for i := 18; i < 25; i++ {
		xs1, _ := New(newStrs)
		xassert.NotNil(t, xs1)

		address := fmt.Sprintf("192.168.1.%d:80:%s", i, randomEven())
		newStrs = append(newStrs, address)
		xs2, _ := New(newStrs)
		xassert.NotNil(t, xs2)

		xassert.IsNil(t, xs1.Update(address))
		xassert.IsNil(t, xs1.equal(xs2))
	}
}

func TestRemove(t *testing.T) {
	var strs = []string{
		"192.168.1.10:80:10",
		"192.168.1.11:80:10",
		"192.168.1.12:80:20",
		"192.168.1.13:80:30",
		"192.168.1.14:80:30",
		"192.168.1.15:80:50",
		"192.168.1.16:80:20",
		"192.168.1.17:80:70",
	}

	for len(strs) > 0 {
		xs1, _ := New(strs)
		xassert.NotNil(t, xs1)

		i := int(rand.Uint32()) % len(strs)
		address := strs[i]
		strs = append(strs[:i], strs[i+1:]...)

		xs2, _ := New(strs)
		xassert.NotNil(t, xs2)

		xassert.IsNil(t, xs1.Remove(address))
		xassert.IsNil(t, xs1.equal(xs2))
	}
}

// Generate an even number between 2 and 50.
func randomEven() string {
	return fmt.Sprintf("%d", (rand.Uint32()%50)/2*2+2)
}

// Check whether two schedulers are equal, this method is only used to debug.
func (xs *XScheduler) equal(x *XScheduler) error {
	if len(xs.addrs) != len(x.addrs) {
		return notEqualErrorf("addrs-number", len(xs.addrs), len(x.addrs))
	}

	if len(xs.addrm) != len(x.addrm) {
		return notEqualErrorf("addrm-number", len(xs.addrm), len(x.addrm))
	}

	if xs.n != x.n {
		return notEqualErrorf("n", xs.n, x.n)
	}

	if xs.max != x.max {
		return notEqualErrorf("max", xs.max, x.max)
	}

	if xs.delta != x.delta {
		return notEqualErrorf("delta", xs.delta, x.delta)
	}

	if xs.i != x.i {
		return notEqualErrorf("i", xs.i, x.i)
	}

	if xs.cw != x.cw {
		return notEqualErrorf("cw", xs.cw, x.cw)
	}

	for address, u := range xs.addrm {
		if unit := x.addrm[address]; unit != nil {
			if err := u.equal(unit); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("second scheduler doesn't have address (%s)", address)
		}
	}

	return nil
}

// Check whether two units are equal, this method is only used to debug.
func (u *addrUnit) equal(x *addrUnit) error {
	if u.address != x.address {
		return notEqualErrorf("address", u.address, x.address)
	}

	if u.weight != x.weight {
		return notEqualErrorf("weight", u.weight, x.weight)
	}

	if u.available != x.available {
		return notEqualErrorf("available", u.available, x.available)
	}

	if u.total != x.total {
		return notEqualErrorf("total", u.total, x.total)
	}

	if u.fail != x.fail {
		return notEqualErrorf("fail", u.fail, x.fail)
	}

	if u.samplePeriod != x.samplePeriod {
		return notEqualErrorf("samplePeriod", u.samplePeriod, x.samplePeriod)
	}

	if u.waitInterval != x.waitInterval {
		return notEqualErrorf("waitInterval", u.waitInterval, x.waitInterval)
	}

	return nil
}

func notEqualErrorf(name string, v1, v2 interface{}) error {
	return fmt.Errorf("%s-1 (%v) is not equal to %s-2 (%v)", name, v1, name, v2)
}
