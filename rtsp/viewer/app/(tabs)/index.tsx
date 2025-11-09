import { CachedVideoPlayer } from '@/components/CachedVideoPlayer';
import { LiveVideoPlayer } from '@/components/LiveVideoPlayer';
import React, { useEffect, useRef } from 'react';
import {
  ActivityIndicator,
  Dimensions,
  Platform,
  StyleSheet,
  Text,
  TouchableOpacity,
  View
} from 'react-native';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';
import { useProtocol } from '../protocolProvider';
import { useWebRtc } from '../webRtcProvider';
import { useWebSocket } from '../websocketProvider';

const { width, height } = Dimensions.get('window');

const App = () => {
  const lastUriRef = useRef(null);

  
  const {
    isConnecting, 
    setIsConnecting, 
    serverUrl, 
    httpServerUrl, 
    handlePlayRef: wsHandlePlayRef, 
    isConnected, 
    imageUri,
    frameCountRef,
    clientCount,
    lastFrameTime,
    setClientCount,
    handleStopRef: wsHandleStopRef,
  } = useWebSocket();
  const { protocol } = useProtocol();
  const { handlePlayRef, handleStopRef: webrtcHandleStopRef, isConnected: isWebRtcConnected } = useWebRtc();

  const connect = () => {
    if (isConnecting || isConnected) return;
    setIsConnecting(true);
    if (protocol.current === "WEBRTC_PROTOCOL") {
      handlePlayRef.current();
    } else {
      wsHandlePlayRef.current();
    }
  };

  const disconnect = () => {
    if (protocol.current === "WEBRTC_PROTOCOL") {
      webrtcHandleStopRef.current();
    } else {
      wsHandleStopRef.current();
    }
  };

  const fetchStatus = async () => {
    try {
      const finalUrl = `${httpServerUrl}/status`;
      const response = await fetch(finalUrl, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (response.ok) {
        const data = await response.json();
        setClientCount(data.clients || 0);
      }
    } catch (error) {
      console.error('Error fetching status:', error);
    }
  };
  useEffect(() => {
    if (protocol.current === "WEBRTC_PROTOCOL") {
      console.log('WebRTC connection state changed:', isWebRtcConnected);
      setIsConnected(isWebRtcConnected);
      if (isWebRtcConnected) {
        setIsConnecting(false);
      }
    }
  }), [isWebRtcConnected];
  useEffect(() => {
    const interval = setInterval(() => {
      if (isConnected) {
        fetchStatus();
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [isConnected, serverUrl]);

  useEffect(() => {
    return () => {
      wsHandleStopRef.current();
    };
  }, []);

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.container}>
        <CachedVideoPlayer
          imageUri={imageUri}
          frameCountRef={frameCountRef}
          styles={styles}
          isConnected={isConnected}
        />
        <LiveVideoPlayer isConnected={isConnected} />
        

        <View style={styles.connectionContainer}>
          <TouchableOpacity
            style={[styles.button, isConnected ? styles.disconnectButton : styles.connectButton]}
            onPress={isConnected ? disconnect : connect}
            disabled={isConnecting}
          >
            {isConnecting ? (
              <ActivityIndicator color="#fff" size="small" />
            ) : (
              <Text style={styles.buttonText}>
                {isConnected ? 'Disconnect' : 'Connect'}
              </Text>
            )}
          </TouchableOpacity>
        </View>

        <View style={styles.infoContainer}>
          <View style={styles.infoRow}>
            <Text style={styles.infoLabel}>Platform:</Text>
            <Text style={styles.infoValue}>{Platform.OS}</Text>
          </View>
          <View style={styles.infoRow}>
            <Text style={styles.infoLabel}>Connected Clients:</Text>
            <Text style={styles.infoValue}>{clientCount}</Text>
          </View>
          {lastFrameTime && (
            <View style={styles.infoRow}>
              <Text style={styles.infoLabel}>Last Frame:</Text>
              <Text style={styles.infoValue}>{lastFrameTime}</Text>
            </View>
          )}
          {frameCountRef.current > 0 && (
            <View style={styles.infoRow}>
              <Text style={styles.infoLabel}>Frames Received:</Text>
              <Text style={styles.infoValue}>{frameCountRef.current}</Text>
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
    ...(Platform.OS === 'web' && {
      maxHeight: '100vh',
      overflow: 'hidden',
    }),
  },
  header: {
    padding: 20,
    borderBottomWidth: 1,
    borderBottomColor: '#333',
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#fff',
    textAlign: 'center',
    marginBottom: 10,
  },
  statusContainer: {
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: 8,
  },
  statusText: {
    color: '#fff',
    fontSize: 16,
  },
  connectionContainer: {
    padding: 20,
    borderBottomWidth: 1,
    borderBottomColor: '#333',
  },
  label: {
    color: '#fff',
    fontSize: 16,
    marginBottom: 8,
  },
  input: {
    backgroundColor: '#333',
    color: '#fff',
    padding: 12,
    borderRadius: 8,
    marginBottom: 15,
    fontSize: 16,
    ...(Platform.OS === 'web' && {
      outlineStyle: 'none',
    }),
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
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: 'bold',
  },
  videoContainer: {
    flex: 1,
    margin: 20,
    borderRadius: 12,
    overflow: 'hidden',
    backgroundColor: '#000',
    position: 'relative',
  },
  video: {
    width: '100%',
    height: '100%',
  },
  noVideoContainer: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#000',
  },
  noVideoText: {
    color: '#666',
    fontSize: 16,
    textAlign: 'center',
    paddingHorizontal: 20,
  },
  placeholderIcon: {
    width: 48,
    height: 48,
    marginBottom: 16,
  },
  loading: {
    opacity: 0.8,
  },
  infoContainer: {
    padding: 20,
    borderTopWidth: 1,
    borderTopColor: '#333',
  },
  infoRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  infoLabel: {
    color: '#999',
    fontSize: 14,
  },
  infoValue: {
    color: '#fff',
    fontSize: 14,
    fontWeight: 'bold',
  },
});

export default App;