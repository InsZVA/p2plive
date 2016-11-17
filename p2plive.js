/**
 * Created by InsZVA on 2016/11/16.
 */
var PeerConnection = (window.PeerConnection ||
window.webkitPeerConnection00 ||
window.webkitRTCPeerConnection ||
window.mozRTCPeerConnection);

const RETRY_GETSOURCE_INTERVAL = 500,
    UPDATE_INTERVAL = 10000,
    MAX_PUSH_NUM = 3,
    MAX_PULL_NUM = 2;

var trackerAddress, forwardAddress, trackerWS;

var client = {
    pullNum: 0,	//the number of clients this client pull from
    pushNum: 0, //the number of clients this client push to
    pulls: [],
    pushs: []
};
var getSourceTimer, updateTimer;

var canvas, player, directwsclient;
canvas = document.getElementById('videoCanvas');
var ctx=canvas.getContext('2d');
ctx.fillStyle='#FF0000';
ctx.fillRect(0,0,80,100);

$.get("http://127.0.0.1:8080/tracker", function(data) {
    trackerAddress = data;
    console.log("选择" + trackerAddress + "作为Tracker服务器\n");
    trackerWS = new WebSocket( 'ws://' + trackerAddress + "/resource");
    trackerWS.onmessage = function(event) {
        msg = JSON.parse(event.data);
        var pc;
        switch (msg.type) {
            case "directPull":
                clearInterval(getSourceTimer);
                forwardAddress = msg.address;
                client.pullNum = 1;
                // Setup the WebSocket connection and start the player
                directwsclient = new WebSocket( 'ws://127.0.0.1:9998/' );
                player = new decoder(directwsclient, {canvas:canvas});
                break;
            case "push":
                var address = msg.address;
                if (client.pushs.length > 3) {
                    console.log("推流超过" + MAX_PUSH_NUM);
                    return;
                }
                pc = new PeerConnection({"iceServers": []});
                pc.onicecandidate = function(event){
                    trackerWS.send(JSON.stringify({
                        "method": "candidate",
                        "candidate": event.candidate,
                        "address": address
                    }));
                };

                var dc = pc.createDataChannel("live stream", {
                    ordered:true,
                    maxRetransmitTime: 3000
                });

                dc.onmessage = function (event) {
                    console.log("received: " + event.data);
                };

                dc.onopen = function () {
                    console.log("datachannel open");
                    dc.send("123");
                    player.addForwardDC(this);
                };

                dc.onclose = function () {
                    console.log("datachannel close");
                };
                console.log(dc);

                pc.createOffer().then(function(offer) {
                    return pc.setLocalDescription(offer);
                }).then(function() {
                    trackerWS.send(JSON.stringify({
                        "method": "offer",
                        "sdp": pc.localDescription,
                        "address": address
                    }));
                });

                for (var i = 0;i < MAX_PUSH_NUM;i++)
                    if (client.pushs[i] == undefined || client.pushs[i].state == "close") {
                        client.pushs[i] = {
                            pc: pc,
                            state: "starting",
                            remote: address,
                            dc: dc
                        };
                        break;
                    }
                break;
            case "pull":
                pc = new PeerConnection({"iceServers": []});
                var address = msg.address;
                pc.onicecandidate = function(event){
                    trackerWS.send(JSON.stringify({
                        "method": "candidate",
                        "candidate": event.candidate,
                        "address": address
                    }));
                };
                for (var i = 0;i < MAX_PULL_NUM;i++)
                    if (client.pulls[i] == undefined || client.pulls[i].state == "close") {
                        client.pulls[i] = {
                            pc: pc,
                            state: "starting",
                            remote: address
                        };
                        break;
                    }

                break;
            case "candidate":
                for (var i = 0;i < MAX_PULL_NUM;i++)
                    if (client.pulls[i] && client.pulls[i].state == "starting") {
                        pc = client.pulls[i].pc;
                        break;
                    }
                for (var i = 0;i < MAX_PUSH_NUM;i++)
                    if (client.pushs[i] && client.pushs[i].state == "starting") {
                        pc = client.pushs[i].pc;
                        break;
                    }
                    if (msg.candidate != null)
                        pc.addIceCandidate(new RTCIceCandidate(msg.candidate));
                break;
            case "offer":
                var address, pull;
                for (var i = 0;i < MAX_PULL_NUM;i++)
                    if (client.pulls[i] && client.pulls[i].state == "starting") {
                        pull = client.pulls[i];
                        pc = client.pulls[i].pc;
                        address = client.pulls[i].remote;
                        break;
                    }
                pc.setRemoteDescription(new RTCSessionDescription(msg.sdp));
                pc.createAnswer().then(function(answer) {
                    return pc.setLocalDescription(answer);
                }).then(function() {
                    trackerWS.send(JSON.stringify({
                        "method": "answer",
                        "sdp": pc.localDescription,
                        "address": address
                    }));
                });
                client.pullNum++;
                pc.ondatachannel = function(ev) {
                    console.log('Data channel is created!');
                    player = new decoder(ev.channel, {canvas:canvas});
                    pull.dc = ev.channel;
                    pull.state = "transferring";

                    if (client.pullNum < MAX_PULL_NUM && directwsclient == undefined) {
                        trackerWS.send(JSON.stringify({method: "getSource"}));
                    }
                };
                break;
            case "answer":
                var push;
                for (var i = 0;i < MAX_PUSH_NUM;i++)
                    if (client.pushs[i] && client.pushs[i].state == "starting") {
                        push = client.pushs[i];
                        pc = client.pushs[i].pc;
                        break;
                    }
                pc.setRemoteDescription(new RTCSessionDescription(msg.sdp));
                client.pushNum++;
        }
    };

    var update = function() {
        trackerWS.send(JSON.stringify({
            method: "update",
            pullNum: client.pullNum,
            pushNum: client.pushNum
        }));
    };

    trackerWS.onopen = function() {
        trackerWS.send(JSON.stringify({method: "getSource"}));
        updateTimer = setInterval(update, UPDATE_INTERVAL);
    }
});