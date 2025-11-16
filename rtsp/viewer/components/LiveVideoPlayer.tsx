import { useWebRtc } from '@/app/webRtcProvider';
import { Text, View } from 'react-native';


interface LiveVideoPlayerProps {
  isConnected: boolean;
}

export const LiveVideoPlayer: React.FC<LiveVideoPlayerProps> = ({isConnected}) => {
  const { videoRef } = useWebRtc();

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