package handler

import (
	"html/template"
	"log"
	"socketio_server/chatsocket"
	"socketio_server/router"
	"strings"
)

const (
	viewsPath = "views"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func setHtml(ctx *router.Context) {
	ctx.SetContentType("text/html; charset=utf-8")
}
func Index(ctx *router.Context) {
	setHtml(ctx)
	if tmpl, err := template.ParseFiles(viewsPath + "/index.html"); err != nil {
		log.Fatalf("pase index failed:", err)
	} else {
		if err = tmpl.Execute(ctx, struct {
			Title string
		}{"hello socket golang"}); err != nil {
			log.Fatalf("execut html failed:", err)
		}
	}
}
func Rooms(ctx *router.Context) {
	var (
		err  error
		tmpl *template.Template
	)
	setHtml(ctx)
	if tmpl, err = template.ParseFiles(viewsPath + "/room.html"); err != nil {
		log.Fatalf("pase index failed:", err)
	} else {
		if err = tmpl.Execute(ctx, struct {
			Title, User, Room string
		}{"hello socket golang",
			ctx.Params["user"],
			ctx.Params["room"],
		}); err != nil {
			log.Fatalf("execut html failed:", err)
		}

	}

}

func Chat(ctx *router.Context) {
	var (
		err  error
		chat *chatsocket.ChatConn
	)

	stop := make(chan bool, 1)
	chat, err = chatsocket.NewChatConn(ctx.ResponseWriter, ctx.Request)
	log.Println("enter chat...")
	if err != nil {
		log.Println(err)
	} else {
		chat.On("JoinRoom", func(chat *chatsocket.ChatConn, msg *chatsocket.Msg) {
			msg.Type = chatsocket.NoticeMsg
			msg.EventName = "enter_room"
			seg := strings.Split(msg.Content, ":")
			chat.SetId(chatsocket.ChatId(seg[1]))
			if err := chat.JoinRoom(chatsocket.RoomId(seg[0]), msg.Token); err == nil {
				msg.Content = seg[1]
				err = chat.BroadCast(*msg)
				if err == nil {
					log.Println("user:", chat.Id(), "broadcastting")
				} else {
					log.Println("user:", chat.Id(), "broadcast failed:", err)
				}
				log.Println("user:", chat.Id(), "join room:", seg[0], "message:\n", msg)
			} else {
				log.Println("user:", chat.Id(), "join room:", seg[0], "failed:", err)
			}
		})
		chat.On("Close", func(chat *chatsocket.ChatConn, msg *chatsocket.Msg) {
			chat.Close()
			stop <- true
			log.Println("close chat:", chat.Id())
		})
		for {
			select {
			case <-stop:
				return
			}
		}
	}
	log.Println("leave chat...")
}
