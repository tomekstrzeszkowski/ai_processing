
import { useProtocol } from '@/app/protocolProvider';
import { Picker } from '@react-native-picker/picker';
import React from 'react';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';



export default () => {
  const { setProtocol, protocol } = useProtocol();
  return (
    <SafeAreaProvider style={{backgroundColor: "#1a1a1a", color: "#b9b9b9ff"}}>
      <SafeAreaView>
        <div style={{ display: 'flex', padding: 20, alignItems: 'center' }}>
          <div>Protocol: </div>
          <Picker
            selectedValue={protocol}
            onValueChange={(itemValue, itemIndex) =>
              setProtocol(itemValue)
            }>
            <Picker.Item label="WebRTC" value="WEBRTC_PROTOCOL" />
            <Picker.Item label="P2P" value="P2P_PROTOCOL" />
          </Picker>
        </div>
      </SafeAreaView>
    </SafeAreaProvider>
  );
};
