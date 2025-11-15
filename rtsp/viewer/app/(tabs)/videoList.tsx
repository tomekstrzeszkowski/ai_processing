import { VideoPlayer } from '@/components/VideoPlayer';
// No external converter needed
import { useProtocol } from '@/app/protocolProvider';
import { useWebRtc } from '@/app/webRtcProvider';
import { useWebSocket } from '@/app/websocketProvider';
import React, { useEffect, useRef, useState } from 'react';
import {
  Button,
  Dimensions,
  ScrollView,
  StyleSheet,
  View
} from 'react-native';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';

const { width } = Dimensions.get('window');
const ITEM_MARGIN = 5;
const ITEMS_PER_ROW = 6;
const ITEM_WIDTH = (width - (ITEM_MARGIN * (ITEMS_PER_ROW + 1))) / ITEMS_PER_ROW;

export default () => {
  const { isWebRtc } = useProtocol();
  const { fetchVideoList: fetchVideoListWs, fetchVideo: fetchVideoWs } = useWebSocket();
  const { handlePlayRef: webrtcHandlePlayRef, handleStopRef: webrtcHandleStopRef, offereeRef }= useWebRtc();
  const [items, setItems] = useState([]);
  const [videoData, setVideoData] = useState("");
  const [videoName, setVideoName] = useState("");
  const videoPlayerRef = useRef<View>(null);
  const scrollViewRef = useRef<ScrollView>(null);

  // Cleanup MediaSource and blob URLs when component unmounts or video changes
  useEffect(() => {
    return () => {
      if (videoData && videoData.startsWith('blob:')) {
        const url = videoData;
        // Give time for the video element to release the MediaSource
        setTimeout(() => {
          try {
            URL.revokeObjectURL(url);
          } catch (e) {
            console.warn('Error revoking URL:', e);
          }
        }, 100);
      }
    };
  }, [videoData]);

  const [startDate, setStartDate] = useState(() => {
    const before = new Date();
    before.setDate(before.getDate() - 7);
    return before.toISOString().split('T')[0];
  });
  const [endDate, setEndDate] = useState(new Date().toISOString().split('T')[0]);
  const scrollToVideoPlayer = () => {
    if (scrollViewRef.current) {
      videoPlayerRef.current?.measureInWindow((x, y) => {
        scrollViewRef.current?.scrollTo({
          y,
          animated: false
        });
      });
    }
  };

  const onChangeDateRange = (startDate: string, endDate: string) => {
    setStartDate(startDate);
    setEndDate(endDate);

    fetchVideoList(startDate, endDate);
  }
  const fetchVideoList = async (startDate: string, endDate: string) => {
    setItems([]);
    let items;
    if (isWebRtc) {
      if (!offereeRef.current.isConnected()) {
        await webrtcHandlePlayRef.current();
      }
      items = await offereeRef.current.fetchVideoList(startDate, endDate);
    } else {
      items = await fetchVideoListWs(startDate, endDate);  
    }
    setItems(items);
  };
  const fetchVideo = async (nameToFetch: string) => {
    setVideoData("");
    setVideoName("");
    const {name, videoUrl} = await fetchVideoWs(nameToFetch);
    setVideoData(videoUrl);
    setVideoName(name);
    setTimeout(() => {
      scrollToVideoPlayer();
    }, 100);
  };

  const renderVideoItem = (item: any, index: number) => (
    <View key={index} style={styles.gridItem}>
      <Button 
        title={`${item.Name} (${item.Size ?? "-"})`}
        color="#007AFF"
        onPress={() => {
          fetchVideo(item.Name);
        }}
      />
    </View>
  );

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.container}>
        <ScrollView 
          ref={scrollViewRef}
          contentContainerStyle={styles.scrollContent}>
          <View>
            <div style={{ 
              display: 'flex', 
              padding: 20, 
              alignItems: 'center', 
              alignSelf: 'center',
              gap: 20
            }}>
              <input
                type="date"
                value={startDate}
                onChange={(e) => onChangeDateRange(e.target.value, endDate)}
              />
              <input
                type="date"
                value={endDate}
                onChange={(e) => onChangeDateRange(startDate, e.target.value)}
                min={startDate}
              />
            </div>
          </View>
          <View style={styles.gridContainer}>
            {items.map((item, index) => renderVideoItem(item, index))}
          </View>
          <View ref={videoPlayerRef}>
            {videoData ? <VideoPlayer videoUrl={videoData} name={videoName} scrollView={scrollViewRef} /> : <div>No video selected</div>}
          </View>
        </ScrollView>
      </SafeAreaView>
    </SafeAreaProvider>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#1a1a1a',
    color: "#b9b9b9ff"
  },
  scrollContent: {
    flexGrow: 1,
  },
  gridContainer: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'space-around',
    paddingHorizontal: ITEM_MARGIN / 2,
    paddingTop: 10,
  },
  gridItem: {
    width: ITEM_WIDTH,
    marginBottom: ITEM_MARGIN,
    marginHorizontal: ITEM_MARGIN / 2,
  },
});