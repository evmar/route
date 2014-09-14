// Package route is a simple URL router.
//
// Unlike other routers, there are no regexes; paths are interpreted
// as sequences of slash-separated components.
//
// This allows a uniform representation of routers as a tree of path
// matchers. The following two expressions are equivalent, both
// mapping to the matcher for the path "/user/new":
//
//     r := &Router{}
//     r.Route("user").Route("new")
//     r.Route("user/new")
//
// You then attach a handler to a Router to handle that specific path.
// (Attaching a handler to the zero router handles "/".)
//
// Constructing intermediate handlers allows structured construction
// of match trees, as in the following:
//
//     userRouter := r.Route("user")
//     userRouter.Route("new").Func(newUserHandler)
//     userRouter.Route("edit").Func(editUserHandler)
//
// Router additionally supports capturing components within the path
// and path wildcards.  See the Route function for details.
package route

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

type handler func(w http.ResponseWriter, r *http.Request, env map[string]string)

// Router represents a single node in the matching tree.
type Router struct {
	// matchers contains the subentries under this path.
	matchers map[string]*Router

	// If this router has a child glob matcher like "{entryId}", then
	// varName holds the name of the variable and varRouter is the
	// router to handle it.
	varName   string
	varRouter *Router

	// handler is the handler for matches to this exact node.
	handler handler

	// fallback is the handler for falling back to if none of the above
	// match; conceptually it's the "*" handler.
	fallbackRouter *Router
}

func (r *Router) lookup(path []string, env map[string]string) handler {
	// Empty path => we've matched on this router exactly.
	if len(path) == 0 {
		if r.handler != nil {
			return r.handler
		}
		// TODO: maybe we should rely on fallback here too?
		// E.g. with fallback on "/foo", is "/foo" itself a match?
		return nil
	}

	if r.matchers != nil {
		if r2 := r.matchers[path[0]]; r2 != nil {
			if h := r2.lookup(path[1:], env); h != nil {
				return h
			}
		}
	}
	if path[0] != "" && r.varRouter != nil {
		env[r.varName] = path[0]
		if h := r.varRouter.lookup(path[1:], env); h != nil {
			return h
		}
	}
	if r.fallbackRouter != nil {
		env["*"] = strings.Join(path, "/")
		return r.fallbackRouter.handler
	}
	return nil
}

// lookupPath computes the handler matching a given request path string.
// It just forwards to lookup.
func (r *Router) lookupPath(path string, env map[string]string) handler {
	if path[0] != '/' {
		panic("bad path")
	}
	parts := strings.Split(path[1:], "/")
	return r.lookup(parts, env)
}

// ServeHTTP is the adapter for use in http.ListenAndServe.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	env := map[string]string{}
	if h := r.lookupPath(req.URL.Path, env); h != nil {
		h(w, req, env)
		return
	}
	http.NotFound(w, req)
}

func (r *Router) route(parts []string) *Router {
	if len(parts) == 0 {
		return r
	}

	part := parts[0]
	if len(part) > 0 && part[0] == ':' {
		part = part[1:]
		if r.varName != "" && part != r.varName {
			log.Panicf("overlapping vars: %q / %q", r.varName, part)
		}
		if r.varRouter == nil {
			r.varName = part
			r.varRouter = &Router{}
		}
		r = r.varRouter
	} else if part == "*" {
		if r.fallbackRouter != nil {
			log.Panicf("overlapping fallback routes")
		}
		r.fallbackRouter = &Router{}
		return r.fallbackRouter
	} else {
		if r.matchers == nil {
			r.matchers = make(map[string]*Router)
		}
		if r.matchers[part] == nil {
			r.matchers[part] = &Router{}
		}
		r = r.matchers[part]
	}
	return r.route(parts[1:])
}

// Route gets the router for a subpath off the current router.
//
// There are two special path components:
//
// 1) components starting with ":", e.g. "/foo/:id/bar", match any
// string and capture the value in the environment (see the example);
//
// 2) the "*" component matches all paths, leaving it up to the
// handler to further parse the path.  The matched subpath is also
// captured in the environment (see the example).
func (r *Router) Route(path string) *Router {
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	parts := strings.Split(path, "/")
	return r.route(parts)
}

// FuncE registers an "extended" handler, which takes an additional
// environment parameter, at the current point.
func (r *Router) FuncE(f func(w http.ResponseWriter, r *http.Request, env map[string]string)) {
	if r.handler != nil {
		panic("duplicate handler")
	}
	r.handler = f
}

// Func registers an http.HandlerFunc at the current point.
func (r *Router) Func(f func(http.ResponseWriter, *http.Request)) {
	r.FuncE(func(w http.ResponseWriter, r *http.Request, env map[string]string) {
		f(w, r)
	})
}

// Dump dumps the routing table to stdout.
// It can be useful for debugging.
func (r *Router) Dump(prefix string) {
	if r.handler != nil {
		fmt.Printf("%s=> %v\n", prefix, r.handler)
	}

	if r.matchers != nil {
		for k, v := range r.matchers {
			fmt.Printf("%s%s/\n", prefix, k)
			v.Dump(prefix + "  ")
		}
	}

	if r.varName != "" {
		fmt.Printf("%s:%s\n", prefix, r.varName)
		r.varRouter.Dump(prefix + "  ")
	}

	if r.fallbackRouter != nil {
		fmt.Printf("%s*\n", prefix)
		r.fallbackRouter.Dump(prefix + "  ")
	}
}
