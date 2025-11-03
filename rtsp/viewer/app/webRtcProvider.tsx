import { SignalingMessage } from '@/helpers/message';
import { WebSocketSignalingClient } from '@/helpers/signaling';
import { WebRtcOfferee } from '@/helpers/webRtc';
import { createContext, useContext, useEffect, useRef, useState } from 'react';

type WebRtcContextType = {
  remoteStream: MediaStream | null;
  videoRef: React.RefObject<HTMLVideoElement>;
};

const WebRtcContext = createContext<WebRtcContextType | null>(null);

export const useWebRtc = () => {
  const context = useContext(WebRtcContext);
  if (!context) {
    throw new Error('useWebRtc must be used within a WebRtcProvider');
  }
  return context;
};

export const WebRtcProvider = ({ children }: { children: React.ReactNode }) => {
  const [remoteStream, setRemoteStream] = useState<MediaStream | null>(null);
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    const host = document.location.hostname || 'localhost';
    const signalingServerUrl = `ws://${host}:7070/ws`;
    const signalingClient = new WebSocketSignalingClient(52, signalingServerUrl);
    signalingClient.connect();
    const offeree = new WebRtcOfferee();
    signalingClient.onIce(async candidates => {
        await offeree.handleIceCandidates({ice: candidates.ice as RTCIceCandidate[]});
    }); 
    signalingClient.onOffer(async (offer: SignalingMessage) => {
        offeree.handlePC();
        offeree.pc.addEventListener("track", (event) => {
          const stream = event.streams[0] || new MediaStream([event.track]);
          setRemoteStream(stream);
        });
        offeree.handleDataChannel();
        const sdp = String(offer.sdp);
        await offeree.pc.setRemoteDescription({sdp, type: 'offer'});
        const answer = await offeree.pc.createAnswer(); 
        console.log("set local");
        await offeree.pc.setLocalDescription(answer);
        console.log("send answer");
        signalingClient.sendAnswer(answer);
        setTimeout(() => {
            console.log("icCandidates Generated", offeree.iceCandidatesGenerated)
            signalingClient.sendIceCandidates(offeree.iceCandidatesGenerated);
        }, 1000);
    });
    return () => {
      // signalingClient.disconnect();
      // offeree.pc.close();
    }
  });
  // Update video element when stream changes
  useEffect(() => {
    if (videoRef.current && remoteStream) {
      videoRef.current.srcObject = remoteStream;
    }
  }, [remoteStream]);

  const value = {
    remoteStream,
    videoRef,
  };
  return (
    <WebRtcContext.Provider value={value}>
      {children}
    </WebRtcContext.Provider>
  );
};

export default WebRtcProvider;