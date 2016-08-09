function Msg() {
  this.Content = "";
  this.EventName = "";
  this.Token = "";
  this.Type = "9";
}

Msg.prototype.toJson = function() {
  return JSON.stringify(this)
}

Msg.prototype.parseJson = function(str) {
  var o = JSON.parse(str);
  for (var p in o) {
    if (this.hasOwnProperty(p)) {
      this[p] = o.p;
    }
  }
}

function ChatSocket(host) {
  this.ws = new WebSocket("ws://" + host + "/ws/");
}
ChatSocket.prototype.send = function(message, callback) {
  var ws = this.ws;
  var _send = function() {
    if (ws.readyState === 1) {
      ws.send(message.toJson());
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
ChatSocket.prototype.on = function(evt, callback) {
  this.ws["on" + evt] = callback;
}
$(function() {
  var OwnMsg = "0",
    NormalMsg = "1",
    EventMsg = "9";
  var cur_url = location.href,
    host = cur_url.split("/")[2],
    chatconn = new ChatSocket(host);

  (function() {
    var msg = new Msg();
    msg.EventName = "JoinRoom";
    msg.Content = _room;
    chatconn.send(msg.toJson());

    msg.EventName = "SetName";
    msg.Content = _user;
    chatconn.send(msg.toJson());
  }());

  chatconn.on("message", function(evt) {
    var msg = new Msg();
    msg.parseJson(evt.data);

    console.log(msg);
  })

  chatconn.on("error", function(evt) {
    print("ERROR: " + evt.data);
  })

  $("#send").click(function() {
    var text = $("#message").val();
    var msg = new Msg();
    msg.EventName = "BroadCast";
    msg.RoomName = _room;
    msg.Content = text;
    chatconn.send(msg.toJson());
  })
  $(window).unload(function() {
    msg.EventName = "Close";
    msg.RoomName = _room;
    chatconn.send(msg.toJson());
  })
})
