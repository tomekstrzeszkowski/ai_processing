import { createContext, useContext, useEffect, useState } from "react";
import { Platform } from "react-native";

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
  p2pPlayer: string;
  setP2pPlayer: (player: string) => void;
  stream: MediaStream | string | null;
  setStream: (stream: MediaStream | string | null) => void;
  host: string;
  setHost: (host: string) => void;
};
const ProtocolContext = createContext<ProtocolContextType | null>(null);

export const useProtocol = () => {
  const context = useContext(ProtocolContext);
  if (!context) {
    throw new Error("useProtocol must be used within a ProtocolContext");
  }
  return context;
};

export const ProtocolProvider = ({
  children,
}: {
  children: React.ReactNode;
}) => {
  const [protocol, setProtocol] = useState<string>("WEBRTC_PROTOCOL");
  const [p2pPlayer, setP2pPlayer] = useState<string>("hls");
  const [isConnected, setIsConnected] = useState(false);
  const [isConnecting, setIsConnecting] = useState(false);
  const [isWebRtc, setIsWebRtc] = useState(true);
  const [lastFrameTime, setLastFrameTime] = useState<string | null>(null);
  const [stream, setStream] = useState<MediaStream | string | null>(null);
  const [host, setHost] = useState<string>(
    process.env.EXPO_PUBLIC_P2P_HOST ??
      (Platform.OS === "web"
        ? document.location.hostname || "http://localhost:7071"
        : "localhost"),
  );
  function handleSetLastFrameTime(time: Date) {
    setLastFrameTime(time.toLocaleTimeString());
  }
  useEffect(() => {
    setIsConnected(false);
    setStream(null);
    setIsWebRtc(protocol === "WEBRTC_PROTOCOL");
  }, [protocol]);
  useEffect(() => {
    setStream(null);
  }, [p2pPlayer]);
  return (
    <ProtocolContext.Provider
      value={{
        protocol,
        setProtocol,
        isConnected,
        setIsConnected,
        isConnecting,
        setIsConnecting,
        lastFrameTime,
        setLastFrameTime: handleSetLastFrameTime,
        isWebRtc,
        p2pPlayer,
        setP2pPlayer,
        stream,
        setStream,
        host,
        setHost,
      }}
    >
      {children}
    </ProtocolContext.Provider>
  );
};
export default ProtocolProvider;
