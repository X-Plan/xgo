// xrouter_test.go
//
// Author: blinklv <blinklv@icloud.com>
// Create Time: 2017-03-23
// Maintainer: blinklv <blinklv@icloud.com>
// Last Change: 2017-05-25

package xrouter

import (
	"encoding/json"
	"fmt"
	"github.com/X-Plan/xgo/go-xassert"
	"github.com/X-Plan/xgo/go-xrandstring"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestXParam(t *testing.T) {
	xassert.IsTrue(t, fmt.Sprintf("%s", XParam{"hello", "world"}) == "hello=world")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParam{"foo", "bar"}) == "foo=bar")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParam{"boy", "girl"}) == "boy=girl")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParam{"name", "age"}) == "name=age")
}

func TestXParams(t *testing.T) {
	xassert.IsTrue(t, fmt.Sprintf("%s", XParams{}) == "")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParams{XParam{"Who", "Are"}}) == "Who=Are")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParams{XParam{"Who", "Are"}, XParam{"You", "Am"}}) == "Who=Are,You=Am")
	xassert.IsTrue(t, fmt.Sprintf("%s", XParams{XParam{"Who", "Are"}, XParam{"You", "Am"}, XParam{"I", "Alone"}}) == "Who=Are,You=Am,I=Alone")
}

func TestSupportMethod(t *testing.T) {
	ms := []string{"get", "Post", "hEad", "puT", "optIons", "patch", "delete"}
	for _, m := range ms {
		xassert.IsTrue(t, SupportMethod(m))
	}

	for _, m := range methods {
		xassert.IsTrue(t, SupportMethod(m))
	}

	for _, m := range methods {
		xassert.IsFalse(t, SupportMethod(xrandstring.Replace(m, "X")))
	}
}

func TestNew(t *testing.T) {
	xassert.IsNil(t, New(nil))
	xr := New(&XConfig{})
	for _, method := range methods {
		xassert.NotNil(t, xr.trees[method])
	}
}

func TestHandle(t *testing.T) {
	xr := New(&XConfig{})
	for _, method := range methods {
		paths, _ := generatePaths(100, 3, 6)
		for _, path := range paths {
			xassert.IsNil(t, xr.Handle(method, path, generateXHandle(path)))
		}
	}

	// Test the path duplicate case, which is produced by CleanPath function.
	xassert.IsNil(t, xr.Handle("POST", "/hello/world/", generateXHandle("/hello/world/")))
	xassert.NotNil(t, xr.Handle("post", "/hello/xxx/../world/", generateXHandle("/hello/xxx/../world/")))
	xassert.IsNil(t, xr.Handle("GET", "/hello/xxx/../world/", generateXHandle("/hello/xxx/../world/")))

	// Test the unsupported methods.
	ums := []string{"POXX", "GEX", "XATCH", "FOO", "bar"}
	for _, um := range ums {
		xassert.Match(t, xr.Handle(um, "/foo", nil), fmt.Sprintf(`http method \(%s\) is unsupported`, um))
	}
}

func TestServeHTTP(t *testing.T) {
	var (
		// Github paths, I copy this from https://github.com/julienschmidt/go-http-routing-benchmark
		pairs = []requestPair{
			// OAuth Authorizations
			{"GET", "/authorizations"},
			{"GET", "/authorizations/:id"},
			{"POST", "/authorizations"},
			//{"PUT", "/authorizations/clients/:client_id"},
			//{"PATCH", "/authorizations/:id"},
			{"DELETE", "/authorizations/:id"},
			{"GET", "/applications/:client_id/tokens/:access_token"},
			{"DELETE", "/applications/:client_id/tokens"},
			{"DELETE", "/applications/:client_id/tokens/:access_token"},

			// Activity
			{"GET", "/events"},
			{"GET", "/repos/:owner/:repo/events"},
			{"GET", "/networks/:owner/:repo/events"},
			{"GET", "/orgs/:org/events"},
			{"GET", "/users/:user/received_events"},
			{"GET", "/users/:user/received_events/public"},
			{"GET", "/users/:user/events"},
			{"GET", "/users/:user/events/public"},
			{"GET", "/users/:user/events/orgs/:org"},
			{"GET", "/feeds"},
			{"GET", "/notifications"},
			{"GET", "/repos/:owner/:repo/notifications"},
			{"PUT", "/notifications"},
			{"PUT", "/repos/:owner/:repo/notifications"},
			{"GET", "/notifications/threads/:id"},
			//{"PATCH", "/notifications/threads/:id"},
			{"GET", "/notifications/threads/:id/subscription"},
			{"PUT", "/notifications/threads/:id/subscription"},
			{"DELETE", "/notifications/threads/:id/subscription"},
			{"GET", "/repos/:owner/:repo/stargazers"},
			{"GET", "/users/:user/starred"},
			{"GET", "/user/starred"},
			{"GET", "/user/starred/:owner/:repo"},
			{"PUT", "/user/starred/:owner/:repo"},
			{"DELETE", "/user/starred/:owner/:repo"},
			{"GET", "/repos/:owner/:repo/subscribers"},
			{"GET", "/users/:user/subscriptions"},
			{"GET", "/user/subscriptions"},
			{"GET", "/repos/:owner/:repo/subscription"},
			{"PUT", "/repos/:owner/:repo/subscription"},
			{"DELETE", "/repos/:owner/:repo/subscription"},
			{"GET", "/user/subscriptions/:owner/:repo"},
			{"PUT", "/user/subscriptions/:owner/:repo"},
			{"DELETE", "/user/subscriptions/:owner/:repo"},

			// Gists
			{"GET", "/users/:user/gists"},
			{"GET", "/gists"},
			//{"GET", "/gists/public"},
			//{"GET", "/gists/starred"},
			{"GET", "/gists/:id"},
			{"POST", "/gists"},
			//{"PATCH", "/gists/:id"},
			{"PUT", "/gists/:id/star"},
			{"DELETE", "/gists/:id/star"},
			{"GET", "/gists/:id/star"},
			{"POST", "/gists/:id/forks"},
			{"DELETE", "/gists/:id"},

			// Git Data
			{"GET", "/repos/:owner/:repo/git/blobs/:sha"},
			{"POST", "/repos/:owner/:repo/git/blobs"},
			{"GET", "/repos/:owner/:repo/git/commits/:sha"},
			{"POST", "/repos/:owner/:repo/git/commits"},
			//{"GET", "/repos/:owner/:repo/git/refs/*ref"},
			{"GET", "/repos/:owner/:repo/git/refs"},
			{"POST", "/repos/:owner/:repo/git/refs"},
			//{"PATCH", "/repos/:owner/:repo/git/refs/*ref"},
			//{"DELETE", "/repos/:owner/:repo/git/refs/*ref"},
			{"GET", "/repos/:owner/:repo/git/tags/:sha"},
			{"POST", "/repos/:owner/:repo/git/tags"},
			{"GET", "/repos/:owner/:repo/git/trees/:sha"},
			{"POST", "/repos/:owner/:repo/git/trees"},

			// Issues
			{"GET", "/issues"},
			{"GET", "/user/issues"},
			{"GET", "/orgs/:org/issues"},
			{"GET", "/repos/:owner/:repo/issues"},
			{"GET", "/repos/:owner/:repo/issues/:number"},
			{"POST", "/repos/:owner/:repo/issues"},
			//{"PATCH", "/repos/:owner/:repo/issues/:number"},
			{"GET", "/repos/:owner/:repo/assignees"},
			{"GET", "/repos/:owner/:repo/assignees/:assignee"},
			{"GET", "/repos/:owner/:repo/issues/:number/comments"},
			//{"GET", "/repos/:owner/:repo/issues/comments"},
			//{"GET", "/repos/:owner/:repo/issues/comments/:id"},
			{"POST", "/repos/:owner/:repo/issues/:number/comments"},
			//{"PATCH", "/repos/:owner/:repo/issues/comments/:id"},
			//{"DELETE", "/repos/:owner/:repo/issues/comments/:id"},
			{"GET", "/repos/:owner/:repo/issues/:number/events"},
			//{"GET", "/repos/:owner/:repo/issues/events"},
			//{"GET", "/repos/:owner/:repo/issues/events/:id"},
			{"GET", "/repos/:owner/:repo/labels"},
			{"GET", "/repos/:owner/:repo/labels/:name"},
			{"POST", "/repos/:owner/:repo/labels"},
			//{"PATCH", "/repos/:owner/:repo/labels/:name"},
			{"DELETE", "/repos/:owner/:repo/labels/:name"},
			{"GET", "/repos/:owner/:repo/issues/:number/labels"},
			{"POST", "/repos/:owner/:repo/issues/:number/labels"},
			{"DELETE", "/repos/:owner/:repo/issues/:number/labels/:name"},
			{"PUT", "/repos/:owner/:repo/issues/:number/labels"},
			{"DELETE", "/repos/:owner/:repo/issues/:number/labels"},
			{"GET", "/repos/:owner/:repo/milestones/:number/labels"},
			{"GET", "/repos/:owner/:repo/milestones"},
			{"GET", "/repos/:owner/:repo/milestones/:number"},
			{"POST", "/repos/:owner/:repo/milestones"},
			//{"PATCH", "/repos/:owner/:repo/milestones/:number"},
			{"DELETE", "/repos/:owner/:repo/milestones/:number"},

			// Miscellaneous
			{"GET", "/emojis"},
			{"GET", "/gitignore/templates"},
			{"GET", "/gitignore/templates/:name"},
			{"POST", "/markdown"},
			{"POST", "/markdown/raw"},
			{"GET", "/meta"},
			{"GET", "/rate_limit"},

			// Organizations
			{"GET", "/users/:user/orgs"},
			{"GET", "/user/orgs"},
			{"GET", "/orgs/:org"},
			//{"PATCH", "/orgs/:org"},
			{"GET", "/orgs/:org/members"},
			{"GET", "/orgs/:org/members/:user"},
			{"DELETE", "/orgs/:org/members/:user"},
			{"GET", "/orgs/:org/public_members"},
			{"GET", "/orgs/:org/public_members/:user"},
			{"PUT", "/orgs/:org/public_members/:user"},
			{"DELETE", "/orgs/:org/public_members/:user"},
			{"GET", "/orgs/:org/teams"},
			{"GET", "/teams/:id"},
			{"POST", "/orgs/:org/teams"},
			//{"PATCH", "/teams/:id"},
			{"DELETE", "/teams/:id"},
			{"GET", "/teams/:id/members"},
			{"GET", "/teams/:id/members/:user"},
			{"PUT", "/teams/:id/members/:user"},
			{"DELETE", "/teams/:id/members/:user"},
			{"GET", "/teams/:id/repos"},
			{"GET", "/teams/:id/repos/:owner/:repo"},
			{"PUT", "/teams/:id/repos/:owner/:repo"},
			{"DELETE", "/teams/:id/repos/:owner/:repo"},
			{"GET", "/user/teams"},

			// Pull Requests
			{"GET", "/repos/:owner/:repo/pulls"},
			{"GET", "/repos/:owner/:repo/pulls/:number"},
			{"POST", "/repos/:owner/:repo/pulls"},
			//{"PATCH", "/repos/:owner/:repo/pulls/:number"},
			{"GET", "/repos/:owner/:repo/pulls/:number/commits"},
			{"GET", "/repos/:owner/:repo/pulls/:number/files"},
			{"GET", "/repos/:owner/:repo/pulls/:number/merge"},
			{"PUT", "/repos/:owner/:repo/pulls/:number/merge"},
			{"GET", "/repos/:owner/:repo/pulls/:number/comments"},
			//{"GET", "/repos/:owner/:repo/pulls/comments"},
			//{"GET", "/repos/:owner/:repo/pulls/comments/:number"},
			{"PUT", "/repos/:owner/:repo/pulls/:number/comments"},
			//{"PATCH", "/repos/:owner/:repo/pulls/comments/:number"},
			//{"DELETE", "/repos/:owner/:repo/pulls/comments/:number"},

			// Repositories
			{"GET", "/user/repos"},
			{"GET", "/users/:user/repos"},
			{"GET", "/orgs/:org/repos"},
			{"GET", "/repositories"},
			{"POST", "/user/repos"},
			{"POST", "/orgs/:org/repos"},
			{"GET", "/repos/:owner/:repo"},
			//{"PATCH", "/repos/:owner/:repo"},
			{"GET", "/repos/:owner/:repo/contributors"},
			{"GET", "/repos/:owner/:repo/languages"},
			{"GET", "/repos/:owner/:repo/teams"},
			{"GET", "/repos/:owner/:repo/tags"},
			{"GET", "/repos/:owner/:repo/branches"},
			{"GET", "/repos/:owner/:repo/branches/:branch"},
			{"DELETE", "/repos/:owner/:repo"},
			{"GET", "/repos/:owner/:repo/collaborators"},
			{"GET", "/repos/:owner/:repo/collaborators/:user"},
			{"PUT", "/repos/:owner/:repo/collaborators/:user"},
			{"DELETE", "/repos/:owner/:repo/collaborators/:user"},
			{"GET", "/repos/:owner/:repo/comments"},
			{"GET", "/repos/:owner/:repo/commits/:sha/comments"},
			{"POST", "/repos/:owner/:repo/commits/:sha/comments"},
			{"GET", "/repos/:owner/:repo/comments/:id"},
			//{"PATCH", "/repos/:owner/:repo/comments/:id"},
			{"DELETE", "/repos/:owner/:repo/comments/:id"},
			{"GET", "/repos/:owner/:repo/commits"},
			{"GET", "/repos/:owner/:repo/commits/:sha"},
			{"GET", "/repos/:owner/:repo/readme"},
			//{"GET", "/repos/:owner/:repo/contents/*path"},
			//{"PUT", "/repos/:owner/:repo/contents/*path"},
			//{"DELETE", "/repos/:owner/:repo/contents/*path"},
			//{"GET", "/repos/:owner/:repo/:archive_format/:ref"},
			{"GET", "/repos/:owner/:repo/keys"},
			{"GET", "/repos/:owner/:repo/keys/:id"},
			{"POST", "/repos/:owner/:repo/keys"},
			//{"PATCH", "/repos/:owner/:repo/keys/:id"},
			{"DELETE", "/repos/:owner/:repo/keys/:id"},
			{"GET", "/repos/:owner/:repo/downloads"},
			{"GET", "/repos/:owner/:repo/downloads/:id"},
			{"DELETE", "/repos/:owner/:repo/downloads/:id"},
			{"GET", "/repos/:owner/:repo/forks"},
			{"POST", "/repos/:owner/:repo/forks"},
			{"GET", "/repos/:owner/:repo/hooks"},
			{"GET", "/repos/:owner/:repo/hooks/:id"},
			{"POST", "/repos/:owner/:repo/hooks"},
			//{"PATCH", "/repos/:owner/:repo/hooks/:id"},
			{"POST", "/repos/:owner/:repo/hooks/:id/tests"},
			{"DELETE", "/repos/:owner/:repo/hooks/:id"},
			{"POST", "/repos/:owner/:repo/merges"},
			{"GET", "/repos/:owner/:repo/releases"},
			{"GET", "/repos/:owner/:repo/releases/:id"},
			{"POST", "/repos/:owner/:repo/releases"},
			//{"PATCH", "/repos/:owner/:repo/releases/:id"},
			{"DELETE", "/repos/:owner/:repo/releases/:id"},
			{"GET", "/repos/:owner/:repo/releases/:id/assets"},
			{"GET", "/repos/:owner/:repo/stats/contributors"},
			{"GET", "/repos/:owner/:repo/stats/commit_activity"},
			{"GET", "/repos/:owner/:repo/stats/code_frequency"},
			{"GET", "/repos/:owner/:repo/stats/participation"},
			{"GET", "/repos/:owner/:repo/stats/punch_card"},
			{"GET", "/repos/:owner/:repo/statuses/:ref"},
			{"POST", "/repos/:owner/:repo/statuses/:ref"},

			// Search
			{"GET", "/search/repositories"},
			{"GET", "/search/code"},
			{"GET", "/search/issues"},
			{"GET", "/search/users"},
			{"GET", "/legacy/issues/search/:owner/:repository/:state/:keyword"},
			{"GET", "/legacy/repos/search/:keyword"},
			{"GET", "/legacy/user/search/:keyword"},
			{"GET", "/legacy/user/email/:email"},

			// Users
			{"GET", "/users/:user"},
			{"GET", "/user"},
			//{"PATCH", "/user"},
			{"GET", "/users"},
			{"GET", "/user/emails"},
			{"POST", "/user/emails"},
			{"DELETE", "/user/emails"},
			{"GET", "/users/:user/followers"},
			{"GET", "/user/followers"},
			{"GET", "/users/:user/following"},
			{"GET", "/user/following"},
			{"GET", "/user/following/:user"},
			{"GET", "/users/:user/following/:target_user"},
			{"PUT", "/user/following/:user"},
			{"DELETE", "/user/following/:user"},
			{"GET", "/users/:user/keys"},
			{"GET", "/user/keys"},
			{"GET", "/user/keys/:id"},
			{"POST", "/user/keys"},
			//{"PATCH", "/user/keys/:id"},
			{"DELETE", "/user/keys/:id"},
		}
		port = getTempPort(t)
		xcfg = &XConfig{}
	)
	if setupServer(t, xcfg, pairs, port) {
		setupClient(t, xcfg, pairs, port)
	}
}

type requestPair struct {
	Method string
	Path   string
}

type responsePair struct {
	Ret int    `json:"ret"`
	Msg string `json:"msg"`
}

func setupServer(t *testing.T, xcfg *XConfig, pairs []requestPair, port string) (complete bool) {
	var (
		xr = New(xcfg)
	)
	for _, pair := range pairs {
		xassert.IsNil(t, xr.Handle(pair.Method, pair.Path, (func(pair requestPair) XHandle {
			return func(w http.ResponseWriter, r *http.Request, xps XParams) {
				var rsp = &responsePair{}

				defer func() {
					data, _ := json.Marshal(rsp)
					w.Write(data)
				}()

				// Check whether HTTP method is matching.
				if r.Method != pair.Method {
					rsp.Ret, rsp.Msg = -1, fmt.Sprintf("request method (%s) is not equal to register method (%s)", r.Method, pair.Method)
					return
				}

				var ok bool
				// Check wheter request path is matching.
				if ok = match(pair.Path, r.URL.Path); !ok {
					if xcfg.CompatibleWithTrailingSlash {
						if r.URL.Path[len(r.URL.Path)-1] == '/' {
							ok = match(pair.Path, r.URL.Path[:len(r.URL.Path)-1])
						} else {
							ok = match(pair.Path, r.URL.Path+"/")
						}
					}
				}

				if !ok {
					rsp.Ret, rsp.Msg = -1, fmt.Sprintf("request path (%s) doesn't match register path (%s)", r.URL.Path, pair.Path)
					return
				}

				if len(xps) > 0 {
					rsp.Msg = xps.String()
				} else {
					rsp.Msg = "success"
				}

				return
			}
		})(pair)))
	}

	s := &http.Server{
		Addr:           "127.0.0.1:" + port,
		Handler:        xr,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go s.ListenAndServe()
	time.Sleep(time.Second)
	complete = true
	return
}

func setupClient(t *testing.T, xcfg *XConfig, pairs []requestPair, port string) {
	var (
		client = &http.Client{}
	)

	for _, pair := range pairs {
		for _, method := range methods {
			var (
				rp    = &responsePair{}
				rpath = pattern2path(pair.Path)
			)
			r, err := http.NewRequest(method, "http://127.0.0.1:"+port+rpath, nil)
			xassert.IsNil(t, err)
			rsp, err := client.Do(r)
			xassert.IsNil(t, err)
			// 			fmt.Println(method, pair.Path, rpath)

			if pair.Method == method {
				xassert.Equal(t, rsp.StatusCode, 200)
				data, err := ioutil.ReadAll(rsp.Body)
				xassert.IsNil(t, err)
				xassert.IsNil(t, json.Unmarshal(data, rp))
				xassert.Equal(t, rp.Ret, 0)
			} else if method == "OPTIONS" && xcfg.HandleOptions {
				xassert.Equal(t, rsp.StatusCode, 200)
				xassert.IsTrue(t, rsp.Header.Get("Allow") != "")
			} else if xcfg.HandleMethodNotAllowed {
				xassert.Equal(t, rsp.StatusCode, 405)
				xassert.IsTrue(t, rsp.Header.Get("Allow") != "")
			}
		}
	}
}

// Returns a temporary avaliable local port for binding.
func getTempPort(t *testing.T) string {
	laddr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	l, err := net.ListenTCP("tcp", laddr)
	xassert.IsNil(t, err)
	defer l.Close()
	_, port, _ := net.SplitHostPort(l.Addr().String())
	return port
}

// Check whether the string matches the pattern, because this
// function is only used in test, so the pattern must be valid.
func match(pattern, str string) bool {
	var (
		pn, sn = len(pattern), len(str)
		pi, si int
	)

	if pn == 0 || sn == 0 {
		// No matter the pattern or the string is empty, return false.
		return false
	}

	for pi < pn && si < sn {
		switch pattern[pi] {
		case ':':
			if i := strings.IndexByte(pattern[pi:], '/'); i == -1 {
				pi = pn
			} else {
				pi += i
			}

			if i := strings.IndexByte(str[si:], '/'); i == -1 {
				si = sn
			} else {
				si += i
			}
		case '*':
			pi, si = pn, sn
		default:
			if pattern[pi] != str[si] {
				return false
			}
			pi++
			si++
		}
	}

	if pi != pn || si != sn {
		return false
	}

	return true
}

var (
	paramRe = regexp.MustCompile(`:.+?(\/|$)`)
	allRe   = regexp.MustCompile(`\*.+$`)
)

// Generate request path from the register pattern.
func pattern2path(pattern string) string {
	return allRe.ReplaceAllString(paramRe.ReplaceAllString(pattern, xrandstring.Get(8)+"$1"), xrandstring.Get(8))
}
