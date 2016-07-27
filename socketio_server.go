package main

import (
	"flag"
	"github.com/googollee/go-socket.io"
	"log"
	"net/http"
	"socketio_server/handler"
	"socketio_server/router"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

var listenAddr string

func main() {
	flag.StringVar(&listenAddr, "port", ":8080", "listen port")
	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		log.Fatalln()
	}
	r := router.GetRouter()
	r.AddRouterMethod("/", "get", handler.Index)
	r.AddRouterMethod("/chat_room/:room.:user", "get", handler.Rooms)
	r.AddStatic("static/javascripts")
	r.AddStatic("static/stylesheets")
	//r.AddStatic("static/javascripts/socket.io")
	server := http.Server{
		Addr:    listenAddr,
		Handler: router.Entry{},
	}
	socket := AddSocketIO(r, nil)
	ns := socket.Of("/chat_room")
	ns.On("connection", func(so socketio.Socket) {
		so.On("join_room", func(data string) {
			log.Println("receive join room request:", data)
			so.Emit("user_entered", data)
		})
		so.On("message", func(message string) {
			log.Println("receive message:", message)
		})
	})

	if err := server.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}

func AddSocketIO(r *router.Router, transportName []string) *socketio.Server {
	server, err := socketio.NewServer(transportName)
	if err != nil {
		log.Println("start socketio failed:", err)
		return nil
	}

	r.AddRouter("/socket.io/", func(ctx *router.Context) {
		server.ServeHTTP(ctx.ResponseWriter, ctx.Request)
	})
	return server
}
