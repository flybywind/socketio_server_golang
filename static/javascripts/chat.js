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

function ChatSocket(host, user_name) {
  this.ws = new WebSocket("ws://" + host + "/ws/", user_name);
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
  var own_evt = {
    "message": 1,
    "open": 1,
    "close": 1,
    "error": 1
  };

  if (own_evt.hasOwnProperty(evt_name)) {
    this.ws["on" + evt_name] = callback;
  } else {
    // user defined event:
    var oldhandler = this.ws.onmessage;
    this.ws.onmessage = function(evt) {
      if (typeof oldhandler === "function") {
        oldhandler(evt);
      }
      var msg = new Msg();
      msg.parseJson(evt.data);
      if (msg.EventName == evt_name &&
        typeof callback == "function") {
        callback(msg, evt);
      }
    }
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
    chatconn.Emit("JoinRoom", _room);
  })
  chatconn.On("enter_room", function(msg) {
    var divContent = "<div class=serverMessage>";
    if (msg.Content !== "-") {
      divContent += (msg.Content + " 进入聊天室…… ");
    } else {
      divContent += (msg.Content + " 无法进入聊天室…… ");
    }
    domMsg.append(divContent + "</div>");
  });

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

  chatconn.On("error", function(evt) {
    print("ERROR: " + evt.data);
  })

  window.onunload = function() {
    chatconn.Emit("Close");
  }
})
