package route

import (
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func F1(w http.ResponseWriter, r *http.Request, env map[string]string) {
}

func TestEmpty(t *testing.T) {
	r := &Router{}
	assert.Nil(t, r.lookupPath("/", nil))
	assert.Nil(t, r.lookupPath("/foo", nil))
}

func TestBasic(t *testing.T) {
	r := &Router{}
	r.Route("/").FuncE(F1)
	assert.NotNil(t, r.lookupPath("/", nil))
	assert.Nil(t, r.lookupPath("/foo", nil))
}

func TestOne(t *testing.T) {
	r := &Router{}
	r.Route("/foo").FuncE(F1)
	assert.Nil(t, r.lookupPath("/", nil))
	assert.NotNil(t, r.lookupPath("/foo", nil))
}

func TestTwo(t *testing.T) {
	r := &Router{}
	r.Route("/").FuncE(F1)
	r.Route("/foo").FuncE(F1)
	assert.NotNil(t, r.lookupPath("/", nil))
	assert.NotNil(t, r.lookupPath("/foo", nil))
}

func TestDir(t *testing.T) {
	r := &Router{}
	r.Route("/foo/").FuncE(F1)
	assert.Nil(t, r.lookupPath("/", nil))
	assert.Nil(t, r.lookupPath("/foo", nil))
	assert.NotNil(t, r.lookupPath("/foo/", nil))
	assert.Nil(t, r.lookupPath("/foo/bar", nil))
}

func TestDirTwoEntries(t *testing.T) {
	r := &Router{}
	r.Route("/foo/").FuncE(F1)
	r.Route("/foo/bar").FuncE(F1)
	assert.Nil(t, r.lookupPath("/", nil))
	assert.Nil(t, r.lookupPath("/foo", nil))
	assert.NotNil(t, r.lookupPath("/foo/", nil))
	assert.NotNil(t, r.lookupPath("/foo/bar", nil))
	assert.Nil(t, r.lookupPath("/foo/baz", nil))
}

func TestVar(t *testing.T) {
	r := &Router{}
	r.Route("/foo/:id").FuncE(F1)

	env := map[string]string{}
	assert.Nil(t, r.lookupPath("/", env))
	assert.Nil(t, r.lookupPath("/foo/", env))
	assert.Equal(t, 0, len(env))

	env = map[string]string{}
	assert.NotNil(t, r.lookupPath("/foo/bar", env))
	assert.Equal(t, 1, len(env))
	assert.Equal(t, "bar", env["id"])

	env = map[string]string{}
	r.Route("/foo/:id/edit").FuncE(F1)
	assert.NotNil(t, r.lookupPath("/foo/bar", env))
	assert.Nil(t, r.lookupPath("/foo/bar/xyz", env))
	assert.NotNil(t, r.lookupPath("/foo/bar/edit", env))
}

func TestFallback(t *testing.T) {
	r := &Router{}
	r.Route("/foo/*").FuncE(F1)

	assert.Nil(t, r.lookupPath("/", nil))
	assert.Nil(t, r.lookupPath("/foo", nil))

	env := map[string]string{}
	assert.NotNil(t, r.lookupPath("/foo/", env))
	assert.Equal(t, env["*"], "")

	env = map[string]string{}
	assert.NotNil(t, r.lookupPath("/foo/bar", env))
	assert.Equal(t, env["*"], "bar")
}

func ExampleRouter_basic() {
	myHandler := func(w http.ResponseWriter, r *http.Request) {
		// (A handler as in the http library.)
	}

	r := &Router{}

	// Any call to .Route() returns the Router for that path.
	// Then attach a handler for it via .Func().
	r.Route("/hello").Func(myHandler)

	// The trailing slash in a path matters.  This matches /foo only:
	r.Route("/foo").Func(myHandler)
	// This matches /foo/ only:
	r.Route("/foo/").Func(myHandler)
	// One exception: the root path, "/", is equivalent to the empty string.

	// These are equivalent:
	r.Route("/foo").Route("/bar").Func(myHandler)
	r.Route("/foo/bar").Func(myHandler)

	http.ListenAndServe(":8080", r)
}

// Variables, marked with a colon in routes, allow wildcards on paths.
// The handler function takes an extra argument: a map of variables to
// values.
//
// Use .FuncE to register a handler-with-environment function,
// that has the extra "env" argument.
func ExampleRouter_variables() {
	r := &Router{}
	myHandlerWithEnv := func(w http.ResponseWriter, r *http.Request, env map[string]string) {
		log.Println("username is", env["username"])
	}
	u := r.Route("/users/:username")
	u.Route("greet").FuncE(myHandlerWithEnv)

	// myHandlerWithEnv will match paths like "/users/foobar/greet",
	// and env["username"] in that case will be "foobar".
}

// Fallbacks, marked with "*" in routes, allow full path
// wildcards, for use in cases like serving a whole tree of files.
// The matched subpath is available in env["*"].  (The full path
// is always available in r.URL.Path, as this package never
// modifies the HTTP request or response).
func ExampleRouter_fallbacks() {
	r := &Router{}
	staticHandler := func(w http.ResponseWriter, r *http.Request, env map[string]string) {
		log.Println("subdir is", env["*"], "full path is", r.URL.Path)
	}
	r.Route("/static/*").FuncE(staticHandler)

	// Paths like "/static/foo/bar" will match staticHandler;
	// env["*"] will be "foo/bar".
}
