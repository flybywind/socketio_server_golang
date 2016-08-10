var OwnMsg = "0",
  NormalMsg = "1",
  NoticeMsg = "2",
  EventMsg = "9";

function Msg() {
  this.Content = "";
  this.EventName = "";
  this.Token = "";
  this.Type = EventMsg;
}

Msg.prototype.toJson = function() {
  return JSON.stringify(this)
}

Msg.prototype.parseJson = function(str) {
  var o = JSON.parse(str);
  for (var p in o) {
    if (this.hasOwnProperty(p)) {
      this[p] = o[p];
    }
  }
}

function ChatSocket(host) {
  this.ws = new WebSocket("ws://" + host + "/ws/");
  this.own_evt = {
    "open": 1,
    "close": 1,
    "error": 1
  };
  this.user_evt_handler = {};
}
ChatSocket.prototype.isFunction = function(f) {
  return (typeof f === "function")
}
ChatSocket.prototype.Emit = function(evt_name, content, callback) {
  var ws = this.ws;
  var msg = new Msg();
  msg.EventName = evt_name;
  msg.Content = content;
  var _send = function() {
    if (ws.readyState === 1) {
      ws.send(msg.toJson());
      if (typeof callback == "function") {
        callback();
      }
      return true;
    }
    return false;
  }
  if (!_send()) {
    var tid = setInterval(function() {
      if (_send()) {
        clearInterval(tid);
      }
    }, 10)
  }
}
ChatSocket.prototype.On = function(evt_name, callback) {
  var self = this,
    own_evt = this.own_evt;
  if (own_evt.hasOwnProperty(evt_name)) {
    self.ws["on" + evt_name] = callback;
  } else {
    if (!self.isFunction(self.ws.onmessage)) {
      self.ws.onmessage = function(evt) {
        var msg = new Msg();
        msg.parseJson(evt.data);
        var f = self.user_evt_handler[msg.EventName];

        if (self.isFunction(f)) {
          f(msg, evt)
        }
      }
    }
    self.user_evt_handler[evt_name] = callback;
  }
}
ChatSocket.prototype.BroadCast = function(msg) {
  var msg = new Msg();
  msg.EventName = "BroadCast";
  msg.Content = msg;
  msg.Type = NormalMsg;
  this.ws.send(msg.toJson());
}

$(function() {
  var cur_url = location.href,
    host = cur_url.split("/")[2],
    chatconn = new ChatSocket(host),
    domMsg = $("#messages");
  // 从服务器发过来的事件都是下划线命名法
  // 从客服端发送过去的事件是Pascal命名法
  chatconn.On("connect", function() {
    chatconn.Emit("JoinRoom", _room + ":" + _user);
  })
  chatconn.On("enter_room", function(msg) {
    var divContent = "<div class=serverMessage>" +
      msg.Content + " 进入聊天室…… " +
      "</div>";
    domMsg.append(divContent);
  });

  chatconn.On("join_room_fail", function(msg) {
    var divContent = "<div class=serverMessage>" +
      msg.Content + " 无法进入聊天室…… " +
      "</div>";
    domMsg.append(divContent);
  })
  $("#send").click(function() {
    var text = $("#message").val();
    chatconn.BroadCast(text);
  })
  chatconn.On("BroadCast", function(msg) {
    var divContent = "<div class=";
    if (msg.Type == OwnMsg) {
      divContent += "myMessage>"
    } else if (msg.Type == NormalMsg) {
      divContent += "userMessage>"
    }
    domMsg.append(divContent + msg.Content + "</div>");
  })
  chatconn.On("broad_cast_fail", function(msg) {
    console.log("broad_cast_fail:" + msg)
  })
  chatconn.On("error", function(evt) {
    print("ERROR: " + evt.data);
  })

  // window.onunload = function() {
  //   chatconn.Emit("Close");
  // }
})
