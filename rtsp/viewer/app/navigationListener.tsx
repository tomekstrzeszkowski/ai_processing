import { useP2p } from "@/app/p2pProvider";
import { useProtocol } from "@/app/protocolProvider";
import { useWebRtc } from "@/app/webRtcProvider";
import { usePathname, useSegments } from "expo-router";
import "react-native-reanimated";

import { useEffect } from "react";
export function NavigationListener() {
  const pathname = usePathname();
  const segments = useSegments();
  const { isWebRtc, p2pPlayer, setStream, isConnected, isConnecting, host } =
    useProtocol();
  const { handlePlayRef: p2pHandlePlayRef, handleStopRef: p2pHandleStopRef } =
    useP2p();
  const {
    handlePlayRef: webrtcHandlePlayRef,
    handleStopRef: webrtcHandleStopRef,
    offereeRef,
  } = useWebRtc();

  useEffect(() => {
    if (isWebRtc) {
      if (isConnected || isConnecting) return;
      webrtcHandlePlayRef.current((stream: MediaStream) => {
        setStream(stream);
      });
      const firstKey = [...offereeRef.current.streamIdToStream.keys()]?.[0];
      if (firstKey) {
        const mainStream = offereeRef.current.streamIdToStream.get(firstKey);
        if (mainStream) {
          setStream(mainStream);
        }
      }
    } else {
      p2pHandlePlayRef.current(() => {
        if (p2pPlayer === "hls") {
          setStream(`${host}/hls/stream.m3u8`);
        } else {
          setStream(`${host}/stream`);
        }
      });
    }
  }, [pathname, segments]);

  return null;
}
