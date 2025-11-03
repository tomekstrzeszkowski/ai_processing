import { useWebRtc } from '@/app/webRtcProvider';

interface LiveVideoPlayerProps {}

export const LiveVideoPlayer: React.FC<LiveVideoPlayerProps> = () => {
  const { videoRef } = useWebRtc();

  return (
    <div className="video-container">
        <video 
            style={{ width: '100%' }}
            ref={videoRef}
            autoPlay 
            playsInline
        />
    </div>
  );
}