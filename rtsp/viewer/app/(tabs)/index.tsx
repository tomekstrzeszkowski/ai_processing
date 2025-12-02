import { useProtocol } from "@/app/protocolProvider";
import { VideoPlayer } from "@/components/VideoPlayer";
import React, { useState } from "react";
import { Text, View } from "react-native";
import { SafeAreaProvider, SafeAreaView } from "react-native-safe-area-context";

export default function app() {
  const { lastFrameTime, isConnected, stream } = useProtocol();
  const [isPlaying, setIsPlaying] = useState<boolean>(true);

  return (
    <SafeAreaProvider>
      <SafeAreaView
        style={{
          flex: 1,
          backgroundColor: "#1a1a1a",
          flexDirection: "row",
        }}
      >
        <View
          style={{
            flex: 30,
            marginTop: "auto",
            marginBottom: "auto"
          }}
        >
          <VideoPlayer 
            stream={stream} 
            isConnected={isConnected} 
            isLive={true}
            isPlaying={isPlaying}
            handlePlay={(video: HTMLVideoElement) => {
              console.log('video play', video)
              video.play()
              setIsPlaying(true);
            }}
            handlePause={(video: HTMLVideoElement) => {
              console.log('video pause', video)
              video.pause()
              setIsPlaying(false);
            }}
          />
          {lastFrameTime && (
            <View
              style={{
                padding: 20,
              }}
            >
              <Text
                style={{
                  color: "#b9b9b9ff",
                }}
              >
                Last Signal:
              </Text>
              <Text
                style={{
                  color: "#b9b9b9ff",
                }}
              >
                {lastFrameTime}
              </Text>
            </View>
          )}
        </View>

        <View
          style={{
            flex: 1,
            display: "none",
          }}
        >
          <Text>Timeline</Text>
        </View>
      </SafeAreaView>
    </SafeAreaProvider>
  );
}
