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
  const handlePlayRef = useRef<EventListenerOrEventListenerObject>(() => {});
  const handlePauseRef = useRef<EventListenerOrEventListenerObject>(() => {});

  useEffect(() => {
    const host = document.location.hostname || 'localhost';
    const signalingServerUrl = `ws://${host}:7070/ws`;
    const signalingClient = new WebSocketSignalingClient(52, signalingServerUrl);
    const offeree = new WebRtcOfferee();
    offeree.handlePC();
    offeree.pc.addEventListener("track", (event) => {
      const stream = event.streams[0] || new MediaStream([event.track]);
      setRemoteStream(stream);
    });
    handlePlayRef.current = async () => {
        await signalingClient.connect();
        signalingClient.onIce(async candidates => {
          try{
            await offeree.handleIceCandidates({ice: candidates.ice as RTCIceCandidate[]});
          } catch (error) {
            console.error("Error handling ICE candidates:", error);
          }
        }); 
        signalingClient.onOffer(async (offer: SignalingMessage) => {
          if (offeree.pc.connectionState === "closed" || offeree.pc.signalingState === "closed") {
            offeree.initializePeerConnection();
          }
          const sdp = String(offer.sdp);
          console.log("signaling state", offeree.pc.signalingState);
          await offeree.pc.setRemoteDescription({sdp, type: 'offer'});
          const answer = await offeree.pc.createAnswer();
          await offeree.pc.setLocalDescription(answer);
          signalingClient.sendAnswer(answer);
          await offeree.waitForCandidates();
          signalingClient.sendIceCandidates(offeree.iceCandidatesGenerated);
        });
    };
    handlePauseRef.current = () => {
      signalingClient.disconnect();
      offeree.close();
    };
    offeree.handleDataChannel();
    videoRef.current?.addEventListener("play", handlePlayRef.current);
    videoRef.current?.addEventListener("pause", handlePauseRef.current);
    return () => {
      console.log("Cleaning up WebRTC connections");
      videoRef.current?.removeEventListener("play", handlePlayRef.current);
      videoRef.current?.removeEventListener("pause", handlePauseRef.current);
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
    handlePlayRef,
    handlePauseRef,
  };
  return (
    <WebRtcContext.Provider value={value}>
      {children}
    </WebRtcContext.Provider>
  );
};

export default WebRtcProvider;