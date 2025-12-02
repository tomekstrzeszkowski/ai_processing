import { useP2p } from "@/app/p2pProvider";
import { useProtocol } from "@/app/protocolProvider";
import { useWebRtc } from "@/app/webRtcProvider";
import { VideoPlayer } from "@/components/VideoPlayer";
import { formatBytes } from "@/helpers/formatters";
import { useIsFocused } from "@react-navigation/native";
import React, { useEffect, useRef, useState } from "react";
import { Button, Dimensions, ScrollView, StyleSheet, View } from "react-native";
import { SafeAreaProvider, SafeAreaView } from "react-native-safe-area-context";

const { width } = Dimensions.get("window");
const ITEM_MARGIN = 5;
const ITEMS_PER_ROW = 6;
const ITEM_WIDTH = (width - ITEM_MARGIN * (ITEMS_PER_ROW + 1)) / ITEMS_PER_ROW;

export default function videoList() {
  const isFocused = useIsFocused();
  const { isWebRtc, isConnected } = useProtocol();
  const { fetchVideoList: fetchVideoListWs, fetchVideo: fetchVideoWs } =
    useP2p();
  const {
    handlePlayRef: webrtcHandlePlayRef,
    handleStopRef: webrtcHandleStopRef,
    offereeRef,
    setRemoteStream,
  } = useWebRtc();
  const [items, setItems] = useState([]);
  const [videoName, setVideoName] = useState("");
  const videoPlayerRef = useRef<View>(null);
  const scrollViewRef = useRef<ScrollView>(null);
  const [stream, setStream] = useState<MediaStream | string | null>(null);
  const [startDate, setStartDate] = useState(() => {
    const before = new Date();
    before.setDate(before.getDate() - 7);
    return before.toISOString().split("T")[0];
  });
  const [endDate, setEndDate] = useState(
    new Date().toISOString().split("T")[0],
  );
  const [seek, setSeek] = useState<number>(0);
  const [isPlaying, setIsPlaying] = useState<boolean>(false);
  const [isLoop, setIsLoop] = useState<boolean>(false);
  const [duration, setDuration] = useState<number>(0);

  
  useEffect(() => {
    if (!isFocused) return;
    if (!isConnected) {
      setItems([]);
      setStream("");
      setVideoName("");
      return;
    }
    if (items.length === 0) {
      fetchVideoList(startDate, endDate);
    }
    if (isWebRtc) {
      offereeRef.current.registerOrSkipDataChannelListener(
        "status",
        function (
          { 
            seek, 
            isPlaying,
            isLoop,
            duration,
          }: { 
            type: string, 
            seek: number | undefined, 
            isPlaying: boolean | undefined, 
            isLoop: boolean | undefined,
            duration: number | undefined, 
          }
        ) {
          console.log("durat", duration)
          if (seek !== undefined) {
              setSeek(seek);
          }
          if (isPlaying !== undefined) {
            setIsPlaying(isPlaying);
          }
          if (isLoop !== undefined) {
            setIsLoop(isLoop);
          }
          if (duration !== undefined) {
              setDuration(duration);
          }
        },
      );
    }
  }, [isFocused, isConnected]);

  const scrollToVideoPlayer = () => {
    if (scrollViewRef.current) {
      videoPlayerRef.current?.measureInWindow((x, y) => {
        scrollViewRef.current?.scrollTo({
          y,
          animated: false,
        });
      });
    }
  };

  const onChangeDateRange = (startDate: string, endDate: string) => {
    setStartDate(startDate);
    setEndDate(endDate);
    fetchVideoList(startDate, endDate);
  };
  const fetchVideoList = async (startDate: string, endDate: string) => {
    setItems([]);
    let items;
    if (isWebRtc) {
      items = await offereeRef.current.fetchVideoList(startDate, endDate);
    } else {
      items = await fetchVideoListWs(startDate, endDate);
    }
    setItems(items);
  };
  async function fetchVideo(nameToFetch: string) {
    setStream("");
    setVideoName("");
    if (isWebRtc) {
      const stream = await offereeRef.current.fetchVideo(nameToFetch);
      if (stream) setStream(stream);
    } else {
      const { name, videoUrl } = await fetchVideoWs(nameToFetch);
      setIsPlaying(true);
      setStream(videoUrl);
      setVideoName(name);
      setSeek(300);
    }
    setTimeout(() => {
      scrollToVideoPlayer();
    }, 100);
  }

  const renderVideoItem = (item: any, index: number) => (
    <View key={index} style={styles.gridItem}>
      <Button
        title={`${item.Name} (${formatBytes(item.Size)})`}
        color="#007AFF"
        onPress={() => {
          fetchVideo(item.Name);
        }}
      />
    </View>
  );

  async function handleSeek(video: React.RefObject<HTMLVideoElement>, seek: number) {
    console.log("handle Seek", seek, video)
    if (!isWebRtc) {
      video.current.currentTime = seek;
      setSeek(seek);
    } else {
      await offereeRef.current.dataChannel?.send(
        JSON.stringify({ type: "seek", seek }),
      );
    }
  }
  async function handlePause(video: React.RefObject<HTMLVideoElement>) {
    if (!isWebRtc) {
      video.current.pause();
      setIsPlaying(false);
    } else {
      await offereeRef.current.dataChannel?.send(
        JSON.stringify({ type: "pause" }),
      );
    }
  }
  async function handlePlay(video: React.RefObject<HTMLVideoElement>) {
    if (!isWebRtc) {
      video.current.play();
      setIsPlaying(true);
    } else {
      await offereeRef.current.dataChannel?.send(
        JSON.stringify({ type: "resume" }),
      );
    }
  }
  async function handleLoop() {
    if (!isWebRtc) return;
    console.log("SEND LOOP")
    await offereeRef.current.dataChannel?.send(
      JSON.stringify({ type: "loop" }),
    );
  }
  async function handleFrame(isForward: boolean = true) {
    if (!isWebRtc) return;
    await offereeRef.current.dataChannel?.send(
      JSON.stringify({ type: "frame", isForward }),
    );
  }
  function handleMetaData(video: React.RefObject<HTMLVideoElement>) {
    if (isWebRtc) return;
    setDuration(video.current.duration);
  }
  function handleTimeUpdate(video: React.RefObject<HTMLVideoElement>) {
    if (isWebRtc) return;
    setSeek(video.current.currentTime);
  }
  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.container}>
        <ScrollView
          ref={scrollViewRef}
          contentContainerStyle={styles.scrollContent}
        >
          <View>
            <View
              style={{
                display: "flex",
                padding: 20,
                alignItems: "center",
                alignSelf: "center",
                gap: 10,
                flexDirection: "row",
              }}
            >
              <input
                type="date"
                value={startDate}
                onChange={(e) => onChangeDateRange(e.target.value, endDate)}
                style={{ display: "block" }}
              />
              <input
                type="date"
                value={endDate}
                onChange={(e) => onChangeDateRange(startDate, e.target.value)}
                min={startDate}
                style={{ display: "block" }}
              />
            </View>
          </View>
          <View style={styles.gridContainer}>
            {items.map((item, index) => renderVideoItem(item, index))}
          </View>
          <View ref={videoPlayerRef}>
            <VideoPlayer
              stream={stream}
              isConnected={isConnected}
              isLive={false}
              seekValue={seek}
              seekMax={duration}
              isPlaying={isPlaying}
              isLoop={isLoop}
              handleSeek={handleSeek}
              handlePause={handlePause}
              handlePlay={handlePlay}
              handleLoop={handleLoop}
              handleFrame={handleFrame}
              onLoadedMetadata={handleMetaData}
              onTimeUpdate={handleTimeUpdate}
            />
          </View>
        </ScrollView>
      </SafeAreaView>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: "#1a1a1a",
    color: "#b9b9b9ff",
  },
  scrollContent: {
    flexGrow: 1,
  },
  gridContainer: {
    flexDirection: "row",
    flexWrap: "wrap",
    paddingHorizontal: ITEM_MARGIN / 2,
    paddingTop: 10,
  },
  gridItem: {
    width: ITEM_WIDTH,
    marginBottom: ITEM_MARGIN,
    marginHorizontal: ITEM_MARGIN / 2,
  },
});
