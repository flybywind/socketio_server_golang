package main

import (
	"flag"
	"github.com/googollee/go-socket.io"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"log"
	"net"
	"net/http"
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
	r.AddRouterMethod("/", "get", handler.Index)
	r.AddRouterMethod("/chat_room/:room.:user", "get", handler.Rooms)
	r.AddStatic("static/javascripts")
	r.AddStatic("static/stylesheets")
	r.AddStatic("static/javascripts/socket.io")
	fasthttpServer := fasthttp.Server{
		Handler: router.Entry,
		Name:    "test-socketio",
	}
	ln := GetListener()

	if err := fasthttpServer.Serve(ln); err != nil {
		log.Fatalln("start listenning failed:", err)
	}
	socket := GetSocketIO(nil)
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

func AddSocketIO(r *router.Router, transportName []string) *socketio.Server {
	server, err := socketio.NewServer(transportName)
	if err != nil {
		log.Println("start socketio failed:", err)
		return nil
	}

	r.AddRouter("/socket.io/", func(ctx *router.Context) {
	})
	return server
}
