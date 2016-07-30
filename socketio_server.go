package main

import (
	"flag"
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

	socket.On("connection", func(so socketio.Socket) {
		log.Println("server connection from client!") // 收到！
		ns := socket.Of("/chat_room")
		so.On("join_room", func(data string) {
			log.Println("server join_room!") // 没有收到
			so.Emit("user_entered", data)
		})
		so.On("message", func(message string) {
			log.Println("server receive message:", message)
		})
		ns.On("join_room", func(data string) {
			log.Println("ns receive join room request:", data)
			so.Emit("user_entered", data)
		})
		ns.On("message", func(message string) {
			log.Println("ns receive message:", message)
		})

	})

	/*ns.On("connection", func(so socketio.Socket) {
		log.Println("namespace connection from client!") // 没有收到
		so.On("join_room", func(data string) {
			log.Println("receive join room request:", data)
			so.Emit("user_entered", data)
		})
		so.On("message", func(message string) {
			log.Println("receive message:", message)
		})
	})*/

	if err := server.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
