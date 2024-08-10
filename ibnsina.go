package ibnsina

import (
	"context"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

var (
	defaultNotFound = func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		response.Write([]byte("the requested resource could not be found\n"))
	}

	defaultMethodNotAllowed = func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		response.Write([]byte("the method " + request.Method + " is not supported for the requested resource\n"))
	}

	defaultOptions = func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	}

	allMethods = []string{
		http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace,
	}

	rxPatterns = map[string]*regexp.Regexp{}
)

type contextKey string

func Param(ctx context.Context, name string) string {
	value, ok := ctx.Value(contextKey(name)).(string)
	if !ok {
		return ""
	}

	return value
}

type Handler func(context.Context, http.ResponseWriter, *http.Request)

type Router struct {
	NotFound         Handler
	MethodNotAllowed Handler
	Options          Handler
	routes           []*route
	middlewares      []func(Handler) Handler
}

func NewRouter() *Router {
	return &Router{
		NotFound:         defaultNotFound,
		MethodNotAllowed: defaultMethodNotAllowed,
		Options:          defaultOptions,
		routes:           []*route{},
	}
}

func (router *Router) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	segments := strings.Split(request.URL.EscapedPath(), "/")
	methods := []string{}

	for index := 0; index < len(router.routes); index++ {
		ctx, ok := router.routes[index].match(request.Context(), segments)
		if ok {
			if request.Method == router.routes[index].method {
				router.routes[index].handler(context.Background(), response, request.WithContext(ctx))
				return
			}

			if !slices.Contains(methods, router.routes[index].method) {
				methods = append(methods, router.routes[index].method)
			}
		}

		if len(methods) > 0 {
			response.Header().Set("Allow", strings.Join(append(methods, http.MethodOptions), ", "))

			if request.Method == http.MethodOptions {
				router.wrap(router.Options)(context.Background(), response, request)
			} else {
				router.wrap(router.MethodNotAllowed)(context.Background(), response, request)
			}

			return
		}
	}

	router.wrap(router.NotFound)(context.Background(), response, request)
}

type route struct {
	method   string
	segments []string
	wildcard bool
	handler  Handler
}

func (router *Router) Handle(path string, handler Handler, methods ...string) {
	if slices.Contains(methods, http.MethodGet) && !slices.Contains(methods, http.MethodHead) {
		methods = append(methods, http.MethodHead)
	}

	if len(methods) == 0 {
		methods = allMethods
	}

	segments := strings.Split(path, "/")

	for index := 0; index < len(methods); index++ {
		route := &route{
			method:   strings.ToUpper(methods[index]),
			segments: segments,
			wildcard: strings.HasSuffix(path, "/..."),
			handler:  handler,
		}

		router.routes = append(router.routes, route)
	}

	for index := 0; index < len(segments); index++ {
		if strings.HasPrefix(segments[index], ":") {
			if _, rx, contains := strings.Cut(segments[index], "|"); contains {
				rxPatterns[rx] = regexp.MustCompile(rx)
			}
		}
	}
}

func (route *route) match(ctx context.Context, segments []string) (context.Context, bool) {
	if !route.wildcard && len(route.segments) != len(segments) {
		return ctx, false
	}

	for index := 0; index < len(route.segments); index++ {
		if index > len(segments)-1 {
			return ctx, false
		}

		if route.segments[index] == "..." {
			ctx = context.WithValue(ctx, contextKey("..."), strings.Join(segments[index:], "/"))
			return ctx, true
		}

		if strings.HasPrefix(route.segments[index], ":") {
			key, rx, contains := strings.Cut(strings.TrimPrefix(route.segments[index], ":"), "|")

			if contains {
				if rxPatterns[rx].MatchString(segments[index]) {
					ctx = context.WithValue(ctx, contextKey(key), segments[index])
					continue
				}
			} else if segments[index] != "" {
				ctx = context.WithValue(ctx, contextKey(key), segments[index])
				continue
			}

			return ctx, false
		}

		if segments[index] != route.segments[index] {
			return ctx, false
		}
	}

	return ctx, true
}

func (router *Router) wrap(handler Handler) Handler {
	for index := len(router.middlewares) - 1; index > -1; index-- {
		handler = router.middlewares[index](handler)
	}

	return handler
}
