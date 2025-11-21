import { useProtocol } from "@/app/protocolProvider";
import { LiveVideoPlayer } from "@/components/LiveVideoPlayer";
import React from "react";
import { Text, View } from "react-native";
import { SafeAreaProvider, SafeAreaView } from "react-native-safe-area-context";

export default function app() {
  const { lastFrameTime, isConnected, stream } = useProtocol();

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
          }}
        >
          <LiveVideoPlayer stream={stream} isConnected={isConnected} />
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
