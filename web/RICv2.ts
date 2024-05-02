export default class RobotikInterConnect {
    websocket: WebSocket;
    version: number;
    src: number;
    decoder: TextDecoder;
    encoder: TextEncoder;
    clientSet: Set<number>;

    constructor(client_addr: number, recv_callback = (version: number,src: number,dst: number,flags: number,data: string)=>{} ,url: URL) {
        this.version = 1;
        this.src = client_addr;

        this.decoder = new TextDecoder();
        this.encoder = new TextEncoder();
        this.clientSet = new Set();
        
        this.websocket = new WebSocket(url.toString());
        this.websocket.binaryType = "arraybuffer";

        this.websocket.onopen = ()=>{
            this.send_uint8arr(this.src,3,new Uint8Array(20));

            this.send_uint8arr(this.src,4,new Uint8Array(20)); //ask for list!
        };

        this.websocket.onclose = () => {
            alert("Connection is closed...");
        };

        this.websocket.onmessage = ({data}) => {
            let view = new Uint8Array(data);

            let version = view[0];
            let src = view[1];
            let dst = view[2];
            let flags = view[3];
            let d = view.subarray(4,24);

            if ((flags & 4) != 0) {
                for (let byte of d) {
                    this.clientSet.add(byte)
                }
            }

            if ((flags & 1) != 0) {
                this.clientSet.add(src);
            }  
            
            if ((flags & 128) != 0) {
                this.clientSet.delete(src);
            }

            recv_callback(version,src,dst,flags,this.decoder.decode(d)); 
        };
        
    }

    /**
     * 
     * @param {ArrayBuffer} arrBuffer 
     */
    send_buffer(arrBuffer: ArrayBufferLike) {
        if (!arrBuffer || !arrBuffer.byteLength || arrBuffer.byteLength != 24) throw "send_buffer need a ArrayBuffer of size 24";

        this.websocket.send(arrBuffer);
    }

    /**
     * 
     * @param {number} target 0-255
     * @param {number} flags 8Bit = MSB[ disconnted (src) ,Unused,Unused,Unused,Unused,Unused,snoop,register  ]LSB
     * @param {Uint8Array} data length 20 Byte
     */
    send_uint8arr(target: number,flags: number, data: Uint8Array) {
        let buf = new Uint8Array(24);
        
        buf[0] = this.version;
        buf[1] = this.src;

        buf[2] = target
        buf[3] = flags
        
        let data_view = new Uint8Array(data.buffer)
        
        buf.set(data_view,4)
        this.send_buffer( buf.buffer );
    }

    send(target: number, text: string) {
        let data = this.encoder.encode(text).subarray(0,24);

        this.send_uint8arr(target, 0,data);
    }

    close() {
        this.websocket.close();
    }

}