class RobotikInterConnect {

    constructor(client_addr, recv_callback = (version,src,dst,tags,data)=>{} ,url) {
        this.version = 1;
        this.src = client_addr;

        if ("WebSocket" in window) {
            this.websocket = new WebSocket(url);
            this.websocket.binaryType = "arraybuffer";

            this.websocket.onopen = ()=>{
                this.send(this.src,3,new Uint8Array(8));
            };

            this.websocket.onclose = () => {
                alert("Connection is closed...");
            };

            this.websocket.onmessage = ({data}) => {
                let view = new Uint8Array(data);

                let version = view[0];
                let src = view[1];
                let dst = view[2];
                let tags = view[3];
                let d = view.subarray(4,24);

                recv_callback(version,src,dst,tags,d); 
            };
        } else {
            // The browser doesn't support WebSocket
            alert("WebSocket NOT supported by your Browser!");
        }
    }

    /**
     * 
     * @param {ArrayBuffer} arrBuffer 
     */
    send_buffer(arrBuffer) {
        if (!arrBuffer || !arrBuffer.byteLength || arrBuffer.byteLength != 24) throw "send_buffer need a ArrayBuffer of size 24";

        this.websocket.send(arrBuffer);
    }

    /**
     * 
     * @param {number} target 0-255
     * @param {number} flags 8Bit = MSB[ disconnted (src) ,Unused,Unused,Unused,Unused,Unused,snoop,register  ]LSB
     * @param {Uint8Array | Uint16Array | Uint32Array | BigUint64Array | Float32Array | Float64Array} data length 20 Byte
     */
    send(target,flags, data) {
        let buf = this.new_package()
        buf[2] = target
        buf[3] = flags
        
        let data_view = new Uint8Array(data.buffer)
        
        buf.set(data_view,4)
        this.send_buffer( buf.buffer );
    }

    /**
     * @returns {Uint8Array} Uint8Array of length 24Byte
     */
    new_package() {
        let buf = new Uint8Array(24);

        buf[0] = this.version;
        buf[1] = this.src;

        return buf;
    }


}