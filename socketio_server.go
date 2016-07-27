package main

import (
	"flag"
	//"github.com/googollee/go-socket.io"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"socketio_server/handler"
	"socketio_server/router"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

var Prefork bool
var child bool
var listenAddr string

func main() {
	flag.BoolVar(&Prefork, "prefork", false, "whether use prefork mode")
	flag.BoolVar(&child, "child", false, "if child process")
	flag.StringVar(&listenAddr, "port", ":8080", "listen port")
	flag.Parse()
	if !flag.Parsed() {
		flag.Usage()
		log.Fatalln()
	}
	r := router.GetRouter()
	r.AddRouter("/", handler.Index)
	r.AddRouter("/chat_room/:room.:user", handler.Rooms)
	r.AddStatic("static/javascript")
	r.AddStatic("static/stylesheets")
	fasthttpServer := fasthttp.Server{
		Handler: router.Entry,
		Name:    "test-socketio",
	}
	ln := GetListener()

	if err := fasthttpServer.Serve(ln); err != nil {
		log.Fatalln("start listenning failed:", err)
	}
}

func GetListener() net.Listener {
	if !Prefork {
		runtime.GOMAXPROCS(runtime.NumCPU())
		ln, err := net.Listen("tcp4", listenAddr)
		if err != nil {
			log.Fatal(err)
		}
		return ln
	}

	if !child {
		children := make([]*exec.Cmd, runtime.NumCPU())
		for i := range children {
			children[i] = exec.Command(os.Args[0], "-prefork", "-child", "-listenAddr", listenAddr)
			children[i].Stdout = os.Stdout
			children[i].Stderr = os.Stderr
			if err := children[i].Start(); err != nil {
				log.Fatal(err)
			}
		}
		for _, ch := range children {
			if err := ch.Wait(); err != nil {
				log.Print(err)
			}
		}
		log.Println("exit main")
		os.Exit(0)
		panic("unreachable")
	}

	runtime.GOMAXPROCS(1)
	ln, err := reuseport.Listen("tcp4", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	return ln
}
