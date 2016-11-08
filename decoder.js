var decoder = window.decoder = function decoder ( url, opts ) {
    window.jsmpeg.call(this, url, opts);
}

decoder.prototype = new jsmpeg();
decoder.prototype.constructor = decoder;

decoder.prototype.decodeSocketHeader = function( data ) {
	// Custom header sent to all newly connected clients when streaming
	// over websockets:
	// struct { char magic[4] = 'jsmp'; unsigned short width, height; };
    this.width = 640;
    this.height = 480;
    this.initBuffers();
};

decoder.prototype.receiveSocketMessage = function( event ) {
    var messageData = new Uint8Array(event.data);
    event.data = messageData.slice(11)
    jsmpeg.prototype.receiveSocketMessage.call(this, event);
}