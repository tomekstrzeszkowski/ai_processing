import { useProtocol } from '@/app/protocolProvider';
import { createContext, useContext, useEffect, useRef, useState } from 'react';

type P2pContextType = {
  setIsConnecting: (isConnecting: boolean) => void;
  httpServerUrl: string;
  handlePlayRef: React.RefObject<Function>;
  handleStopRef: React.RefObject<Function>;
  fetchVideoList: Function;
  fetchVideo: Function;
};

const P2pContext = createContext<P2pContextType | null>(null);
export const useP2p = () => {
  const context = useContext(P2pContext);
  if (!context) {
    throw new Error('useP2p must be used within a P2pProvider');
  }
  return context;
};

export const P2pProvider = ({ children }: { children: React.ReactNode }) => {
  const {
    setIsConnected, 
    setIsConnecting, 
    isConnected, 
    setLastFrameTime,
    isWebRtc,
    host,
  } = useProtocol();
  const [httpServerUrl, ] = useState(`http://${host}:7080`);
  const [_, setClientCount] = useState(0);
  const handlePlayRef = useRef<Function>(() => {});
  const handleStopRef = useRef<Function>(() => {});

  handlePlayRef.current = function () {}

  handleStopRef.current = function () {}

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

  async function fetchVideo(name: string): Promise<Object> {
    const url = `${httpServerUrl}/video/${name}`;
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Accept': 'video/mp4,video/*'
      },
    });
    
    if (response.ok) {
      // Get video data as ArrayBuffer
      const buffer = await response.arrayBuffer();
      console.log('Received video data, size:', buffer.byteLength);
      
      const bytes = new Uint8Array(buffer);
      const videoUrl = URL.createObjectURL(new Blob([bytes], { type: 'video/mp4' }));
      console.log('Created video URL:', videoUrl);
      return {
        name, videoUrl,
      }
    }
    return {}
  };

  async function fetchVideoList(startDate: string, endDate: string): Promise<Array<Object>> {
    const finalUrl = `${httpServerUrl}/video-list?start=${startDate}&end=${endDate}`;
    const response = await fetch(finalUrl, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
    });
    
    if (response.ok) {
      const data = await response.json();
      return data??[];
    }
    return [];
  };

  return (
    <P2pContext.Provider value={{
      setIsConnecting,
      httpServerUrl, 
      handlePlayRef,
      handleStopRef,
      fetchVideoList,
      fetchVideo,
    }}>
      {children}
    </P2pContext.Provider>
  );
};

export default P2pProvider;