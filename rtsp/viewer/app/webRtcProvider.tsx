import { useProtocol } from '@/app/protocolProvider';
import { useToast } from '@/app/toastProvider';
import { SignalingMessage } from '@/helpers/message';
import { WebSocketSignalingClient } from '@/helpers/signaling';
import { WebRtcOfferee } from '@/helpers/webRtc';
import { createContext, useContext, useEffect, useRef, useState } from 'react';


type WebRtcContextType = {
  remoteStream: MediaStream | null;
  videoRef: React.RefObject<HTMLVideoElement | null>;
  handlePlayRef: React.RefObject<Function>;
  handleStopRef: React.RefObject<Function>;
  offereeRef: React.RefObject<WebRtcOfferee>;
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
  const {
    setIsConnected, 
    setIsConnecting, 
    setLastFrameTime, 
    isConnected,
    isWebRtc,
  } = useProtocol();
  const { showAlert } = useToast();
  const host = document.location.hostname || 'localhost';
  const signalingServerUrl = `ws://${host}:7070/ws`;
  const signalingClient = new WebSocketSignalingClient(11, signalingServerUrl);
  const videoRef = useRef<HTMLVideoElement>(null);
  const frameInterval = useRef<number | null>(null);
  const handlePlayRef = useRef<Function>(() => {});
  const handleStopRef = useRef<Function>(() => {});
  const offereeRef = useRef<WebRtcOfferee>(new WebRtcOfferee((state) => {
    setIsConnecting(false);
    setIsConnected(state === "connected");
  }, (stream) => {
    setRemoteStream(stream);
  }));

  useEffect(() => {
    const handlePlay = () => handlePlayRef.current();
    const handlePause = () => () => handleStopRef.current();
    videoRef.current?.addEventListener("play", handlePlay);
    videoRef.current?.addEventListener("pause", handlePause);

    handlePlayRef.current = async () => {
        const offeree = offereeRef.current;
        if (!isWebRtc || ["connected", "connecting"].includes(offeree.pc?.connectionState ?? "")) {
            return;
        }
        try {
          await signalingClient.connect();
          signalingClient.ws?.send(JSON.stringify({"type": "start"}))
        } catch (err) {
          setIsConnecting(false);
          console.error(err);
          showAlert("Can not connect to signaling server. Try again later")
          return;
        }
        offeree.initializePeerConnection();
        signalingClient.onIce(async candidates => {
          try{
            await offeree.handleIceCandidates({ice: candidates.ice as RTCIceCandidate[]});
          } catch (error) {
            console.error("Error handling ICE candidates:", error, candidates);
          }
        }); 
        signalingClient.onOffer(async (offer: SignalingMessage) => {
          console.log('onOffer')
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
          if (offeree.pc?.signalingState === "stable") return;
          const answer = await offeree.pc?.createAnswer();
          try {
            await offeree.pc?.setLocalDescription(answer);
          } catch (error) {
            console.error(error)
            return;
          }
          if(!answer) return;
          signalingClient.sendAnswer(answer);
          await offeree.waitForCandidates();
          console.log("icCandidates Generated", offeree.iceCandidatesGenerated)
          signalingClient.sendIceCandidates(offeree.iceCandidatesGenerated);
        });
    };
    handleStopRef.current = () => {
      if (offereeRef.current.dataChannel) {
        signalingClient.ws?.send(JSON.stringify({type: "disconnected"}))
        offereeRef.current.dataChannel.send(JSON.stringify({type: "close"}))
      }
      signalingClient.disconnect();
      offereeRef.current.close();
      setIsConnected(false);
      videoRef.current?.removeEventListener("play", handlePlay);
      videoRef.current?.removeEventListener("pause", handlePause);
    };
    return () => {
      console.log("Cleaning up WebRTC connections");
      //offereeRef.current.close();
      handleStopRef.current();
    }
  }, []);
  useEffect(() => {
    if (videoRef.current && remoteStream) {
      videoRef.current.srcObject = remoteStream;
      setLastFrameTime(new Date());
    }
  }, [remoteStream]);

  useEffect(() => {
    if (isConnected) {
      if (!frameInterval.current) {
        frameInterval.current = setInterval(() => {
          setLastFrameTime(new Date());
        }, 1000);
      }
    } else {
      if (frameInterval.current) {
        clearInterval(frameInterval.current);
      }
      handleStopRef.current();
    }
  }, [isConnected]);
  return (
    <WebRtcContext.Provider value={{
      remoteStream,
      videoRef,
      handlePlayRef,
      handleStopRef,
      offereeRef,
    }}>
      {children}
    </WebRtcContext.Provider>
  );
};

export default WebRtcProvider;