const sendButton = document.getElementById("send");
const myVideoElement = document.getElementById("myVideo");
const receiveButton = document.getElementById("receive");

let client;

async function createPC(iceCandidatesGenerated, stream) {
    const pc = new RTCPeerConnection({
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
    });
    window.pc = pc;

    pc.addEventListener("negotiationneeded", async () => {
        // const offer = await pc.createOffer();
        // await pc.setLocalDescription(offer);
    });
    if (stream) {
        stream.getTracks().forEach(track => {
            console.log("adding track", track);
            pc.addTrack(track);
        }); 
    }
    pc.addEventListener("track", (event) => {
        console.log("track event", event);
        myVideoElement.srcObject = event.streams[0] || new MediaStream([event.track]);
    });

    let stats = await pc.getStats();
    stats.forEach(report => {
        //console.log("stat", report);
    })
    pc.addEventListener("icecandidate", ({candidate}) => {
        if(!candidate) return;
        iceCandidatesGenerated.push(candidate);
    });
    pc.addEventListener("iceconnectionstatechange", () => {
        console.log(`iceconnectionstatechange ${pc.iceConnectionState}`);
        if (pc.iceConnectionState === "disconnected" && pc) {
            pc.close();
            pc = null;
        }
    });
    return pc;
}

function createDataChannel(isOfferor, dataChannel) {
    if (isOfferor) {
        dataChannel = pc.createDataChannel("learing-webrtc", {
            ordered: false, 
            maxRetransmits: 0
        });
        registerDataChannelEventListeners(dataChannel);
    } else {
        pc.addEventListener("datachannel", (e) => {
            console.log("the ondatachannel event was emmited to PEER2")
            dataChannel = e.channel;
            registerDataChannelEventListeners(dataChannel);
            window.dc = dataChannel;
        });
    }
    window.dc = dataChannel
};

function registerDataChannelEventListeners(dataChannel) {
    dataChannel.addEventListener("message", (e) => {
        console.log("message has been received from a Data Channel", e);
    });
    dataChannel.addEventListener("close", (e) => {
        console.log("The close event was fired on you data channel object");
    });
    dataChannel.addEventListener("open", (e) => {
        console.log("Data Channel has been opened. You are now ready to send/receive messsages over your Data Channel");
    });
};

async function handleIceCandidates(pc, {ice: candidates}, iceCandidateReceivedBuffer) {
    console.log("handleIceCandidates", pc.remoteDescription ? 'add' : 'buffer')
    if (pc.remoteDescription) {
        candidates.forEach(async candidate => {
            await pc.addIceCandidate(candidate);
        })
    } else {
        candidates.forEach(candidate => {
            iceCandidateReceivedBuffer.push(candidate);
        })
    }
}

sendButton.addEventListener("click", async () => {
    const userId = 51;
    client = new WebSocketSignalingClient(userId);
    await client.connect();
    const iceCandidateReceivedBuffer = [];
    client.onIce(async candidates => {
        await handleIceCandidates(pc, candidates, iceCandidateReceivedBuffer);
        await pc.setRemoteDescription(receivedAnswer);
    });
    client.onAnswer(async answer => {
        client.sendIceCandidates(iceCandidatesGenerated);
        receivedAnswer = answer;
    });
    let receivedAnswer;
    let localStream = await navigator.mediaDevices.getUserMedia({
        video: { deviceId: true },
        audio: true,
        // video: { deviceId: {exact: "<id>"}}
    });
    const iceCandidatesGenerated = [];
    myVideoElement.srcObject = localStream;
    const pc = await createPC(iceCandidatesGenerated, localStream);
    let dataChannel = null;
    createDataChannel(true, dataChannel);
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    client.sendOffer(offer);
});
receiveButton.addEventListener("click", async () => {
    let pc = null;
    const userId = 52;
    client = new WebSocketSignalingClient(userId);
    await client.connect();
    const iceCandidateReceivedBuffer = [];
    const iceCandidatesGenerated = [];

    client.onIce(async candidates => {
        if (!pc) return;
        await handleIceCandidates(pc, candidates, iceCandidateReceivedBuffer);
    }); 
    client.onOffer(async offer => {
        console.log('I have offer:', offer);
        pc = await createPC(iceCandidatesGenerated);
        const dataChannel = null;
        createDataChannel(false, dataChannel);
        await pc.setRemoteDescription(offer);
        const answer = await pc.createAnswer(); 
        await pc.setLocalDescription(answer);
        client.sendAnswer(answer);
        setTimeout(() => {
            console.log("icCandidates Generated", iceCandidatesGenerated)
            client.sendIceCandidates(iceCandidatesGenerated);
        }, 1000);
    });
});
