var newMsg = ''; // Holds new messages to be sent to the server
var chatContent = ''; // A running list of chat messages displayed on the screen
var username = null; // Our username
var socket = new WebSocket('ws://' + window.location.host + '/ws');
socket.onmessage = function(e) {
  var msg = JSON.parse(e.data);
  if (msg.command == "START GAME") {
    battle();
    return;
  }
  chatContent += '<div class="chip">'
   + msg.username
   + '</div>'
   + (msg.message) + '<br/>';
  var element = document.getElementById('chat-messages');
  element.innerHTML=chatContent;
  element.scrollTop = element.scrollHeight; // Auto scroll to the bottom
};

function send () {
    newMsg=document.getElementById("msgbox").value;
    console.log("newmsg is"+newMsg);
    if (newMsg != '') {
        socket.send(
            JSON.stringify({
                username: username,
                message: newMsg,
		command: ""
            }
        ));
        document.getElementById("msgbox").value=""; // Reset the message box
    }
}

function join () {
    username = document.getElementById("usernamebox").value;
    console.log("username is "+username);
    if (!username) {
        Materialize.toast('You must choose a username', 2000);
        return;
    }
    document.getElementById("afterjoin").style.display = "block";
    document.getElementById("beforejoin").style.display = "none";
}

function toggleReady () {
    console.log("(Un)readying for game...");
    readyStatus = document.getElementById("readybutton").innerHTML;
    if (readyStatus.search("Unready for game")==-1) {
        var command = "READY";
        document.getElementById("readybutton").innerHTML="Unready for game";
    } else {
        var command = "UNREADY";
        document.getElementById("readybutton").innerHTML="Ready for game";
    }
    socket.send(
        JSON.stringify({
            username:username,
            message: "",
            command: command
	}
    ));
}

function enter (event) {
    if (event.keyCode == 13) {
        send()
    }
}

function battle () {
  document.getElementById('chat').style.display="none"
  document.getElementById('battleUI').style.display="block"
  // should probably play a sound to notify the user when they get matched
  var input = "NONE"
  document.addEventListener('keyup', function(e) {
    if (e.keyCode == 32) {
      input = "NONE"
      }
    return
  });
  document.addEventListener('keydown', function(e) {
    if (e.keyCode == 32) {
      input = "BLOCK"
      return
    }
    if (e.keyCode == 81) {
      input = "LIGHT"
      return
    }
    if (e.keyCode == 87) {
      input = "HEAVY"
      return
    }
    if (e.shiftKey) {
      input = "DODGE"
      return
    }
    if (e.ctrlKey) {
      input = "SAVE"
      return
    }
  });
  socket.onmessage = function(e) {
    var update = JSON.parse(e.data)
    document.getElementById('ownLife').style.width=update.self.life.toString()+"%"
    document.getElementById('ownStam').style.width=update.self.stamina.toString()+"%"
    document.getElementById('ownDuration').style.width=update.self.stateDur.toString()+"%"
    document.getElementById('enemyLife').style.width=update.enemy.life.toString()+"%"
    document.getElementById('enemyStam').style.width=update.enemy.stamina.toString()+"%"
    document.getElementById('enemyDuration').style.width=update.enemy.stateDur.toString()+"%"
    var ownState=update.self.state
    var enemyState=update.enemy.state
    document.getElementById('ownBlockSymbol').style.display="none"
    document.getElementById('ownLightSymbol').style.display="none"
    document.getElementById('ownLeftLightSymbol').style.display="none"
    document.getElementById('ownHeavySymbol').style.display="none"
    document.getElementById('enemyBlockSymbol').style.display="none"
    document.getElementById('enemyLightSymbol').style.display="none"
    document.getElementById('enemyHeavySymbol').style.display="none"
    document.getElementById('enemyRightLightSymbol').style.display="none"
    switch (ownState) {
      case "blocking":
        document.getElementById('ownBlockSymbol').style.display="inline-block"
        break;
      case "light attack":
        document.getElementById('ownLightSymbol').style.display="inline-block"
        break;
      case "heavy attack":
        document.getElementById('ownHeavySymbol').style.display="inline-block"
        break;
      case "counterattack":
        document.getElementById('ownBlockSymbol').style.display="inline-block"
        document.getElementById('ownLightSymbol').style.display="inline-block"
        break;
      case "countered":
        document.getElementById('ownLeftLightSymbol').style.display="inline-block"
        document.getElementById('ownBlockSymbol').style.display="inline-block"
    }
    console.log("checking enemy state")
    switch (enemyState) {
      case "blocking":
        console.log("enemy blocking")
        document.getElementById('enemyBlockSymbol').style.display="inline-block"
        break;
      case "light attack":
        document.getElementById('enemyLightSymbol').style.display="inline-block"
        break;
      case "heavy attack":
        document.getElementById('enemyHeavySymbol').style.display="inline-block"
        break;
      case "counterattack":
        document.getElementById('enemyBlockSymbol').style.display="inline-block"
        document.getElementById('enemyLightSymbol').style.display="inline-block"
        break;
      case "countered":
        document.getElementById('enemyRightLightSymbol').style.display="inline-block"
        document.getElementById('enemyBlockSymbol').style.display="inline-block"
    }
  };
  function sendUpdate () {
//    console.log("sending input to server:",input)
    socket.send(JSON.stringify({
    "username":username,
    "message":input,
    "command":""}))
    if (input!="BLOCK"){
      input = "NONE"
    }
  }
  setInterval(sendUpdate,20)
}