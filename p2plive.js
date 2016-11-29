/**
 * Created by InsZVA on 2016/11/16.
 */
var PeerConnection = (window.PeerConnection ||
window.webkitPeerConnection00 ||
window.webkitRTCPeerConnection ||
window.mozRTCPeerConnection);

const
    UPDATE_INTERVAL = 10000,
    MAX_PUSH_NUM = 3,
    MAX_PULL_NUM = 2;

var trackerAddress, forwardAddress, trackerWS;

var client = {
    pullNum: 0,	//the number of clients this client pull from
    pushNum: 0, //the number of clients this client push to
    pulls: [],
    pushs: [],
    preferId: -1,
    forwardTimes: 0,
    delay: 0
};
var getSourceTimer, updateTimer;

var canvas, player, directwsclient;
canvas = document.getElementById('videoCanvas');

var debug = true;

var pcConfig = {"iceServers": [{
    "url": "stun:stun.voipbuster.com:19302"
}, {
    "url": "turn:115.159.227.38",
    "username": "inszva",
    "credential": "0x471150246375ea14baaf678d678ceada"
}]};

//var pcConfig = {"iceServers": []};

var update = function() {
    trackerWS.send(JSON.stringify({
        method: "update",
        pullNum: client.pullNum,
        pushNum: client.pushNum
    }));
};

var selector = {
    pullState: 'close',
    pushState: 'close',
    cacheTime: 1000,
    interval: 1,
    onmessage: null,
    onopen: null,
    onerror: null,
    onclose: null,
    pcTimeout: 1000,
    analyseDuration: 10000,
    analyseInterval: 1000,
    maxCachedPackets: 32,  // to avoid the cache too large(when chrome doesn't render)
    open: function() {
        setInterval(this.playTimer, this.interval);
        //setInterval(this.forwardTimer, this.interval);
        //setInterval(this.whoisyourdaddyCHROME, 1000);
        this.onopen();
    },
    playTimer: function() {

        // Find better pull source & get earliest create time packet
        var minCreateTime = 9999999999999;
        var preferId = -1;
        var kills = [];
        for (var i = 0; i < client.pulls.length; i++) {
            var cache = client.pulls[i].cache;
            if (cache == undefined) {
                if (client.pulls[i].startTime && new Date().getTime() - client.pulls[i].startTime > selector.pcTimeout) {
                    kills.push(i);
                    console.log("kill:" + i);
                }
                continue;
            }
            if (cache[0] && cache[0].createTime < minCreateTime) minCreateTime = cache[0].createTime;
            if (cache[0] && (preferId == -1 || [0].forwardTimes < client.pulls[preferId].cache[0].forwardTimes))
                preferId = i;
        }

        // Kill those peerconnections created when pusher reject
        if (kills.length != 0) {
            console.log(kills);
            var pulls = [];
            var k = 0;
            for (var i = 0; i < client.pulls.length && k < kills.length; i++) {
                if (i == kills[k]) {
                    if (preferId > i) {
                        preferId--;
                    }
                    client.pulls[i].pc.close();
                    k++;
                    continue;
                }
                pulls.push(client.pulls[i]);
            }
            client.pulls = pulls;
            selector.pullState = "close";
            trackerWS.send(JSON.stringify({method: "getSource"}));
        }
        if (client.preferId == -1)
            client.preferId = preferId;
        if (preferId == -1) return;

        var overtime = minCreateTime + selector.cacheTime;

        // Play those packets that over cache time
        for (var i = client.pulls[preferId].cache.length > selector.maxCachedPackets ?
                client.pulls[preferId].cache.length - selector.maxCachedPackets : 0;
             i < client.pulls[preferId].cache.length; i++) {
            if (client.pulls[preferId].cache[i].createTime < overtime) {
                var messageData = new Uint8Array(client.pulls[preferId].cache[i].event.data);
                client.pulls[preferId].cache[i].event.data = messageData.slice(11);
                if (!document.hidden)
                    selector.onmessage(client.pulls[preferId].cache[i].event);

                //Analyse
                client.forwardTimes = client.pulls[preferId].cache[i].forwardTimes;
                var delays = 0;
                for (var j = 0; j < selector.analyseDuration / selector.analyseInterval; j++) {
                    if (client.pulls[preferId] && client.pulls[preferId].delay && client.pulls[preferId].delay[j])
                        delays += client.pulls[preferId].delay[j];
                }
                client.delay = delays / (selector.analyseDuration / selector.analyseInterval);
            } else {
                break;
            }
        }

        // Clean packets in all pull sources' cache that over time
        for (var i = 0;i < client.pulls.length;i++) {
            for (var j = 0; j < client.pulls[i].cache.length; j++) {
                if (client.pulls[i].cache[i] && client.pulls[i].cache[i].createTime > overtime)
                    break;
            }
            client.pulls[i].cache = client.pulls[i].cache.slice(j);
        }

    },
    refreshAvailablePuller: function() {
        var availablePullSource = 0;

        // Clear the closed pull source
        var pulls = [];
        for (var i = 0;i < client.pulls.length;i++) {
            if (client.pulls[i].pc && (client.pulls[i].pc.iceConnectionState == "disconnected" ||
                client.pulls[i].pc.iceConnectionState == "failed" || client.pulls[i].pc.iceConnectionState == "closed"))
            {
                if (client.pulls[i].dc) {
                    client.pulls[i].dc.close();
                }
                continue;
            }
            if (!client.pulls[i].dc || client.pulls[i].dc.readyState == "closing" || client.pulls[i].dc.readyState == "closed")
                if (selector.pullState != "connecting" || !client.pulls[i].pc)
                    continue;
            if (client.pulls[i].dc && client.pulls[i].dc.readyState == "open")
                availablePullSource++;
            pulls.push(client.pulls[i]);
        }
        client.pulls = pulls;

        return availablePullSource;
    },
    refreshPushingTarget: function() {
        var pushingTarget = 0;

        // Clear the closed push target
        var pushs = [];
        for (var i = 0;i < client.pushs.length;i++) {
            if (client.pushs[i].pc && (client.pushs[i].pc.iceConnectionState == "disconnected" ||
                client.pushs[i].pc.iceConnectionState == "failed" || client.pushs[i].pc.iceConnectionState == "closed"))
            {
                if (client.pushs[i].dc) {
                    client.pushs[i].dc.close();
                }
                continue;
            }
            if (!client.pushs[i].dc || client.pushs[i].dc.readyState == "closing" || client.pushs[i].dc.readyState == "closed")
                if (selector.pushState != "connecting" || !client.pushs[i].pc)
                    continue;
            if (client.pushs[i].dc && client.pushs[i].dc.readyState == "open")
                pushingTarget++;
            pushs.push(client.pushs[i]);
        }
        client.pushs = pushs;

        return pushingTarget;
    },
    getPulling: function() {
        return client.pulls[client.pulls.length - 1]
    },
    getPushing: function() {
        return client.pushs[client.pushs.length - 1];
    },
    refresh: function(newState, newSocket, external) {
        switch (newState) {
            case 'pulling':
                if (selector.pushState == "connecting") {
                    // Avoid this case
                }
                var availablePullSource = selector.refreshAvailablePuller();
                if (newSocket instanceof PeerConnection) {
                    if (selector.pullState == "direct") {
                        console.log("already has direct pull source!");
                        newSocket.close();
                        break;
                    }
                    if (availablePullSource >= MAX_PULL_NUM) {
                        console.log("already has enough pull source!");
                        newSocket.close();
                        break;
                    }
                    if (selector.pullState == "connecting") {
                        console.log("only process a socket in one time");
                        newSocket.close();
                        break;
                    }
                    client.pulls.push({
                        pc: newSocket,
                        remote: external,
                        startTime: new Date().getTime(),
                        receiveBytes: [],
                        delay: [],
                        analyseLoopTime: new Date().getTime()
                    });
                } else {
                    console.error("assert type PeerConnection");
                }
                selector.pullState = "connecting";
                break;
            case "pushing":
                var pushingTarget = selector.refreshPushingTarget();
                if (newSocket instanceof PeerConnection) {
                    if (selector.pullState == "connecting") {
                        newSocket.close();
                        client.pushNum = pushingTarget;
                        break;
                    }
                    if (pushingTarget >= MAX_PULL_NUM) {
                        console.log("already push many targets!");
                        newSocket.close();
                    }
                    client.pushs.push({
                        pc: newSocket,
                        remote: external,
                        startTime: new Date().getTime(),
                        sendBytes: [],
                        analyseLoopTime: new Date().getTime()
                    });
                } else {
                    console.error("assert type PeerConnection");
                }
                selector.pushState = "connecting";
                break;
            case 'pull':
                if (selector.pullState == 'connecting') {
                    var availablePullSource = selector.refreshAvailablePuller();
                    var pull = selector.getPulling();
                    newSocket.onopen = function () {
                        console.log("Datachannel open");
                        pull.cache = [];
                        newSocket.binaryType = 'arraybuffer';
                        newSocket.onmessage = function (event) {
                            var messageData = new Uint8Array(event.data);
                            var cache = {};
                            cache.forwardTimes = messageData[6];
                            cache.createTime = messageData[7] + messageData[8] * 256 + messageData[9] * 256 * 256 +
                                messageData[10] * 256 * 256 * 256;
                            cache.createTime *= 1000;
                            cache.event = event;
                            //if (pull.cache.length < selector.maxCachedPackets)
                            pull.cache.push(cache);

                            //Forward
                            if (client.pulls[client.preferId] &&
                                client.pulls[client.preferId].dc &&
                                client.pulls[client.preferId].dc == newSocket ) {
                                for (var j = 0; j < client.pushs.length; j++) {
                                    if (client.pushs[j].dc && client.pushs[j].dc.readyState == "open") {
                                        messageData[6] = messageData[6]+1;
                                        client.pushs[j].dc.send(messageData);

                                        //Analyse
                                        var push = client.pushs[j];
                                        var t = new Date().getTime();
                                        if (t - push.analyseLoopTime < selector.analyseInterval) {
                                            push.sendBytes[0] = push.sendBytes[0] ? push.sendBytes[0]
                                            + messageData.length : messageData.length;
                                        } else {
                                            var loops = selector.analyseDuration / selector.analyseInterval;
                                            for (var i = loops - 1; i > 0; i--) {
                                                push.sendBytes[i] = push.sendBytes[i - 1];
                                            }
                                            push.sendBytes[0] = messageData.length;
                                            push.analyseLoopTime = t;
                                        }
                                    }
                                }
                            }

                            //Analyse
                            var t = new Date().getTime();
                            if (t - pull.analyseLoopTime < selector.analyseInterval) {
                                pull.receiveBytes[0] = pull.receiveBytes[0] ? pull.receiveBytes[0]
                                + messageData.length : messageData.length;
                                pull.delay[0] = t - cache.createTime;
                            } else {
                                var loops = selector.analyseDuration / selector.analyseInterval;
                                for (var i = loops - 1; i > 0; i--) {
                                    pull.receiveBytes[i] = pull.receiveBytes[i - 1];
                                    pull.delay[i] = pull.delay[i - 1];
                                }
                                pull.receiveBytes[0] = messageData.length;
                                pull.delay[0] = t - cache.createTime;
                                pull.analyseLoopTime = t;
                            }
                        };
                        newSocket.onclose = function () {
                            trackerWS.send(JSON.stringify({method: "getSource"}));
                        };
                    };
                    client.pullNum = selector.refreshAvailablePuller();
                    selector.pullState = "pull";
                    //For debug eaily 1to1
                } else {
                    console.log("only process a pull at one time");
                    newSocket.close();
                }
                break;
            case 'push':
                if (selector.pushState == "connecting") {
                    var pushingTarget = selector.refreshPushingTarget();
                    var push = selector.getPushing();
                    console.log(client);
                    push.dc = newSocket;
                    client.pushNum = pushingTarget + 1;

                }

                break;
            case "directPull":
                for (var i = 0; i < client.pulls.length; i++) {
                    if (client.pulls[i].ws) {
                        if (client.pulls[i].ws.readyState == 1/*OPEN*/) {
                            console.error("You are already pull from a forward server");
                            return;
                        }
                        client.pulls[i].ws.close();
                    }
                    if (client.pulls[i].dc) {
                        client.pulls[i].dc.close();
                    }
                }
                client.pulls = [];
                client.pulls.push({
                    ws: newSocket,
                    cache: [],
                    receiveBytes: [],
                    delay: [],
                    analyseLoopTime: new Date().getTime()
                });
                var pull = client.pulls[0];
                newSocket.onopen = function () {
                    newSocket.binaryType = 'arraybuffer';
                    newSocket.onmessage = function (event) {
                        var messageData = new Uint8Array(event.data);
                        var cache = {};
                        cache.forwardTimes = messageData[6];
                        cache.createTime = messageData[7] + messageData[8] * 256 + messageData[9] * 256 * 256 +
                            messageData[10] * 256 * 256 * 256;
                        cache.createTime *= 1000;
                        cache.event = event;
                        //if (pull.cache.length < selector.maxCachedPackets)
                        pull.cache.push(cache);

                        //Forward
                        if (client.pulls[client.preferId] && client.pulls[client.preferId].ws &&
                            client.pulls[client.preferId].ws == newSocket ) {
                            for (var j = 0; j < client.pushs.length; j++) {
                                if (client.pushs[j].dc && client.pushs[j].dc.readyState == "open") {
                                    messageData[6] = messageData[6]+1;
                                    client.pushs[j].dc.send(messageData);

                                    //Analyse
                                    var push = client.pushs[j];
                                    var t = new Date().getTime();
                                    if (t - push.analyseLoopTime < selector.analyseInterval) {
                                        push.sendBytes[0] = push.sendBytes[0] ? push.sendBytes[0]
                                        + messageData.length : messageData.length;
                                    } else {
                                        var loops = selector.analyseDuration / selector.analyseInterval;
                                        for (var i = loops - 1; i > 0; i--) {
                                            push.sendBytes[i] = push.sendBytes[i - 1];
                                        }
                                        push.sendBytes[0] = messageData.length;
                                        push.analyseLoopTime = t;
                                    }
                                }
                            }
                        }

                        //Analyse
                        var t = new Date().getTime();
                        if (t - pull.analyseLoopTime < selector.analyseInterval) {
                            pull.receiveBytes[0] = pull.receiveBytes[0] ? pull.receiveBytes[0]
                            + messageData.length : messageData.length;
                            pull.delay[0] = t - cache.createTime;
                        } else {
                            var loops = selector.analyseDuration / selector.analyseInterval;
                            for (var i = loops - 1; i > 0; i--) {
                                pull.receiveBytes[i] = pull.receiveBytes[i - 1];
                                pull.delay[i] = pull.delay[i - 1];
                            }
                            pull.receiveBytes[0] = messageData.length;
                            pull.delay[0] = t - cache.createTime;
                            pull.analyseLoopTime = t;
                        }
                        ;
                        newSocket.onclose = function () {
                            trackerWS.send(JSON.stringify({method: "getSource"}));
                        };
                    };
                    client.pullNum = 1;
                    selector.pullState = "directPull";
                };
        }
        update();
    }
};

player = new decoder(selector, {canvas:canvas});
selector.open();

$.get("http://127.0.0.1:8080/tracker", function(data) {
    trackerAddress = data;
    console.log("select" + trackerAddress + "as a tracker server\n");
    trackerWS = new WebSocket( 'ws://' + trackerAddress + "/resource");
    trackerWS.onmessage = function(event) {
        msg = JSON.parse(event.data);
        var pc;
        switch (msg.type) {
            case "directPull":
                forwardAddress = msg.address;
                //client.pullNum = 1;
                // Setup the WebSocket connection and start the player
                directwsclient = new WebSocket( 'ws://127.0.0.1:9998/' );
                //player = new decoder(directwsclient, {canvas:canvas});
                selector.refresh("directPull", directwsclient);
                break;
            case "push":
                for (var i = 0; i < client.pushs.length; i++) {
                    console.log(client.pushs[i].remote);
                    if (client.pushs[i].remote == msg.address) {

                        return;
                    }
                }
                var address = msg.address;
                pc = new PeerConnection(pcConfig);
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
                    selector.refresh('push', this);
                };

                dc.onclose = function () {
                    console.log("datachannel close");
                };

                pc.createOffer().then(function(offer) {
                    return pc.setLocalDescription(offer);
                }).then(function() {
                    trackerWS.send(JSON.stringify({
                        "method": "offer",
                        "sdp": pc.localDescription,
                        "address": address
                    }));
                });
                console.log(pc);
                selector.refresh("pushing", pc, address);
                break;
            case "pull":
                for (var i = 0; i < client.pulls.length; i++) {

                    if (client.pulls[i].remote == msg.address) {
                        console.log(client.pulls[i].remote + " has already been pull source, try get another");
                        setTimeout(function() {trackerWS.send(JSON.stringify({method: "getSource"})); },
                            500);
                        return;
                    }
                }
                pc = new PeerConnection(pcConfig);
                var address = msg.address;
                pc.onicecandidate = function(event){
                    trackerWS.send(JSON.stringify({
                        "method": "candidate",
                        "candidate": event.candidate,
                        "address": address
                    }));
                };
                pc.onclose = function() {
                    selector.pullState = "close";
                    if (selector.pullState == "direct")
                        return;

                    var availablePullSource = selector.refreshAvailablePuller();
                    if (availablePullSource < MAX_PULL_NUM) {
                        trackerWS.send(JSON.stringify({method: "getSource"}));
                    }
                };
                selector.refresh('pulling', pc, address);
                break;
            case "candidate":
                var p;
                if (selector.pushState == "connecting")
                    p = selector.getPushing();
                else
                    p = selector.getPulling();
                if (msg.candidate != null) {
                    p.pc.addIceCandidate(new RTCIceCandidate(msg.candidate));
                }
                break;
            case "offer":
                var address, pull;
                if (selector.pullState == "connecting") {
                    pull = selector.getPulling();
                    pc = pull.pc;

                    pc.setRemoteDescription(new RTCSessionDescription(msg.sdp));
                    pc.createAnswer().then(function(answer) {
                        return pc.setLocalDescription(answer);
                    }).then(function() {
                        trackerWS.send(JSON.stringify({
                            "method": "answer",
                            "sdp": pc.localDescription,
                            "address": pull.remote
                        }));
                    });

                    pc.ondatachannel = function(ev) {
                        console.log('Data channel is created!');
                        pull.dc = ev.channel;
                        selector.refresh("pull", pull.dc);
                        trackerWS.send(JSON.stringify({method: "getSource"}));
                    };
                } else {
                    // TODO
                }

                break;
            case "answer":
                if (selector.pushState == "connecting") {
                    var push = selector.getPushing();
                    push.pc.setRemoteDescription(new RTCSessionDescription(msg.sdp));
                } else {
                    //TODO
                }
        }
    };

    trackerWS.onopen = function() {
        trackerWS.send(JSON.stringify({method: "getSource"}));
        updateTimer = setInterval(update, UPDATE_INTERVAL);
    }
});

if (debug) {
    var debugDiv = document.createElement("div");
    debugDiv.setAttribute("style", "position:absolute; right: 20px; font-size: 16px; top: 20px; color: red; text-align:left");
    document.body.appendChild(debugDiv);
    setInterval(function(){
        if (!client.pulls || client.pulls.length == 0) return;
        var s = "拉流信息：<br>";
        if (selector.pullState == "directPull") {
            s += "直接从服务器拉取：";
            var rate = 0;
            for (var i = 0; i < selector.analyseDuration / selector.analyseInterval; i++) {
                if (client.pulls[0] && client.pulls[0].receiveBytes[i])
                    rate += client.pulls[0].receiveBytes[i];
            }
            rate = rate / (selector.analyseDuration / selector.analyseInterval);
            s += rate/128 + "KBps<br>";
        } else {
            for (var i = 0; i < client.pulls.length; i++) {
                s += client.pulls[i].remote + ":";
                var rate = 0;
                for (var j = 0; j < selector.analyseDuration / selector.analyseInterval; j++) {
                    if (client.pulls[i] && client.pulls[i].receiveBytes[j])
                        rate += client.pulls[i].receiveBytes[j];
                }
                rate = rate / (selector.analyseDuration / selector.analyseInterval);
                s += rate/128 + "KBps<br>";
            }
        }
        s += "最优转发次数：" + client.forwardTimes + "<br>";
        s += "平均延迟（10s内都算优）：" + client.delay + "ms<br>";

        s += "推流信息：<br>";
        for (var i = 0; i < client.pushs.length; i++) {
            s += client.pushs[i].remote + ":";
            var rate = 0;
            for (var j = 0; j < selector.analyseDuration / selector.analyseInterval; j++) {
                if (client.pushs[i] && client.pushs[i].sendBytes[j])
                    rate += client.pushs[i].sendBytes[j];
            }
            rate = rate / (selector.analyseDuration / selector.analyseInterval);
            s += rate/128 + "KBps<br>";
        }

        debugDiv.innerHTML = s;
    }, 1000
    )
}

//TODO List
// 互相建立的DataChannel其实收不到数据 -- solve
// chrome如果不渲染，cache里面会积累大量数据，然后导致线程卡死 -- solve
// 第3个客户端连入的情况 -- solve
// 解码器速度堪忧 考虑换其他解码器
// 在某个push源push数量满的时候，pull会感知不到，故而卡死 -- solve
// STUN服务器太慢，导致延迟叠加