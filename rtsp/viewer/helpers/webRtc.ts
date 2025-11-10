export class WebRtcOfferee{
    pc: RTCPeerConnection | null = null;
    dataChannel: RTCDataChannel | undefined;
    iceCandidatesGenerated: RTCIceCandidate[] = [];
    iceCandidateReceivedBuffer: RTCIceCandidate[] = [];
    configuration: RTCConfiguration = {
        iceServers: [
            // STUN servers
            {
                urls: [
                    "stun:stun.l.google.com:19302",
                    "stun:stun2.l.google.com:19302",
                    "stun:stun3.l.google.com:19302",
                    'stun:stun.1und1.de:3478',
                    'stun:stun.avigora.com:3478',
                    'stun:stun.avigora.fr:3478',
                ]
            },
            // TURN server - Twilio (your existing one, now properly formatted)
            {
                urls: "turn:global.turn.twilio.com:3478?transport=udp",
                username: "dc2d2894d5a9023620c467b0e71cfa6a35457e6679785ed6ae9856fe5bdfa269",
                credential: "tE2DajzSJwnsSbc123"
            },
            // Additional free TURN servers as backup
            {
                urls: ['turn:openrelay.metered.ca:80', 'turn:openrelay.metered.ca:443'],
                username: 'openrelayproject',
                credential: 'openrelayproject'
            },
            {
                urls: 'turn:openrelay.metered.ca:443?transport=tcp',
                username: 'openrelayproject',
                credential: 'openrelayproject'
            }
        ]
    };
    onIceConnectionChange: (state: string) => void = () => {};
    onTrack: (stream: MediaStream) => void = () => {};
    constructor(onIceConnectionChange: (state: string) => void, onTrack: (stream: MediaStream) => void) {
        this.pc = null;
        this.onIceConnectionChange = onIceConnectionChange;
        this.onTrack = onTrack;
    }
    close() {
      if (this.pc?.connectionState !== "closed") {
        this.dataChannel?.close();
        this.pc?.close();
      }
      this.iceCandidatesGenerated = [];
      this.iceCandidateReceivedBuffer = [];
    }  

    initializePeerConnection() {
        if (this.pc) {
            this.close();
        }
        this.pc = null;
        this.pc = new RTCPeerConnection(this.configuration);
        this.handlePC();
        this.handleDataChannel()
    }

    handlePC(stream: MediaStream|null = null) {
        if (this.pc === null) return;
        this.pc.addEventListener("negotiationneeded", async () => {
            console.log("negotiation needed");
            // const offer = await pc.createOffer();
            // await this.pc.setLocalDescription(offer);
        });
        if (stream) {
            stream.getTracks().forEach(track => {
                console.log("adding track", track);
                this.pc?.addTrack(track);
            }); 
        }
        this.pc?.addEventListener("icecandidate", ({candidate}) => {
            if(!candidate) return;
            this.iceCandidatesGenerated.push(candidate);
        });
        this.pc?.addEventListener("iceconnectionstatechange", () => {
            console.log(`iceconnectionstatechange ${this.pc?.iceConnectionState}`);
            this.onIceConnectionChange(this.pc?.iceConnectionState??"");
            if (this.pc?.iceConnectionState === "disconnected") {
                this.close();
            }
        });
        this.pc?.addEventListener("track", (event) => {
          const stream = event.streams[0] || new MediaStream([event.track]);
          this.onTrack(stream);
        });
    };
    handleDataChannel() {
        this.pc?.addEventListener("datachannel", (e) => {
            this.dataChannel = e.channel;
            this.dataChannel?.addEventListener("message", (e) => {
                console.log("message has been received from a Data Channel", e);
            });
            this.dataChannel?.addEventListener("close", (e) => {
                console.log("The close event was fired on you data channel object");
            });
            this.dataChannel?.addEventListener("open", (e) => {
                console.log("Data Channel has been opened. You are now ready to send/receive messsages over your Data Channel");
            });
        });
    };
    handleIceCandidates({ice: candidates}: {ice: RTCIceCandidate[]}) {
        if (this.pc === null) return;
        if (this.pc.remoteDescription) {
            candidates.forEach(async candidate => {
                try{
                    await this.pc?.addIceCandidate(candidate);
                } catch (error) {
                    console.error("Error adding ICE candidate:", error);
                }
            });
        } else {
            candidates.forEach(candidate => {
                this.iceCandidateReceivedBuffer.push(candidate);
            });
        }
    };
    waitForCandidates(minCandidates = 5, timeout = 5000) {
        return new Promise<void>((resolve) => {
            if (this.pc === null) return;
            let timeoutId: number | null = null;
            let resolved = false;

            const finish = () => {
                if (resolved) return;
                resolved = true;
                if (timeoutId) clearTimeout(timeoutId);
                resolve();
            };
            timeoutId = setTimeout(finish, timeout);

            this.pc.addEventListener("icecandidate", ({candidate}) => {
                if (!candidate) {
                    finish();
                    return;
                };
                if (this.iceCandidatesGenerated.length >= minCandidates) {
                    finish();
                }
            });
        });
    };
}
