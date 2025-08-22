import { createContext, useContext, useRef, useState } from 'react';

type WebSocketContextType = {
  wsRef: React.RefObject<WebSocket | null>;
  isConnecting: boolean;
  setIsConnecting: (isConnecting: boolean) => void;
  serverUrl: string;
  httpServerUrl: string;
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
  const wsRef = useRef(null);
  const [isConnecting, setIsConnecting] = useState(false);
  const [serverUrl, ] = useState('ws://localhost:8080/ws');
  const [httpServerUrl, ] = useState('http://localhost:8080');//71

  const value = {wsRef, isConnecting, setIsConnecting, serverUrl, httpServerUrl};
  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
};