import { createContext, useContext, useEffect, useState } from 'react';
type ProtocolContextType = {
  protocol: string;
  setProtocol: (protocol: string) => void;
  isConnected: boolean;
  setIsConnected: (isConnected: boolean) => void;
  isConnecting: boolean;
  setIsConnecting: (isConnecting: boolean) => void;
  setLastFrameTime: (time: Date) => void;
  lastFrameTime: string | null;
  isWebRtc: boolean;
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
  const [protocol, setProtocol] = useState<string>("WEBRTC_PROTOCOL");
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const [isWebRtc, setIsWebRtc] = useState(false);
  const [lastFrameTime, setLastFrameTime] = useState<string | null>(null);
  function handleSetLastFrameTime(time: Date) {
    setLastFrameTime(time.toLocaleTimeString());
  };
  useEffect(() => {
    setIsWebRtc(protocol === "WEBRTC_PROTOCOL");
  }, [protocol]);
  return (
    <ProtocolContext.Provider value={{ 
      protocol, 
      setProtocol,
      isConnected, 
      setIsConnected, 
      isConnecting, 
      setIsConnecting,
      lastFrameTime,
      setLastFrameTime: handleSetLastFrameTime,
      isWebRtc,
    }}>
      {children}
    </ProtocolContext.Provider>
  );
};
export default ProtocolProvider;