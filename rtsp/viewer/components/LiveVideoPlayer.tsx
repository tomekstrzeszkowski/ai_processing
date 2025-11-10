import { useWebRtc } from '@/app/webRtcProvider';

interface LiveVideoPlayerProps {
  isConnected: boolean;
}

export const LiveVideoPlayer: React.FC<LiveVideoPlayerProps> = ({isConnected}) => {
  const { videoRef } = useWebRtc();

  return (
    <div style={{
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
      {!isConnected && <div style={{display: "flex", alignSelf: "center"}}>Connect to view stream</div>}
    </div>
  );
}