import { useProtocol } from '@/app/protocolProvider';
import Hls from 'hls.js';
import { useEffect, useRef } from 'react';
import { Text, View } from 'react-native';


interface LiveVideoPlayerProps {
  isConnected: boolean;
  stream: MediaStream|string|null;
}

export const LiveVideoPlayer: React.FC<LiveVideoPlayerProps> = ({isConnected, stream}) => {
    const videoRef = useRef<HTMLVideoElement>(null);
    const { p2pPlayer } = useProtocol();
    useEffect(() => {
      if (!videoRef.current || !stream) return;
      if (stream instanceof MediaStream) {
        videoRef.current.srcObject = stream;
      } else {
        if (p2pPlayer === "hls") {
          if (videoRef.current.canPlayType('application/vnd.apple.mpegurl')) {
            videoRef.current.src = stream;
          } else if (Hls.isSupported()) {
            const hls = new Hls({
              enableWorker: true,
              lowLatencyMode: true,
            });
            hls.loadSource(stream);
            hls.attachMedia(videoRef.current);
          }
        }
      }
    }, [stream]);

  return (
    <View style={{
      display: "flex", 
      flex: 1,
      flexDirection: "column" ,
      justifyContent: "center",
      alignItems: "center"
    }}>
      {(stream instanceof MediaStream || (stream && p2pPlayer === 'hls')) && <video 
          controls
          style={{ display: isConnected ? "flex": "none", margin:"auto" }}
          ref={videoRef}
          autoPlay 
          playsInline
      
      />}
      {stream && p2pPlayer === 'image' && <img 
          src={stream as string}
          style={{ display: isConnected ? "block": "none", maxWidth: "100%", height: "auto" }}
          alt="Live stream"
      />}
      {(!stream || !isConnected) && (
        <View style={{
          width: '100%',
          height: "100%",
          backgroundColor: '#1a1a1a',
          borderRadius: 8,
          overflow: 'hidden',
          justifyContent: 'center',
          alignItems: 'center',
        }}>
          <View style={{
            position: 'absolute',
            top: 0,
            left: '-100%',
            height: '100%',
            width: '100%',
            background: 'linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.06), transparent)',
            animation: 'shimmer 1.5s infinite'
          }} />
          <style>{`
            @keyframes shimmer {
              0% { left: -100%; }
              100% { left: 100%; }
            }
          `}</style>
          <View style={{display: "flex", alignSelf: "center"}}>
            <Text style={{ color: "#b9b9b9ff" }}>Waiting for stream...</Text>
          </View>
        </View>
      )}
    </View>
  );
}