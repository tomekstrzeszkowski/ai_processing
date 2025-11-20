interface dataChannelMessageTypeToCallbackInterface {
    [key: string]: Function
}
interface dataChannelTimeoutToIdInterface {
    [key: string]: number
}


export class WebRtcOfferee{
    pc: RTCPeerConnection | null = null
    dataChannel: RTCDataChannel | undefined
    iceCandidatesGenerated: RTCIceCandidate[] = []
    iceCandidateReceivedBuffer: RTCIceCandidate[] = []
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
    }
    onIceConnectionChange: (state: string) => void = () => {}
    dataChannelMessageTypeToCallback: dataChannelMessageTypeToCallbackInterface
    dataChannelTimeoutToId: dataChannelTimeoutToIdInterface
    streamIdToStream: Map<string, MediaStream>
    constructor(onIceConnectionChange: (state: string) => void) {
        this.pc = null;
        this.onIceConnectionChange = onIceConnectionChange;
        this.dataChannelMessageTypeToCallback = {};
        this.dataChannelTimeoutToId = {};
        this.streamIdToStream = new Map();
    }
    close() {
      if (this.pc?.connectionState !== "closed") {
        this.dataChannel?.close();
        this.pc?.close();
      }
      this.iceCandidatesGenerated = [];
      this.iceCandidateReceivedBuffer = [];
    }
    isConnected() {
        return ["connected", "connecting"].includes(this.pc?.connectionState ?? "")
    }

    initializePeerConnection() {
        if (this.pc) {
            this.close();
        }
        this.pc = null;
        this.pc = new RTCPeerConnection(this.configuration);
        this.pc.addEventListener("track", (event: RTCTrackEvent) => {
            const stream = event.streams[0] || new MediaStream([event.track]);
            this.streamIdToStream.set(stream.id, stream);
        });
        this.handlePC();
        this.handleDataChannel()
    }

    handlePC(stream: MediaStream|null = null) {
        if (this.pc === null) return;
        this.pc.addEventListener("negotiationneeded", async () => {
            console.log("negotiation needed, wait for offer");
            if (this.pc?.iceConnectionState === "failed") {
                this.pc.restartIce();
            }
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
            if (this.pc?.iceConnectionState === "failed") {
                this.pc.restartIce();
            }
        });
    };
    onTrack(callback: Function) {
        const pc = this.pc;
        const streamIdToStream = this.streamIdToStream;
        function streamTrack(event: RTCTrackEvent) {
            console.log("Added video stream track!", event.streams, event);
            pc?.removeEventListener("track", streamTrack)
            const stream = event.streams[0] || new MediaStream([event.track]);
            streamIdToStream.set(stream.id, stream);
            callback(stream);
        }
        this.pc?.addEventListener("track", streamTrack);
    };
    handleDataChannel() {
        this.pc?.addEventListener("datachannel", ({channel}) => {
            this.dataChannel = channel;
            this.dataChannel.addEventListener("message", (e) => {
                console.log("message has been received from a Data Channel", e);
                if (!(e.data instanceof ArrayBuffer)) return;
                let message;
                try {
                    const jsonText = new TextDecoder().decode(e.data);
                    message = JSON.parse(jsonText);
                } catch (err) {
                    return
                }
                if (message?.type in this.dataChannelMessageTypeToCallback) {
                    this.dataChannelMessageTypeToCallback[message.type](message);
                }
            });
            this.dataChannel.addEventListener("close", (e) => {
                console.log("The close event was fired on you data channel object");
            });
            this.dataChannel.addEventListener("open", (e) => {
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
            //TODO: is it regiestered multiple times?
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
    registerOrSkipDataChannelListener(type: string, callback: Function) {
        if ("type" in this.dataChannelMessageTypeToCallback) return;
        this.dataChannelMessageTypeToCallback[type] = callback;
    };
    async waitForDataChannel() {
        return new Promise<void>((resolve, reject) => {
            if (this.dataChannel) {
                resolve();
                return;
            }
            const timeoutId = setTimeout(() => {
                reject(new Error('Data channel creation timeout'));
            }, 10000);
            this.pc?.addEventListener("datachannel", (event) => {
                clearTimeout(timeoutId);
                this.dataChannel = event.channel;
                resolve();
            });
        });
    };
    async fetchVideoList(startDate: string, endDate: string) {
        return new Promise<Array<object>>((resolve, reject) => {
            if ("videoList" in this.dataChannelTimeoutToId) {
                clearTimeout(this.dataChannelTimeoutToId["videoList"]);
            }
            this.registerOrSkipDataChannelListener("videoList", function (data: any) {
                resolve(data.videoList);
            });
            this.dataChannel?.send(JSON.stringify({type: "videoList", startDate, endDate}));
        });
    };
    async fetchVideo(videoName: string) {
        try {
            await this.waitForDataChannel();
        } catch(err) {
            console.error(err);
            return
        }
        const dataChannel = this.dataChannel;
        const pc = this.pc;
        this.pc?.addEventListener("track", (e) => {
        });
        this.registerOrSkipDataChannelListener("offer", async function (offer: any) {
            console.log("got new offer");
            if (!pc || !dataChannel) {
                throw new Error(`Can not re-negotiate, pc or data channel is empty ${pc} ${dataChannel}`)
            }
            await pc.setRemoteDescription(offer);
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            dataChannel.send(JSON.stringify(answer));
        });
        this.dataChannel?.send(JSON.stringify({type: "video", videoName}));
        const streamIdToStream = this.streamIdToStream;
        return new Promise<MediaStream>((resolve, reject) => {
            function handleTrack(event: RTCTrackEvent) {
                console.log("Added video track", event.streams, event)
                pc?.removeEventListener("track", handleTrack);
                const stream = event.streams[0] || new MediaStream([event.track]);
                streamIdToStream.set(stream.id, stream);
                resolve(stream);
            }
            this.pc?.addEventListener("track", handleTrack);
        });
    };
}
