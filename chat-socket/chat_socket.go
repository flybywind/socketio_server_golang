package chatsocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"strings"
	"time"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type Msg struct {
	Content,
	RoomName,
	Token,
	Type string
}

const (
	OwnMsg              = "0"
	NormalMsg           = "1"
	EventMsg            = "9"
	RoomNotExists       = func(room string) error { return fmt.Errorf("room [%s] not exists", room) }
	JoinRoomWithoutAuth = func(room string) error { return fmt.Errorf("invalid to join room: %s", room) }
)

var (
	WriteWaitSec = 10 * time.Second
	rooms        = make(map[string]*RoomSocket)
)

type RoomSocket struct {
	name, token string
	members     []*ChatConn
}
type ChatConn struct {
	con        *websocket.Conn
	name       string
	belongRoom string
	ownRoom    bool
}

func createRoom(name, token string) (*RoomSocket, error) {
	room, exists := rooms[name]
	if exists {
		return nil, fmt.Errorf("room [%s] exists", name)
	}
	room = &RoomSocket{
		name, make([]*ChatConn), token,
	}

	rooms[name] = room
	return room, nil
}
func findRoom(name string) *RoomSocket {
	room, exists := rooms[name]
	if exists {
		return room
	} else {
		return nil
	}
}

func (r *RoomSocket) BroadCast(fromUser, string, msg Msg) error {

	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(msg); err == nil {
		for _, conn := range r.members {
			if conn.Name() == fromUser {
				msg.Type = OwnMsg
				buf2 := bytes.NewBuffer(make([]byte, buf.Len()))
				err = json.NewEncoder(buf2).Encode(msg)
				if err != nil {
					return err
				}
				return conn.con.WriteMessage(websocket.TextMessage, buf2.Bytes())
			} else {
				return conn.con.WriteMessage(websocket.TextMessage, buf.Bytes())
			}
		}
	} else {
		return err
	}
}

func (r *RoomSocket) AddConn(c *ChatConn) {
	r.members = append(r.members, c)
}

func (c *ChatConn) Name() string {
	return c.name
}
func (c *ChatConn) BroadCast(msg Msg) error {
	roomSocket := findRoom(belongRoom)

	if roomSocket == nil {
		return RoomNotExists(belongRoom)
	}

	roomSocket.BroadCast(c.name, msg)
}
func (c *ChatConn) Active() {
	c.con.SetReadDeadline(time.Time{})
}
func (c *ChatConn) joinRoom(room string, token string) error {
	roomSocket := findRoom(room)

	if roomSocket != nil {
		if roomSocket.token == token {
			roomSocket.AddConn(c)
			c.belongRoom = room
		} else {
			// TODO: need owner allowed
			return JoinRoomWithoutAuth(room)
		}
		return nil
	} else {
		if roomSocket, err := createRoom(room, token); err == nil {
			roomSocket.AddConn(c)
			c.belongRoom = room
			c.ownRoom = true
		} else {
			return err
		}
	}
}

func (c *ChatConn) waitMessage() {
	for {
		_, msg_bytes, err := c.con.ReadMessage()
		if err == nil {
			buf := bytes.NewBuffer(msg_bytes)
			msg := Msg{}
			json.NewDecoder(buf).Decode(&msg)
			switch msg.Type {
			case NormalMsg:
				c.BroadCast(msg)
			case EventMsg:
				cmd := strings.ToLower(msg.Content)
				switch cmd {
				case "join":
					err = c.joinRoom(msg.RoomName, msg.Token)
				case "leave":
					c.con.Close()
				default:
					err = fmt.Errorf("not recognizied action")
				}
				if err == nil {
					msg.Content = "0"
				} else {
					msg.Content = err.Error()
				}
				c.Emit(msg)
			}
		}
	}
}

func (c *ChatConn) Emit(msg Msg) {
	if msg.Type == EventMsg {
		c.BroadCast(msg)
	} else {
		panic("should emit event actions")
	}
}
