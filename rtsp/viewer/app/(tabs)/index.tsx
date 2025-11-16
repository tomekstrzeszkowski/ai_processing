import { useProtocol } from '@/app/protocolProvider';
import { useWebRtc } from '@/app/webRtcProvider';
import { useWebSocket } from '@/app/websocketProvider';
import { CachedVideoPlayer } from '@/components/CachedVideoPlayer';
import { LiveVideoPlayer } from '@/components/LiveVideoPlayer';
import { useIsFocused } from '@react-navigation/native';
import React, { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  StyleSheet,
  Text,
  TouchableOpacity,
  View
} from 'react-native';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';

const App = () => {
  const isFocused = useIsFocused();
  const { 
    isConnected, 
    isConnecting, 
    setIsConnecting, 
    lastFrameTime,
    isWebRtc,
  } = useProtocol();
  const { handlePlayRef: wsHandlePlayRef, handleStopRef: wsHandleStopRef } = useWebSocket();
  const { handlePlayRef: webrtcHandlePlayRef, handleStopRef: webrtcHandleStopRef, offereeRef } = useWebRtc();
  const [stream, setStream] = useState<MediaStream | null>(null);

  useEffect(() => {
    if (!isFocused || !(isWebRtc && isConnected)) return;
    const firstKey = [...offereeRef.current.streamIdToStream.keys()]?.[0];
    if (firstKey) {
      const mainStream = offereeRef.current.streamIdToStream.get(firstKey);
      if (mainStream) {
        setStream(mainStream);
      }
    }
  }, [isFocused]);
  const connect = () => {
    if (isConnecting || isConnected) return;
    setIsConnecting(true);
    if (isWebRtc) {
      webrtcHandlePlayRef.current((stream: MediaStream) => {
        setStream(stream);
      });
    } else {
      wsHandlePlayRef.current();
    }
  };

  const disconnect = () => {
    if (isWebRtc) {
      webrtcHandleStopRef.current();
    } else {
      wsHandleStopRef.current();
    }
  };

  useEffect(() => {
    return () => {
      wsHandleStopRef.current();
    };
  }, []);

  // useEffect(() => {
  //   console.log("STREAM", remoteStream)
  //   setStream(remoteStream)
  // }, [remoteStream])

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.container}>
        {isWebRtc && <LiveVideoPlayer stream={stream} isConnected={isConnected} />}
        {!isWebRtc && <CachedVideoPlayer isConnected={isConnected} styles={styles} />}

        <View style={styles.connectionContainer}>
          <TouchableOpacity
            style={[styles.button, isConnected ? styles.disconnectButton : styles.connectButton]}
            onPress={isConnected ? disconnect : connect}
            disabled={isConnecting}
          >
            {isConnecting ? (
              <ActivityIndicator color="#fff" size="small" />
            ) : (
              <Text style={{color: "#fff", fontWeight: 700}}>
                {isConnected ? 'Disconnect' : 'Connect'}
              </Text>
            )}
          </TouchableOpacity>
        </View>

        <View>
          {lastFrameTime && (
            <View style={{ flex: 1, color: "white", padding: 20}}>
              <Text style={{color: "white"}}>Last Signal:</Text>
              <Text style={{color: "white"}}>{lastFrameTime}</Text>
            </View>
          )}
        </View>
      </SafeAreaView>
    </SafeAreaProvider>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#1a1a1a',
    color: "#b9b9b9ff",
    overflow: "auto"
  },
  connectionContainer: {
    padding: 20,
    borderBottomWidth: 1,
    borderBottomColor: '#333',
  },
  button: {
    padding: 15,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 50,
  },
  connectButton: {
    backgroundColor: '#4CAF50',
  },
  disconnectButton: {
    backgroundColor: '#f44336',
  },
});

export default App;