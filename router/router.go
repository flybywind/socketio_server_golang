package router

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"regexp"
	"strings"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type HandlerFunc func(*Context)
type RouterHandler struct {
	handlerFunc HandlerFunc
	methodName  string
	paramIndex  map[int]string
}

func (rh *RouterHandler) String() string {
	return fmt.Sprintf("method = %s,  paramIndex = %v", rh.methodName, rh.paramIndex)
}

type Router struct {
	routerMap map[*regexp.Regexp]*RouterHandler
}

var globalRouter Router = Router{
	routerMap: map[*regexp.Regexp]*RouterHandler{},
}
var routerPattern *regexp.Regexp = regexp.MustCompile(`:([\w\d_]+)+`)

type Context struct {
	http.ResponseWriter
	*http.Request
	Params map[string]string
}

func (c *Context) Method() string {
	return strings.ToLower(c.Request.Method)
}
func (c *Context) Path() string {
	return c.Request.URL.Path
}
func (c *Context) Write(p []byte) (n int, err error) {
	return c.ResponseWriter.Write(p)
}
func (c *Context) SetContentType(ct string) {
	c.ResponseWriter.Header().Set("Content-Type", ct)
}

func GetRouter() *Router {
	return &globalRouter
}

type Entry struct{}

func (e Entry) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	ctx := &Context{
		resp, req, map[string]string{},
	}
	rawPath := ctx.Path()
	method := ctx.Method()
	if err := globalRouter.Dispatch(ctx, rawPath, method); err != nil {
		//panic(fmt.Errorf("dispatch handler failed:", err))
		log.Println(err)
	}
}

func (r *Router) Dispatch(ctx *Context, rawPath, method string) error {
	failed := true
	for reg, handler := range r.routerMap {
		if handler.methodName == "" || handler.methodName == method {
			m := reg.FindAllStringSubmatch(rawPath, -1)
			if m != nil {
				failed = false
				submatch_len := len(m[0])
				for i := 1; i < submatch_len; i++ {
					ctx.Params[handler.paramIndex[i-1]] = m[0][i]
				}
				handler.handlerFunc(ctx)
				return nil
			}
		}
	}

	if failed {
		log.Println("path =", rawPath, "method =", method) //, "\nrouter map:", r.routerMap)
		return fmt.Errorf("no router found!")
	}
	return nil
}

func (r *Router) AddRouter(pattern string, handler HandlerFunc) *RouterHandler {
	log.Println("add router:", pattern)
	submatch := routerPattern.FindAllStringSubmatch(pattern, -1)
	var newPatternStr string
	if submatch == nil {
		newPatternStr = "^" + pattern + "$"
	} else {
		newPatternStr = "^" + routerPattern.ReplaceAllString(pattern, `([\w\d_]+)`) + `/?`
	}
	newPattern := regexp.MustCompile(newPatternStr)
	rh := &RouterHandler{
		handlerFunc: handler,
		paramIndex:  map[int]string{},
	}

	r.routerMap[newPattern] = rh
	for index, match := range submatch {
		rh.paramIndex[index] = match[1]
	}
	return rh
}
func (r *Router) AddRouterMethod(pattern string, method string, handler HandlerFunc) *RouterHandler {
	rh := r.AddRouter(pattern, handler)
	rh.methodName = strings.ToLower(method)
	return rh
}

func (r *Router) AddStatic(src string) {
	src_seg := strings.Split(src, "/")
	src_len := len(src_seg)
	dest := "/" + src_seg[src_len-1] + "/:subpath"
	r.AddRouterMethod(dest, "get", func(ctx *Context) {
		// matched:
		real_path := ctx.Path()
		ext_name := strings.ToLower(path.Ext(real_path))
		switch ext_name {
		case ".js":
			ctx.SetContentType("application/x-javascript; charset=utf-8")
		case ".css":
			ctx.SetContentType("text/css; charset=utf-8")
		default:
			log.Printf("extention %s not recognized, use text/plain\n", ext_name)
		}

		seg := strings.Split(real_path, "/")
		src_path := src + "/" + strings.Join(seg[2:], "/")
		file_bytes, err := ioutil.ReadFile(src_path)
		if err != nil {
			log.Fatalf("read file %s error: %v\n", src_path, err)
		}
		_, err = ctx.Write(file_bytes)

		if err != nil {
			log.Fatalf("send file %s error: %v\n", dest, err)
		}
	})
}
