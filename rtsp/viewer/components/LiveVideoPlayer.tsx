import { useEffect, useRef } from 'react';
import { Text, View } from 'react-native';


interface LiveVideoPlayerProps {
  isConnected: boolean;
  stream?: MediaStream|null;
  streamUrl?: string;
}

export const LiveVideoPlayer: React.FC<LiveVideoPlayerProps> = ({isConnected, stream, streamUrl}) => {
    const videoRef = useRef<HTMLVideoElement>(null);
    useEffect(() => {
      if (stream && videoRef.current)
      videoRef.current.srcObject = stream;
    }, [stream]);
    // useEffect(() => {
    //   if (streamUrl && videoRef.current) {
    //     videoRef.current.src = streamUrl;
    //   }
    // }, [streamUrl]);
  return (
    <View style={{
      display: "flex", 
      flex: 1,
      flexDirection: "column" ,
      justifyContent: "center",
      alignItems: "center"
    }}>
      {stream && <video 
          controls
          style={{ display: isConnected ? "flex": "none", margin:"auto" }}
          ref={videoRef}
          autoPlay 
          playsInline
      
      />}
      {streamUrl && <img 
          src={streamUrl}
          style={{ display: isConnected ? "block": "none", maxWidth: "100%", height: "auto" }}
          alt="Live stream"
      />}
      {!isConnected && <View style={{display: "flex", alignSelf: "center"}}>
        <Text style={{ color: "#b9b9b9ff" }}>Connect to view stream</Text>
      </View>}
    </View>
  );
}