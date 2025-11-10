import { createContext, useContext, useRef, useState } from 'react';
type ProtocolContextType = {
  protocol: React.RefObject<string | null>;
  isConnected: boolean;
  setIsConnected: (isConnected: boolean) => void;
  isConnecting: boolean;
  setIsConnecting: (isConnecting: boolean) => void;
  setLastFrameTime: (time: Date) => void;
  lastFrameTime: string | null;
};
const ProtocolContext = createContext<ProtocolContextType | null>(null);

export const useProtocol = () => {
  const context = useContext(ProtocolContext);
  if (!context) {
    throw new Error('useProtocol must be used within a ProtocolContext');
  }
  return context;
};

export const ProtocolProvider = ({ children }: { children: React.ReactNode }) => {
  const protocol = useRef<string>("WEBRTC_PROTOCOL");
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const [lastFrameTime, setLastFrameTime] = useState<string | null>(null);
  const handleSetLastFrameTime = (time: Date) => {
    setLastFrameTime(time.toLocaleTimeString());
  };
  return (
    <ProtocolContext.Provider value={{ 
      protocol, 
      isConnected, 
      setIsConnected, 
      isConnecting, 
      setIsConnecting,
      lastFrameTime,
      setLastFrameTime: handleSetLastFrameTime
    }}>
      {children}
    </ProtocolContext.Provider>
  );
};
export default ProtocolProvider;