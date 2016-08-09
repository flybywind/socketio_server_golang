
$(function() {
    var $name = $(".user_info .name").eq(0),
        $room = $(".user_info .room").eq(0);
    $(".enter").click(function() {
        var name = $name.val(),
            room = $room.val();
        if (!name || !room) {
            alert("please input name or room name!")
            return
        }
        location.href = "/chat_room/" + room + "." + name;
    })
})
