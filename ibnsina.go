package ibnsina

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/pborman/uuid"
)

const TraceIDHeader = "X-Trace-ID"

var (
	defaultNotFound = func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusNotFound)
		response.Write([]byte("the requested resource could not be found\n"))
	}

	defaultMethodNotAllowed = func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusMethodNotAllowed)
		response.Write([]byte("the method " + request.Method + " is not supported for the requested resource\n"))
	}

	defaultOptions = func(ctx context.Context, response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	}

	allMethods = []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace}

	rxPatterns = map[string]*regexp.Regexp{}
)

type ctxKey string

type Values struct {
	TraceID string
	Now     time.Time
	Status  int
}

type contextKey int

func (router *Router) Run(addr string, timeout time.Duration, logger *log.Logger) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		IdleTimeout:  timeout,
		ErrorLog:     logger,
	}

	errs := make(chan error, 1)

	go func() {
		errs <- srv.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	select {
	case err := <-errs:
		return err
	case <-signals:
		timeout := 5 * time.Second

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			// kill 9: kill hard
			if err := srv.Close(); err != nil {
				return err
			}
		}

		return <-errs
	}
}

func Param(ctx context.Context, name string) string {
	value, ok := ctx.Value(ctxKey(name)).(string)
	if !ok {
		return ""
	}

	return value
}

type Handler func(context.Context, http.ResponseWriter, *http.Request)

type Middleware func(Handler) Handler

type Router struct {
	NotFound         Handler
	MethodNotAllowed Handler
	Options          Handler
	routes           []*route
	middlewares      []Middleware
}

func NewRouter(middlewares ...Middleware) *Router {
	return &Router{
		NotFound:         defaultNotFound,
		MethodNotAllowed: defaultMethodNotAllowed,
		Options:          defaultOptions,
		routes:           []*route{},
		middlewares:      middlewares,
	}
}

func (router *Router) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	segments := strings.Split(request.URL.EscapedPath(), "/")
	methods := []string{}

	values := Values{
		TraceID: uuid.New(),
		Now:     time.Now(),
	}

	response.Header().Set(TraceIDHeader, values.TraceID)

	ctx := context.WithValue(request.Context(), contextKey(1), values)

	for index := 0; index < len(router.routes); index++ {
		c, ok := router.routes[index].match(request.Context(), segments)
		if ok {
			if request.Method == router.routes[index].method {
				router.routes[index].handler(ctx, response, request.WithContext(c))
				return
			}

			if !slices.Contains(methods, router.routes[index].method) {
				methods = append(methods, router.routes[index].method)
			}
		}
	}

	if len(methods) > 0 {
		response.Header().Set("Allow", strings.Join(append(methods, http.MethodOptions), ", "))

		if request.Method == http.MethodOptions {
			router.wrap(router.Options)(ctx, response, request)
		} else {
			router.wrap(router.MethodNotAllowed)(ctx, response, request)
		}

		return
	}

	router.wrap(router.NotFound)(ctx, response, request)
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
			handler:  router.wrap(handler),
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

func (router *Router) Use(middlewares ...Middleware) {
	router.middlewares = append(router.middlewares, middlewares...)
}

type Group struct {
	router      *Router
	middlewares []Middleware
}

func (router *Router) Group(middlewares ...Middleware) *Group {
	return &Group{
		router:      router,
		middlewares: middlewares,
	}
}

func (group *Group) Handle(path string, handler Handler, methods ...string) {
	group.router.Handle(path, group.router.wrap(handler), methods...)
}

func (route *route) match(ctx context.Context, segments []string) (context.Context, bool) {
	if !route.wildcard && len(segments) != len(route.segments) {
		return ctx, false
	}

	for index, rs := range route.segments {
		if index > len(segments)-1 {
			return ctx, false
		}

		if rs == "..." {
			ctx = context.WithValue(ctx, ctxKey("..."), strings.Join(segments[index:], "/"))
			return ctx, true
		}

		if strings.HasPrefix(rs, ":") {
			key, rx, contains := strings.Cut(strings.TrimPrefix(rs, ":"), "|")
			if contains {
				if rxPatterns[rx].MatchString(segments[index]) {
					ctx = context.WithValue(ctx, ctxKey(key), segments[index])
					continue
				}
			}

			if !contains && segments[index] != "" {
				ctx = context.WithValue(ctx, ctxKey(key), segments[index])
				continue
			}

			return ctx, false
		}

		if rs != segments[index] {
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
