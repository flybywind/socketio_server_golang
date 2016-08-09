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
	r.AddRouterMethod("/ws/", "get", handler.Chat)
	r.AddStatic("static/javascripts")
	r.AddStatic("static/stylesheets")
	server := http.Server{
		Addr:    listenAddr,
		Handler: router.Entry{},
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
