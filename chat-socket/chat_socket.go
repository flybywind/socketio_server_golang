package chatsocket

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"regexp"
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
	id      RoomId
	members map[ChatId]*ChatConn
	msgbox  map[ChatId]time.Time
}
type ChatConn struct {
	con            *websocket.Conn
	id             ChatId
	belongRoom     RoomId
	ownRoom        bool
	leaveRoom      bool
	event_handlers map[string]func(*ChatConn, *Msg)
}

func createRoom(id RoomId, token string) (*roomSocket, error) {
	room, exists := rooms[id]
	if exists {
		return nil, fmt.Errorf("room [%s] exists", id)
	}
	room = &roomSocket{
		token, id,
		map[ChatId]*ChatConn{},
		map[ChatId]time.Time{},
	}

	rooms[id] = room
	log.Println("create room:", id)
	return room, nil
}
func findRoom(id RoomId) *roomSocket {
	room, exists := rooms[id]
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
				if conn.Id() == fromChat {
					msg.Type = OwnMsg
					// 如果指定为长度为buf.Len()的[]byte，会出现乱码
					buf2 := bytes.NewBuffer(nil)
					err = json.NewEncoder(buf2).Encode(msg)
					if err != nil {
						return err
					}
					return conn.con.WriteMessage(websocket.TextMessage, buf2.Bytes())
				} else {
					return conn.con.WriteMessage(websocket.TextMessage, buf.Bytes())
				}
				log.Println("broadcast to user:", conn.Id())
			} else {
				now := time.Now()
				if _, exists := r.msgbox[conn.id]; !exists {
					r.msgbox[conn.id] = now
				}
				// TODO: get db to store messages
				log.Println("user:", conn.Id(), "has leaved")
			}
		}
	} else {
		return err
	}
	return nil
}

func (r *roomSocket) addConn(c *ChatConn) {
	if conn, existsbefore := r.members[c.id]; existsbefore {
		c.ownRoom = conn.ownRoom
		// TODO: retreive older messages in msgbox
		delete(r.msgbox, c.id)
	}

	log.Println("add user:", c.id, "in room:", r.id)
	r.members[c.id] = c
}

func (c *ChatConn) BroadCast(msg Msg) error {
	roomSocket := findRoom(c.belongRoom)

	if roomSocket == nil {
		return RoomNotExists(c.belongRoom)
	}

	return roomSocket.broadCast(c.id, msg)
}
func (c *ChatConn) Active() {
	c.con.SetReadDeadline(time.Time{})
}

func (c *ChatConn) Id() ChatId {
	return c.id
}
func (c *ChatConn) SetId(id ChatId) {
	c.id = id
}

func (c *ChatConn) HasId() bool {
	return (len(string(c.id)) > 0)
}
func (c *ChatConn) JoinRoom(room RoomId, token string) error {
	if !c.HasId() {
		panic("haven't set ID on chat before join room")
	}
	roomSocket := findRoom(room)

	if roomSocket != nil {
		if roomSocket.token == token {
			roomSocket.addConn(c)
			c.belongRoom = room
			log.Println("add chat to existing room:", room, roomSocket.members)
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
			cmd := msg.EventName
			switch msg.Type {
			case NormalMsg:
				err = c.BroadCast(msg)
			case EventMsg:
				if h, ok := c.event_handlers[cmd]; ok {
					h(c, &msg)
				} else {
					err = fmt.Errorf("event: %s not registered", cmd)
				}
			}
			if err != nil {
				msg.Type = EventMsg
				msg.EventName = pascal2underline(cmd) + "_fail"
				msg.Content = err.Error()
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
		panic(fmt.Sprintf("should emit event actions: now message type = %s", msg.Type))
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
	chat = &ChatConn{ws, ChatId(""), "", false, false, map[string]func(*ChatConn, *Msg){}}
	chat.Active()
	chat.Emit(Msg{
		Type:      EventMsg,
		EventName: "connect",
	})
	go chat.waitMessage()

	return chat, nil
}

var _pascal2underline_patt = regexp.MustCompile("[A-Z]")

func pascal2underline(var_name string) string {
	new_var_name := _pascal2underline_patt.ReplaceAllString(var_name, "_")
	if len(new_var_name) < 2 {
		return new_var_name
	}
	return new_var_name[1:]
}
