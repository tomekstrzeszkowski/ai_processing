
import { useProtocol } from '@/app/protocolProvider';
import { Picker } from '@react-native-picker/picker';
import React from 'react';
import {
  Text,
  View
} from 'react-native';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';


export default () => {
  const { setProtocol, protocol, p2pPlayer, setP2pPlayer } = useProtocol();
  return (
    <SafeAreaProvider style={{backgroundColor: "#1a1a1a", color: "#b9b9b9ff"}}>
      <SafeAreaView>
        <View style={{ display: 'flex', padding: 20, alignItems: 'center' }}>
          <View><Text style={{color: "#b9b9b9ff"}}>Protocol: </Text></View>
          <Picker
            selectedValue={protocol}
            onValueChange={(itemValue, _) =>
              setProtocol(itemValue)
            }>
            <Picker.Item label="WebRTC" value="WEBRTC_PROTOCOL" />
            <Picker.Item label="P2P" value="P2P_PROTOCOL" />
          </Picker>
        </View>
        {protocol === 'P2P_PROTOCOL' && (<View style={{ display: 'flex', padding: 20, alignItems: 'center' }}>
          <View><Text style={{color: "#b9b9b9ff"}}>Player: </Text></View>
          <Picker
            selectedValue={p2pPlayer}
            onValueChange={(itemValue, _) =>
              setP2pPlayer(itemValue)
            }>
            <Picker.Item label="Video" value="hls" />
            <Picker.Item label="Image" value="image" />
          </Picker>
        </View>)}
      </SafeAreaView>
    </SafeAreaProvider>
  );
};
