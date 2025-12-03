import { useP2p } from "@/app/p2pProvider";
import { useProtocol } from "@/app/protocolProvider";
import { useWebRtc } from "@/app/webRtcProvider";
import { VideoPlayer } from "@/components/VideoPlayer";
import { formatBytes } from "@/helpers/formatters";
import { useIsFocused } from "@react-navigation/native";
import React, { useEffect, useRef, useState } from "react";
import { ActivityIndicator, Button, Dimensions, ScrollView, StyleSheet, Text, View } from "react-native";
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
  const [videoNameLoading, setVideoNameLoading] = useState("");
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
    setVideoNameLoading(nameToFetch);
    let stream;
    if (isWebRtc) {
      stream = await offereeRef.current.fetchVideo(nameToFetch);
      setVideoName(nameToFetch);
    } else {
      const { name, videoUrl } = await fetchVideoWs(nameToFetch);
      stream = videoUrl;
      setIsPlaying(true);
      setVideoName(name);
    }
    if (stream) {
      setStream(stream);
    }
    setVideoNameLoading("");
  }

  const renderVideoItem = (item: any, index: number) => (
    <View key={index} style={styles.gridItem}>
      {item.Name === videoNameLoading ? (
        <View style={{backgroundColor: "#0b55a5ff", flex: 1, alignItems: "center", justifyContent: "center"}}>
          <Text style={{alignItems: "center", justifyContent: "center", color: "#fff"}}>
            {item.Name}
            <ActivityIndicator color="#fff" size={10} />
          </Text>
        </View>
      ): (
        <Button
          title={`${item.Name} (${formatBytes(item.Size)})`}
          color={videoName !== item.Name ? "#007AFF": "#0b55a5ff"}
          onPress={() => {
            fetchVideo(item.Name);
          }}
        />
      )}
    </View>
  );

  async function handleSeek(video: React.RefObject<HTMLVideoElement>, seek: number) {
    console.log("handle Seek", seek, video)
    if (!isWebRtc) {
      video.current.currentTime = seek;
      setSeek(video.current.currentTime);
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
    if (!isWebRtc) {
      setIsLoop(!isLoop);
    } else {
      await offereeRef.current.dataChannel?.send(
        JSON.stringify({ type: "loop" }),
      );
    }
  }
  async function handleFrame(video: React.RefObject<HTMLVideoElement>, isForward: boolean = true) {
    if (!isWebRtc) {
      video.current.currentTime += 1/30 * (isForward ? 1: -1);
      setSeek(video.current.currentTime);
    } else {
      await offereeRef.current.dataChannel?.send(
        JSON.stringify({ type: "frame", isForward }),
      );
    }
  }
  function handleMetaData(video: React.RefObject<HTMLVideoElement>) {
    if (isWebRtc) return;
    setDuration(video.current.duration);
  }
  function handleTimeUpdate(video: React.RefObject<HTMLVideoElement>) {
    if (isWebRtc) return;
    setSeek(video.current.currentTime);
  }
  function onEnded(video: React.RefObject<HTMLVideoElement>) {
    if (isWebRtc) return;
    console.log('ended', isLoop)
    if (!isLoop) {
      video.current.pause();
      setIsPlaying(false);
    } else {
      video.current.play();
      setIsPlaying(true);
    }
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
              onEnded={onEnded}
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
