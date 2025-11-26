import { useP2p } from "@/app/p2pProvider";
import { useProtocol } from "@/app/protocolProvider";
import { useToast } from "@/app/toastProvider";
import { useWebRtc } from "@/app/webRtcProvider";
import { usePathname, useSegments } from "expo-router";
import "react-native-reanimated";

import { useCallback, useEffect, useRef, useState } from "react";

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
    setProtocol,
    protocol,
    setIsConnecting,
  } = useProtocol();
  const { handlePlayRef: p2pHandlePlayRef, handleStopRef: p2pHandleStopRef } =
    useP2p();
  const {
    handlePlayRef: webrtcHandlePlayRef,
    handleStopRef: webrtcHandleStopRef,
    offereeRef,
  } = useWebRtc();
  const { showAlert } = useToast();
  const [connectionAttempts, setConnectionAttempts] = useState(0);
  const resolversRef = useRef<Array<() => void>>([]);

  useEffect(() => {
    if (isConnected) {
      console.log("Connection established! Resolving all pending promises.");
      resolversRef.current.forEach((resolve) => resolve());
      resolversRef.current = [];
    }
  }, [isConnected]);

  const attemptConnection = useCallback(() => {
    console.log(`Attempting ${protocol} connection...`);
    if (isWebRtc) {
      if (isConnected || isConnecting) return;
      webrtcHandlePlayRef.current?.((stream: MediaStream) => {
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
      p2pHandlePlayRef.current?.(() => {
        if (p2pPlayer === "hls") {
          setStream(`${host}/hls/stream.m3u8`);
        } else {
          setStream(`${host}/stream`);
        }
      });
    }
  }, [
    isWebRtc,
    isConnected,
    isConnecting,
    protocol,
    webrtcHandlePlayRef,
    offereeRef,
    setStream,
    p2pHandlePlayRef,
    p2pPlayer,
    host,
  ]);

  const waitForConnection = useCallback(
    (timeoutMs: number) => {
      return new Promise<void>((resolve, reject) => {
        if (isConnected) {
          resolve();
          return;
        }

        const timeout = setTimeout(() => {
          resolversRef.current = resolversRef.current.filter(
            (r) => r !== wrappedResolve,
          );
          reject(new Error("Connection timeout"));
        }, timeoutMs);

        const wrappedResolve = () => {
          clearTimeout(timeout);
          resolve();
        };

        resolversRef.current.push(wrappedResolve);
      });
    },
    [isConnected],
  );

  useEffect(() => {
    let cancelled = false;

    async function connectionWorkflow() {
      const maxAttempts = 5;
      const defaultProtocol = protocol
      for (let attempt = 1; attempt <= maxAttempts; attempt++) {
        if (cancelled) return;

        if (attempt === 4) {
          const newProtocol = isWebRtc ? "P2P_PROTOCOL" : "WEBRTC_PROTOCOL";
          console.log(`Switching protocol to ${newProtocol}`);
          setProtocol(newProtocol);
          await new Promise((resolve) => setTimeout(resolve, 100));
        }

        try {
          console.log(
            `Connection attempt ${attempt}/${maxAttempts} with protocol: ${protocol}`,
          );
          setConnectionAttempts(attempt);
          attemptConnection();
          const timeout = 10000 + 1000 * (attempt - 1);
          await waitForConnection(timeout);
          if (!cancelled) {
            console.log(`Successfully connected on attempt ${attempt}`);
          }
          return;
        } catch (error) {
          if (cancelled) return;
          console.log(
            `Connection attempt ${attempt}/${maxAttempts} failed:`,
            error,
          );

          if (attempt === maxAttempts) {
            setProtocol(defaultProtocol);
            console.error("All connection attempts failed");
            showAlert("Can not connect to the server. Try again later");
          }
        }
      }
    }
    connectionWorkflow();

    return () => {
      cancelled = true;
      resolversRef.current = [];
    };
  }, [
    attemptConnection,
    waitForConnection,
    isWebRtc,
    protocol,
    setProtocol,
    showAlert,
  ]);

  useEffect(() => {
    if (isConnected) {
      attemptConnection();
    }
  }, [pathname, segments, attemptConnection, isConnected]);

  return null;
}
