import { useProtocol } from "@/app/protocolProvider";
import { createContext, useContext, useRef } from "react";

type P2pContextType = {
  setIsConnecting: (isConnecting: boolean) => void;
  handlePlayRef: React.RefObject<Function>;
  handleStopRef: React.RefObject<Function>;
  fetchVideoList: Function;
  fetchVideo: Function;
};

const P2pContext = createContext<P2pContextType | null>(null);
export const useP2p = () => {
  const context = useContext(P2pContext);
  if (!context) {
    throw new Error("useP2p must be used within a P2pProvider");
  }
  return context;
};

export const P2pProvider = ({ children }: { children: React.ReactNode }) => {
  const { setIsConnected, setIsConnecting, host } = useProtocol();
  const handlePlayRef = useRef<Function>(() => {});
  const handleStopRef = useRef<Function>(() => {});

  handlePlayRef.current = async function (callback: Function | null) {
    try {
      const response = await fetch(host, {
        method: "GET",
      });
      if (!response.ok) {
        return;
      }
    } catch {
      return;
    } finally {
      setIsConnecting(false);
    }
    setIsConnected(true);
    if (callback) {
      callback();
    }
  };

  handleStopRef.current = function () {};
  async function fetchVideo(name: string): Promise<Object> {
    const url = `${host}/video/${name}`;
    const response = await fetch(url, {
      method: "GET",
      headers: {
        Accept: "video/mp4,video/*",
      },
    });

    if (response.ok) {
      // Get video data as ArrayBuffer
      const buffer = await response.arrayBuffer();
      console.log("Received video data, size:", buffer.byteLength);

      const bytes = new Uint8Array(buffer);
      const videoUrl = URL.createObjectURL(
        new Blob([bytes], { type: "video/mp4" }),
      );
      console.log("Created video URL:", videoUrl);
      return {
        name,
        videoUrl,
      };
    }
    return {};
  }

  async function fetchVideoList(
    startDate: string,
    endDate: string,
  ): Promise<Array<Object>> {
    const finalUrl = `${host}/video-list?start=${startDate}&end=${endDate}`;
    const response = await fetch(finalUrl, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
    });

    if (response.ok) {
      const data = await response.json();
      return data ?? [];
    }
    return [];
  }

  return (
    <P2pContext.Provider
      value={{
        setIsConnecting,
        handlePlayRef,
        handleStopRef,
        fetchVideoList,
        fetchVideo,
      }}
    >
      {children}
    </P2pContext.Provider>
  );
};

export default P2pProvider;
