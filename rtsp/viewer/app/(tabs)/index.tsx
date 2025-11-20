import { useProtocol } from '@/app/protocolProvider';
import { LiveVideoPlayer } from '@/components/LiveVideoPlayer';
import React from 'react';
import {
  StyleSheet,
  Text,
  View
} from 'react-native';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';

const App = () => {
  const { 
    lastFrameTime,
    isConnected,
    stream
  } = useProtocol();

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.container}>
        <LiveVideoPlayer stream={stream} isConnected={isConnected} />
        <View>
          {lastFrameTime && (
            <View style={{ flex: 1, color: "white", padding: 20}}>
              <Text style={{color: "white"}}>Last Signal:</Text>
              <Text style={{color: "white"}}>{lastFrameTime}</Text>
            </View>
          )}
        </View>
      </SafeAreaView>
    </SafeAreaProvider>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#1a1a1a',
    color: "#b9b9b9ff",
    overflow: "auto"
  },
  connectionContainer: {
    padding: 20,
    borderBottomWidth: 1,
    borderBottomColor: '#333',
  },
  button: {
    padding: 15,
    borderRadius: 8,
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: 50,
  },
  connectButton: {
    backgroundColor: '#4CAF50',
  },
  disconnectButton: {
    backgroundColor: '#f44336',
  },
});

export default App;