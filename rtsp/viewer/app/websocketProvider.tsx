import { createContext, useContext, useRef, useState } from 'react';
import {
  Alert,
  Platform
} from 'react-native';

type WebSocketContextType = {
  wsRef: React.RefObject<WebSocket | null>;
  isConnecting: boolean;
  setIsConnecting: (isConnecting: boolean) => void;
  serverUrl: string;
  httpServerUrl: string;
};

const WebSocketContext = createContext<WebSocketContextType | null>(null);
const showAlert = (title: string, message: string) => {
  if (Platform.OS === 'web') {
    alert(`${title}: ${message}`);
  } else {
    Alert.alert(title, message);
  }
};
export const useWebSocket = () => {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
};

export const WebSocketProvider = ({ children }: { children: React.ReactNode }) => {
  const wsRef = useRef<WebSocket>(null);
  const [isConnecting, setIsConnecting] = useState(false);
  const [isConnected, setIsConnected] = useState(false);
  const lastUpdateRef = useRef(0);
  const MIN_FRAME_INTERVAL = 33; // ~30fps max (adjust as needed)
  const host = document.location.hostname || 'localhost';
  const [serverUrl, ] = useState(`ws://${host}:7080/ws`);
  const [httpServerUrl, ] = useState(`http://${host}:7080`);
  const frameCountRef = useRef(0);
  const [imageUri, setImageUri] = useState<string | null>(null);
  const [lastFrameTime, setLastFrameTime] = useState<string | null>(null);
  const [clientCount, setClientCount] = useState(0);
  const handlePlayRef = useRef<Function>(() => {});
  const handleStopRef = useRef<Function>(() => {})

  handlePlayRef.current = function () {
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
            
            setImageUri(uri);
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
  };

  handleStopRef.current = function () {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
      setImageUri(null);
      frameCountRef.current = 0;
  }

  const value = {
    wsRef, 
    isConnecting, 
    setIsConnecting, 
    serverUrl, 
    httpServerUrl, 
    handlePlayRef,
    isConnected, 
    imageUri,
    frameCountRef,
    clientCount,
    lastFrameTime,
    setClientCount,
    setImageUri,
    handleStopRef,
  };
  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
};

export default WebSocketProvider;