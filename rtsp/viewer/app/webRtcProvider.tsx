import { SignalingMessage } from '@/helpers/message';
import { WebSocketSignalingClient } from '@/helpers/signaling';
import { WebRtcOfferee } from '@/helpers/webRtc';
import { createContext, useContext, useEffect, useRef, useState } from 'react';

type WebRtcContextType = {
  remoteStream: MediaStream | null;
  videoRef: React.RefObject<HTMLVideoElement | null>;
  isConnected: boolean;
  handlePlayRef: React.RefObject<EventListenerOrEventListenerObject>;
  handleStopRef: React.RefObject<EventListenerOrEventListenerObject>;
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
  const handleStopRef = useRef<EventListenerOrEventListenerObject>(() => {});
  const offereeRef = useRef<WebRtcOfferee>(new WebRtcOfferee((state) => {
    setIsConnected(state === "connected");
  }, (stream) => {
    setRemoteStream(stream);
  }));
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    const host = document.location.hostname || 'localhost';
    const signalingServerUrl = `ws://${host}:7070/ws`;
    const signalingClient = new WebSocketSignalingClient(52, signalingServerUrl);
    videoRef.current?.addEventListener("play", handlePlayRef.current);
    videoRef.current?.addEventListener("pause", handleStopRef.current);

    handlePlayRef.current = async () => {
        const offeree = offereeRef.current;
        await signalingClient.connect();
        offeree.initializePeerConnection();
        signalingClient.onIce(async candidates => {
          try{
            await offeree.handleIceCandidates({ice: candidates.ice as RTCIceCandidate[]});
          } catch (error) {
            console.error("Error handling ICE candidates:", error);
          }
        }); 
        signalingClient.onOffer(async (offer: SignalingMessage) => {
          if (offeree.pc?.connectionState === "closed" || offeree.pc?.signalingState === "closed") {
            //can be closed by other peer
            offeree.initializePeerConnection();
            return;
          } else if (offeree.pc?.connectionState === "connected") {
            return;
          }
          const sdp = String(offer.sdp);
          console.log("signaling state", offeree.pc?.signalingState);
          await offeree.pc?.setRemoteDescription({sdp, type: 'offer'});
          const answer = await offeree.pc?.createAnswer();
          await offeree.pc?.setLocalDescription(answer);
          if(!answer) return;
          signalingClient.sendAnswer(answer);
          await offeree.waitForCandidates();
          signalingClient.sendIceCandidates(offeree.iceCandidatesGenerated);
        });
    };
    handleStopRef.current = () => {
      signalingClient.disconnect();
      offereeRef.current.close();
      setIsConnected(false);
      videoRef.current?.removeEventListener("play", handlePlayRef.current);
      videoRef.current?.removeEventListener("pause", handleStopRef.current);
    };
    return () => {
      console.log("Cleaning up WebRTC connections");
    }
  }, []);
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
    handleStopRef,
    isConnected,
  };
  return (
    <WebRtcContext.Provider value={value}>
      {children}
    </WebRtcContext.Provider>
  );
};

export default WebRtcProvider;