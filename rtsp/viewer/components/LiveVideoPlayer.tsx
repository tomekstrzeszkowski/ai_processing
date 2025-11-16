import { useEffect, useRef } from 'react';
import { Text, View } from 'react-native';


interface LiveVideoPlayerProps {
  isConnected: boolean;
  stream: MediaStream | null,
}

export const LiveVideoPlayer: React.FC<LiveVideoPlayerProps> = ({isConnected, stream}) => {
    const videoRef = useRef<HTMLVideoElement>(null);
    useEffect(() => {
      if (stream && videoRef.current)
      videoRef.current.srcObject = stream;
    }, [stream]);

  return (
    <View style={{
      display: "flex", 
      flex: 1,
      flexDirection: "column" ,
      justifyContent: "center",
      alignItems: "center"
    }}>
      <video 
          controls
          style={{ display: isConnected ? "flex": "none", margin:"auto" }}
          ref={videoRef}
          autoPlay 
          playsInline
      />
      {!isConnected && <View style={{display: "flex", alignSelf: "center"}}><Text>Connect to view stream</Text></View>}
    </View>
  );
}