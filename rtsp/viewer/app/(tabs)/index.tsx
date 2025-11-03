import { CachedVideoPlayer } from '@/components/CachedVideoPlayer';
import { LiveVideoPlayer } from '@/components/LiveVideoPlayer';
import { StatusBar } from 'expo-status-bar';
import React, { useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
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
  const [isConnected, setIsConnected] = useState(false);
  const [lastFrameTime, setLastFrameTime] = useState(null);
  const [clientCount, setClientCount] = useState(0);
  const [imageUri, setImageUri] = useState(null);
  const frameCountRef = useRef(0);
  
  // Frame throttling to prevent memory overflow
  const lastUpdateRef = useRef(0);
  const lastUriRef = useRef(null);
  const MIN_FRAME_INTERVAL = 33; // ~30fps max (adjust as needed)

  const showAlert = (title, message) => {
    if (Platform.OS === 'web') {
      alert(`${title}: ${message}`);
    } else {
      Alert.alert(title, message);
    }
  };
  
  const {
    wsRef, isConnecting, setIsConnecting, serverUrl, httpServerUrl
  } = useWebSocket();
  const { protocol } = useProtocol();
  const { handlePlayRef, handlePauseRef } = useWebRtc();

  const connect = () => {
    if (protocol.current === "WEBRTC_PROTOCOL") {
      handlePlayRef.current();
      setIsConnected(true);
    } else {
      if (isConnecting || isConnected) return;
      
      setIsConnecting(true);
      
      try {
        // For web, ensure we use the correct WebSocket URL
        const wsUrl = Platform.OS === 'web' && serverUrl.includes('localhost') 
          ? serverUrl.replace('localhost', window.location.hostname)
          : serverUrl;
        
        wsRef.current = new WebSocket(wsUrl);
        
        wsRef.current.onopen = () => {
          setIsConnected(true);
          setIsConnecting(false);
          console.log('Connected to WebSocket server');
        };
        
        wsRef.current.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data);
            if (message.type === 'frame') {
              // CRITICAL: Frame throttling to prevent memory overflow
              const now = Date.now();
              if (now - lastUpdateRef.current < MIN_FRAME_INTERVAL) {
                // Skip this frame - too soon since last update
                return;
              }
              
              lastUpdateRef.current = now;
              frameCountRef.current += 1;
              
              // Create new URI
              const uri = `data:image/jpeg;base64,${message.data}`;
              
              // Clear old URI reference to help GC
              lastUriRef.current = null;
              
              setImageUri(uri);
              lastUriRef.current = uri;
              setLastFrameTime(new Date().toLocaleTimeString());
              
            } else if (message.type === 'client_count') {
              setClientCount(message.count);
            }
          } catch (error) {
            console.error('Error parsing message:', error);
          }
        };
        
        wsRef.current.onclose = (event) => {
          setIsConnected(false);
          setIsConnecting(false);
          setImageUri(null);
          lastUriRef.current = null;
          frameCountRef.current = 0;
          console.log('Disconnected from WebSocket server', event.code, event.reason);
        };
        
        wsRef.current.onerror = (error) => {
          setIsConnecting(false);
          showAlert('Connection Error', 'Failed to connect to server');
          console.error('WebSocket error:', error);
        };
      } catch (error) {
        setIsConnecting(false);
        showAlert('Connection Error', 'Invalid server URL');
      }
    }
  };

  const disconnect = () => {
    if (protocol.current === "WEBRTC_PROTOCOL") {
    handlePauseRef.current();
    setIsConnected(false);
    } else {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      setImageUri(null);
      lastUriRef.current = null;
      frameCountRef.current = 0;
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
    const interval = setInterval(() => {
      if (isConnected) {
        fetchStatus();
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [isConnected, serverUrl]);

  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      // Cleanup references
      setImageUri(null);
      lastUriRef.current = null;
    };
  }, []);

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.container}>
        <View style={styles.videoContainer}>
          <CachedVideoPlayer 
            imageUri={imageUri}
            frameCountRef={frameCountRef}
            styles={styles}
          />
          {!imageUri && (
            <View style={styles.noVideoContainer}>
              <Text style={styles.noVideoText}>
                {isConnected ? 'Waiting for video frames...' : 'Connect to server to view stream'}
              </Text>
            </View>
          )}
        </View>

        <LiveVideoPlayer />
        
        <StatusBar style="light" backgroundColor="#1a1a1a" />

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