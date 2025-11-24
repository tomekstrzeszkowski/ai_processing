import { useProtocol } from '@/app/protocolProvider';
import { formatTime } from '@/helpers/formatters';
import Slider from '@react-native-community/slider';
import Hls from 'hls.js';
import { useEffect, useRef, useState } from 'react';
import { Pressable, Text, View } from 'react-native';


interface LiveVideoPlayerProps {
  isConnected: boolean;
  stream: MediaStream|string|null;
  handleSeek?: Function;
  isLive?: boolean;
  seekMax?: number,
  seekValue?: number,
}
const hls = new Hls({
  enableWorker: true,
  lowLatencyMode: true,
});

export const LiveVideoPlayer: React.FC<LiveVideoPlayerProps> = ({
  isConnected, stream, isLive, handleSeek = () => {}, seekMax=120, seekValue=0
}) => {
    const videoRef = useRef<HTMLVideoElement>(null);
    const { p2pPlayer } = useProtocol();
    const [isShowMenu, setIsShowMenu] = useState<boolean>(false);
    
    useEffect(() => {
      if (!videoRef.current || !stream) {
        return;
      }

      if (stream instanceof MediaStream) {
        videoRef.current.srcObject = stream;
      } else {
        if (p2pPlayer === "hls") {
          if (videoRef.current.canPlayType('application/vnd.apple.mpegurl')) {
            videoRef.current.src = stream;
          } else if (Hls.isSupported()) {
            hls.loadSource(stream);
            hls.attachMedia(videoRef.current);
          }
        }
      }
      return () => {
        if (hls) {
          hls.stopLoad();
          hls.detachMedia();
          hls.destroy();
        }
        
        if (videoRef.current) {
          videoRef.current.srcObject = null;
          videoRef.current.src = '';
        }
      };
    }, [stream]);

    useEffect(function () {
      if (!isConnected) {
        hls.stopLoad();
      } else {
        hls.startLoad();
      }
      
    }, [isConnected]);

  return (
    <View style={{
        display: "flex", 
        flex: 1,
      }}
    >
      <Pressable
        onHoverIn={() => setIsShowMenu(true)}
        onHoverOut={() => setIsShowMenu(false)}
      >
      {(stream instanceof MediaStream || (stream && p2pPlayer === 'hls')) && <video 
          style={{ display: isConnected ? "flex": "none", margin:"0" }}
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
          minWidth: 300,
          minHeight: 800,
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
      {(stream instanceof MediaStream || (stream && p2pPlayer === 'hls')) && <View style={{
          display: isShowMenu ? "flex" : "none", 
          position: "absolute",
          bottom: 0,
          left: 0,
          right: 0,
          padding: 30,
          background: 'linear-gradient(transparent, #1a1a1a)',
        }}>
          {!isLive && <Slider
            style={{width: "100%", height: 40}}
            minimumValue={0}
            maximumValue={seekMax}
            value={seekValue}
            minimumTrackTintColor="#e53e3e"
            maximumTrackTintColor="#ffffff8a"
            thumbTintColor="#e53e3e"
            onSlidingComplete={(seek) => handleSeek(seek)}
          />}

          <View
            style={{
              display: 'flex',
              flexDirection: 'row',
              justifyContent: 'space-between',
              marginTop: 15,
            }}
          >
            {isLive && <View style={{ flexDirection: 'row'}}>
              <Text style={{ color: '#ff0000ff', fontSize: 12 }}>
                {isConnected ? "🔴" : "🔘"}
              </Text>
            </View>}

            {!isLive && <View style={{ flexDirection: 'row'}}>
              <Text style={{ color: 'white', fontSize: 12 }}>
                {formatTime(seekValue)}
              </Text>
              <Text style={{ color: '#fff', fontSize: 12, fontWeight: 900}}>&nbsp;/&nbsp;</Text>
              <Text style={{ color: '#b9b9b9ff', fontSize: 12 }}>
                {formatTime(seekMax)}
              </Text>
            </View>}
          </View>
        </View>}
      </Pressable>
    </View>
  );
}
