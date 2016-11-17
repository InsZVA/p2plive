var decoder = window.decoder = function decoder ( url, opts ) {
    window.jsmpeg.call(this, url, opts);
};

decoder.prototype = new jsmpeg();
decoder.prototype.constructor = decoder;

decoder.prototype.decodeSocketHeader = function( data ) {
	// Custom header sent to all newly connected clients when streaming

    this.width = 640;
    this.height = 480;
    this.initBuffers();
};

decoder.prototype.forwardDCs = [];

decoder.prototype.receiveSocketMessage = function( event ) {
    var messageData = new Uint8Array(event.data);
    event.data = messageData.slice(11);
    // forward data to dc list
    for (var i = 0;i < this.forwardDCs.length;i++) {
        if (this.forwardDCs[i].readyState == "open") {
            this.forwardDCs[i].send(event.data);
        }
    }
    jsmpeg.prototype.receiveSocketMessage.call(this, event);
};

decoder.prototype.addForwardDC = function(dc) {
    this.forwardDCs.push(dc);
};

decoder.prototype.removeForwardDC = function(dc) {
    for (var i = 0;i < this.forwardDCs.length;i++) {
        if (this.forwardDCs[i] == dc) {
            this.forwardDCs.splice(i, 1);
            console.log(dc, "has been removed from forward list");
            return;
        }
    }
};