package chatsocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type Msg struct {
	Content,
	Token,
	Type,
	EventName string
}

const (
	OwnMsg    = "0"
	NormalMsg = "1"
	NoticeMsg = "2"
	EventMsg  = "9"
)

type ChatId string
type RoomId string

var (
	WriteWaitSec        = 10 * time.Second
	rooms               = make(map[RoomId]*roomSocket)
	RoomNotExists       = func(room RoomId) error { return fmt.Errorf("room [%s] not exists", room) }
	JoinRoomWithoutAuth = func(room RoomId) error { return fmt.Errorf("invalid to join room: %s", room) }
)

type roomSocket struct {
	token   string
	name    RoomId
	members map[ChatId]*ChatConn
	msgbox  map[ChatId]time.Time
}
type ChatConn struct {
	con            *websocket.Conn
	name           ChatId
	belongRoom     RoomId
	ownRoom        bool
	leaveRoom      bool
	event_handlers map[string]func(*ChatConn, *Msg)
}

func createRoom(name RoomId, token string) (*roomSocket, error) {
	room, exists := rooms[name]
	if exists {
		return nil, fmt.Errorf("room [%s] exists", name)
	}
	room = &roomSocket{
		token, name,
		map[ChatId]*ChatConn{},
		map[ChatId]time.Time{},
	}

	rooms[name] = room
	return room, nil
}
func findRoom(name RoomId) *roomSocket {
	room, exists := rooms[name]
	if exists {
		return room
	} else {
		return nil
	}
}

func (r *roomSocket) broadCast(fromChat ChatId, msg Msg) error {
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(msg); err == nil {
		for _, conn := range r.members {
			if !conn.leaveRoom {
				if conn.Name() == fromChat {
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
			} else {
				now := time.Now()
				if _, exists := r.msgbox[conn.name]; !exists {
					r.msgbox[conn.name] = now
				}
				// TODO: get db to store messages
			}
		}
	} else {
		return err
	}
	return nil
}

func (r *roomSocket) addConn(c *ChatConn) {
	if conn, existsbefore := r.members[c.name]; existsbefore {
		c.ownRoom = conn.ownRoom
		// TODO: retreive older messages in msgbox
		delete(r.msgbox, c.name)
	}

	r.members[c.name] = c
}

func (c *ChatConn) Name() ChatId {
	return c.name
}
func (c *ChatConn) BroadCast(msg Msg) error {
	roomSocket := findRoom(c.belongRoom)

	if roomSocket == nil {
		return RoomNotExists(c.belongRoom)
	}

	roomSocket.broadCast(c.name, msg)
	return nil
}
func (c *ChatConn) Active() {
	c.con.SetReadDeadline(time.Time{})
}

func (c *ChatConn) SetName(name ChatId) {
	c.name = name
}
func (c *ChatConn) JoinRoom(room RoomId, token string) error {
	roomSocket := findRoom(room)

	if roomSocket != nil {
		if roomSocket.token == token {
			roomSocket.addConn(c)
			c.belongRoom = room
		} else {
			// TODO: need owner allowed
			return JoinRoomWithoutAuth(room)
		}
		return nil
	} else {
		if roomSocket, err := createRoom(room, token); err == nil {
			roomSocket.addConn(c)
			c.belongRoom = room
			c.ownRoom = true
		} else {
			return err
		}
	}
	return nil
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
				cmd := msg.EventName
				if h, ok := c.event_handlers[cmd]; ok {
					h(c, &msg)
				} else {
					err = fmt.Errorf("event: %s not registered", cmd)
				}
				if err != nil {
					msg.Content = err.Error()
				}
				c.Emit(msg)
			}
		} else {
			break
		}
	}
}
func (c *ChatConn) On(event_name string, handler func(*ChatConn, *Msg)) {
	c.event_handlers[event_name] = handler
}

func (c *ChatConn) Emit(msg Msg) error {
	if msg.Type == EventMsg {
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(msg); err == nil {
			return c.con.WriteMessage(websocket.TextMessage, buf.Bytes())
		} else {
			return err
		}
	} else {
		panic("should emit event actions")
	}
}

func (c *ChatConn) LeaveRoom() {
	c.leaveRoom = true
	c.event_handlers = nil
}
func (c *ChatConn) Close() {
	c.LeaveRoom()
	c.con.Close()
}

var upgrader = websocket.Upgrader{}

func NewChatConn(w http.ResponseWriter, r *http.Request) (chat *ChatConn, e error) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	name := ws.Subprotocol()
	chat = &ChatConn{ws, ChatId(name), "", false, false, map[string]func(*ChatConn, *Msg){}}
	chat.Active()
	chat.Emit(Msg{
		Type:      EventMsg,
		EventName: "connect",
	})
	go chat.waitMessage()

	return chat, nil
}
