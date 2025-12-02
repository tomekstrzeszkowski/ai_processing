import { useProtocol } from "@/app/protocolProvider";
import { VideoLoader } from "@/components/ui/VideoLoader";
import { formatTime } from "@/helpers/formatters";
import Slider from "@react-native-community/slider";
import Hls from "hls.js";
import { useEffect, useRef, useState } from "react";
import { Pressable, Text, TouchableOpacity, View } from "react-native";


interface LiveVideoPlayerProps {
  isConnected: boolean;
  stream: MediaStream | string | null;
  isLive?: boolean;
  seekMax?: number;
  seekValue?: number;
  isPlaying?: boolean;
  isLoop?: boolean;
  handleSeek?: Function;
  handlePause?: Function;
  handlePlay?: Function;
  handleLoop?: Function;
  handleFrame?: Function;
  onLoadedMetadata?: Function;
  onTimeUpdate?: Function;
}
const hls = new Hls({
  enableWorker: true,
  lowLatencyMode: true,
});

export const LiveVideoPlayer: React.FC<LiveVideoPlayerProps> = ({
  isConnected,
  stream,
  isLive,
  seekMax = 120,
  seekValue = 0,
  isPlaying = true,
  isLoop = false,
  handleSeek = (video: HTMLVideoElement, seek: number) => {},
  handlePause = (video: HTMLVideoElement) => {},
  handlePlay = (video: HTMLVideoElement) => {},
  handleLoop = () => {},
  handleFrame = (isForward: boolean) => {},
  onLoadedMetadata = () => {},
  onTimeUpdate = () => {},
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
        if (videoRef.current.canPlayType("application/vnd.apple.mpegurl") || !isLive) {
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
        videoRef.current.src = "";
      }
    };
  }, [stream]);

  useEffect(
    function () {
      if (!isConnected) {
        hls.stopLoad();
      } else {
        hls.startLoad();
      }
    },
    [isConnected],
  );

  function play() {
    handlePlay(videoRef.current);
  }

  function pause() {
    handlePause(videoRef.current);
  }

  return (
    <View
      style={{
        display: "flex",
        flex: 1,
      }}
    >
      <Pressable
        onHoverIn={() => setIsShowMenu(true)}
        onHoverOut={() => setIsShowMenu(false)}
      >
        {(stream instanceof MediaStream || (stream && p2pPlayer === "hls")) && (
          <video
            style={{ display: isConnected ? "flex" : "none", margin: "0" }}
            ref={videoRef}
            onLoadedMetadata={onLoadedMetadata}
            onTimeUpdate={onTimeUpdate}
            autoPlay
            playsInline
          />
        )}
        {stream && p2pPlayer === "image" && (
          <img
            src={stream as string}
            style={{
              display: isConnected ? "block" : "none",
              maxWidth: "100%",
              height: "auto",
            }}
            alt="Live stream"
          />
        )}
        {(!stream || !isConnected) && (<VideoLoader />)}
        {(stream instanceof MediaStream || (stream && p2pPlayer === "hls")) && (
          <View
            style={{
              display: isShowMenu ? "flex" : "none",
              position: "absolute",
              bottom: 0,
              left: 0,
              right: 0,
              padding: 10,
              paddingTop: 0,
              paddingBottom: 20,
              background: "linear-gradient(transparent, #1a1a1a)",
              cursor: "default",
              transitionDuration: "0.8s",
              transitionTimingFunction: "linear",
              transitionProperty: "opacity",
            }}
          >
            {!isLive && (
              <Slider
                style={{ width: "100%", height: 20, cursor: "pointer" }}
                minimumValue={0}
                maximumValue={seekMax}
                value={seekValue}
                minimumTrackTintColor="#e53e3e"
                maximumTrackTintColor="#ffffff8a"
                thumbTintColor="#e53e3e"
                onSlidingComplete={(seek) => handleSeek(videoRef, seek)}
              />
            )}

            <View
              style={{
                display: "flex",
                flexDirection: "row",
                marginTop: 15,
              }}
            >
              <View
                style={{
                  flexDirection: "row",
                  flex: 1,
                  alignItems: "center",
                }}
              >
                <TouchableOpacity
                  onPress={isPlaying ? pause : play}
                  style={{
                    padding: 8,
                  }}
                >
                  <Text
                    style={{
                      color: "white",
                      fontSize: 22,
                      transitionDuration: "0.8s",
                      transitionTimingFunction: "linear",
                      transitionProperty: "opacity",
                      opacity: 1,
                    }}
                  >
                    {isPlaying ? "⏸" : "▶︎"}
                  </Text>
                </TouchableOpacity>
                {isLive && (
                  <View style={{ flexDirection: "row", paddingLeft: 8 }}>
                    <Text style={{ color: "#ff0000ff", fontSize: 12 }}>
                      {isConnected ? "🔴" : "🔘"}
                    </Text>
                  </View>
                )}
              </View>
              {!isLive && (
                <View
                  style={{
                    flexDirection: "row",
                    gap: 10,
                    flex: 1,
                    alignItems: "center",
                    justifyContent: "center",
                  }}
                >
                  <TouchableOpacity
                    style={{
                      padding: 8,
                    }}
                    onPress={() => handleFrame(false)}
                  >
                    <Text style={{ color: "white", fontSize: 12 }}>⏮</Text>
                  </TouchableOpacity>

                  <View style={{ flexDirection: "row" }}>
                    <Text style={{ color: "white", fontSize: 12 }}>
                      {formatTime(seekValue)}
                    </Text>
                    <Text
                      style={{ color: "#fff", fontSize: 12, fontWeight: 900 }}
                    >
                      &nbsp;/&nbsp;
                    </Text>
                    <Text style={{ color: "#b9b9b9ff", fontSize: 12 }}>
                      {formatTime(seekMax)}
                    </Text>
                  </View>

                  <TouchableOpacity
                    style={{
                      padding: 8,
                    }}
                    onPress={() => handleFrame(true)}
                  >
                    <Text style={{ color: "white", fontSize: 12 }}>⏭</Text>
                  </TouchableOpacity>
                </View>
              )}
              <View
                style={{
                  flexDirection: "row-reverse",
                  flex: 1,
                  alignItems: "center",
                }}
              >
                <TouchableOpacity
                  style={{
                    marginLeft: 10,
                    marginRight: 10,
                    padding: 8,
                  }}
                  onPress={() => videoRef.current!.requestFullscreen()}
                >
                  <Text style={{ color: "white", fontSize: 16 }}>⌞ ⌝</Text>
                </TouchableOpacity>
                {!isLive && (
                  <TouchableOpacity
                    style={{
                      marginLeft: 10,
                      marginRight: 10,
                      padding: 8,
                    }}
                    onPress={() => handleLoop()}
                  >
                    <Text style={{ color: "white", fontSize: 16 }}>{isLoop ? "↻": "↿"}</Text>
                  </TouchableOpacity>
                )}
              </View>
            </View>
          </View>
        )}
      </Pressable>
    </View>
  );
};
