import { useWebRtc } from '@/app/webRtcProvider';

interface LiveVideoPlayerProps {}

export const LiveVideoPlayer: React.FC<LiveVideoPlayerProps> = () => {
  const { videoRef } = useWebRtc();

  return (
    <div className="video-container">
        <video 
            style={{ width: '100%', display: "flex", margin:"auto" }}
            ref={videoRef}
            autoPlay 
            playsInline
        />
    </div>
  );
}