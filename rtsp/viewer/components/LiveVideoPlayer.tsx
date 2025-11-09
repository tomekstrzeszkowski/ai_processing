import { useWebRtc } from '@/app/webRtcProvider';

interface LiveVideoPlayerProps {
  isConnected: boolean;
}

export const LiveVideoPlayer: React.FC<LiveVideoPlayerProps> = ({isConnected}) => {
  const { videoRef } = useWebRtc();

  return (
    <div className="video-container">
      <video 
          controls
          style={{ display: isConnected ? "flex": "none", margin:"auto" }}
          ref={videoRef}
          autoPlay 
          playsInline
      />
      {!isConnected && <div style={{display: "flex"}}>No video.</div>}
    </div>
  );
}