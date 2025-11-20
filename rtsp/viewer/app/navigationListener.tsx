import { useP2p } from '@/app/p2pProvider';
import { useProtocol } from '@/app/protocolProvider';
import { useWebRtc } from '@/app/webRtcProvider';
import { usePathname, useSegments } from 'expo-router';
import 'react-native-reanimated';

import { useEffect } from 'react';
export function NavigationListener() {
  const pathname = usePathname();
  const segments = useSegments();
  const {
    isWebRtc,
    p2pPlayer,
    setStream,
    isConnected,
    isConnecting,
    host,
  } = useProtocol();
  const { handlePlayRef: wsHandlePlayRef, handleStopRef: wsHandleStopRef } = useP2p();
  const { handlePlayRef: webrtcHandlePlayRef, handleStopRef: webrtcHandleStopRef, offereeRef } = useWebRtc();

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
        if (p2pPlayer === "hls") {
            setStream(`http://${host}:7071/hls/stream.m3u8`);
            console.log("Set stream to HLS URL");
        } else {
            setStream(`http://${host}:7071/stream`);
            console.log("Set stream to Image URL");
        }
    }
  }, [pathname, segments]);

  return null;
}