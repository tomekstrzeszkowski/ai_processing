import { useProtocol } from '@/app/protocolProvider';
import { useToast } from '@/app/toastProvider';
import { createContext, useContext, useEffect, useRef, useState } from 'react';
import {
  Platform
} from 'react-native';

type WebSocketContextType = {
  wsRef: React.RefObject<WebSocket | null>;
  setIsConnecting: (isConnecting: boolean) => void;
  httpServerUrl: string;
  handlePlayRef: React.RefObject<Function>;
  handleStopRef: React.RefObject<Function>;
  imageUri: string | null;
};

const WebSocketContext = createContext<WebSocketContextType | null>(null);
export const useWebSocket = () => {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
};

export const WebSocketProvider = ({ children }: { children: React.ReactNode }) => {
  const wsRef = useRef<WebSocket>(null);
  const {
    setIsConnected, 
    setIsConnecting, 
    isConnected, 
    setLastFrameTime,
    isWebRtc,
  } = useProtocol();
  const { showAlert } = useToast();
  const lastUpdateRef = useRef(0);
  const MIN_FRAME_INTERVAL = 33; // ~30fps max (adjust as needed)
  const host = document.location.hostname || 'localhost';
  const [serverUrl, ] = useState(`ws://${host}:7080/ws`);
  const [httpServerUrl, ] = useState(`http://${host}:7080`);
  const [imageUri, setImageUri] = useState<string | null>(null);
  const [_, setClientCount] = useState(0);
  const handlePlayRef = useRef<Function>(() => {});
  const handleStopRef = useRef<Function>(() => {})

  handlePlayRef.current = function () {
    if (isWebRtc) return;
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
            
            // Create new URI
            const uri = `data:image/jpeg;base64,${message.data}`;
            
            setImageUri(uri);
            setLastFrameTime(new Date());
            
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
  }

  const fetchStatus = async () => {
    if (isWebRtc) return;
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
  }, [isConnected]);
  return (
    <WebSocketContext.Provider value={{
      wsRef,
      setIsConnecting,
      httpServerUrl, 
      handlePlayRef,
      imageUri,
      handleStopRef,
    }}>
      {children}
    </WebSocketContext.Provider>
  );
};

export default WebSocketProvider;