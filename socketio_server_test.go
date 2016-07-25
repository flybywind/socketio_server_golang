package main

import (
	"log"
	"socketio_server/router"
	"testing"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func TestRouter(t *testing.T) {
	r := router.GetRouter()
	ctx := &router.Context{
		nil,
		map[string]string{},
	}
	rh := r.AddRouter("/:name-:tel", func(ctx *router.Context) {
		if ctx.Params["name"] != "tom" || ctx.Params["tel"] != "123" {
			t.Errorf("add router: tom-123 failed, %s\n", ctx.Params)
		} else {
			log.Println("sucess:", ctx.Params)
		}
	})
	log.Printf("add /:name-:tel | %v", rh)
	rh = r.AddRouterMethod("/:name/:addr", "Post", func(ctx *router.Context) {
		if ctx.Params["name"] != "Jame" || ctx.Params["addr"] != "yufei" {
			t.Fail()
		} else {
			log.Println("sucess:", ctx.Params)
		}
	})
	log.Printf("add /:name/:addr | %v", rh)

	err := r.Dispatch(ctx, "/tom-123", "")
	if err != nil {
		t.Errorf("Dispatch tom-123 failed: %v\n", err)
	}
	r.Dispatch(ctx, "/Jame/yufei", "post")
	if err != nil {
		t.Errorf("Dispatch Jame/yufei failed: %v\n", err)
	}
}
