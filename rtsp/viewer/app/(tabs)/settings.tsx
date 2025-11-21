import { useP2p } from "@/app/p2pProvider";
import { useProtocol } from "@/app/protocolProvider";
import { useWebRtc } from "@/app/webRtcProvider";
import { useToast } from "@/app/toastProvider";
import { Picker } from "@react-native-picker/picker";
import React from "react";
import {
  ActivityIndicator,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { SafeAreaProvider, SafeAreaView } from "react-native-safe-area-context";

export default function settings() {
  const { showAlert } = useToast();
  const {
    setProtocol,
    protocol,
    isWebRtc,
    p2pPlayer,
    setP2pPlayer,
    isConnecting,
    setIsConnecting,
    isConnected,
  } = useProtocol();
  const { handlePlayRef: p2pHandlePlayRef, handleStopRef: p2pHandleStopRef } =
    useP2p();
  const {
    handlePlayRef: webrtcHandlePlayRef,
    handleStopRef: webrtcHandleStopRef,
    offereeRef,
  } = useWebRtc();

  const connect = async () => {
    if (isConnecting || isConnected) return;
    setIsConnecting(true);
    if (isWebRtc) {
      await webrtcHandlePlayRef.current((stream: MediaStream) => {});
    } else {
      await p2pHandlePlayRef.current();
    }
    if (!isConnecting && !isConnected) {
      showAlert("Can not connect to the server. Try again later");
    }
  };

  const disconnect = () => {
    if (isWebRtc) {
      webrtcHandleStopRef.current();
    } else {
      p2pHandleStopRef.current();
    }
  };
  const styles = StyleSheet.create({
    container: {
      flex: 1,
      backgroundColor: "#1a1a1a",
      color: "#b9b9b9ff",
      overflow: "auto",
    },
    connectionContainer: {
      padding: 20,
      borderBottomWidth: 1,
      borderBottomColor: "#333",
    },
    button: {
      padding: 15,
      borderRadius: 8,
      alignItems: "center",
      justifyContent: "center",
      minHeight: 50,
    },
    connectButton: {
      backgroundColor: "#4CAF50",
    },
    disconnectButton: {
      backgroundColor: "#f44336",
    },
  });
  return (
    <SafeAreaProvider
      style={{ backgroundColor: "#1a1a1a", color: "#b9b9b9ff" }}
    >
      <SafeAreaView>
        <View style={{ display: "flex", padding: 20, alignItems: "center" }}>
          <View>
            <Text style={{ color: "#b9b9b9ff" }}>Protocol: </Text>
          </View>
          <Picker
            selectedValue={protocol}
            onValueChange={(itemValue, _) => setProtocol(itemValue)}
          >
            <Picker.Item label="WebRTC" value="WEBRTC_PROTOCOL" />
            <Picker.Item label="P2P" value="P2P_PROTOCOL" />
          </Picker>
        </View>
        {protocol === "P2P_PROTOCOL" && (
          <View style={{ display: "flex", padding: 20, alignItems: "center" }}>
            <View>
              <Text style={{ color: "#b9b9b9ff" }}>Player: </Text>
            </View>
            <Picker
              selectedValue={p2pPlayer}
              onValueChange={(itemValue, _) => setP2pPlayer(itemValue)}
            >
              <Picker.Item label="Video" value="hls" />
              <Picker.Item label="Image" value="image" />
            </Picker>
          </View>
        )}
        <View style={styles.connectionContainer}>
          <TouchableOpacity
            style={[
              styles.button,
              isConnected ? styles.disconnectButton : styles.connectButton,
            ]}
            onPress={isConnected ? disconnect : connect}
            disabled={isConnecting}
          >
            {isConnecting ? (
              <ActivityIndicator color="#fff" size="small" />
            ) : (
              <Text style={{ color: "#fff", fontWeight: 700 }}>
                {isConnected ? "Disconnect" : "Connect"}
              </Text>
            )}
          </TouchableOpacity>
        </View>
      </SafeAreaView>
    </SafeAreaProvider>
  );
}
