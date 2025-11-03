import { VideoPlayer } from '@/components/VideoPlayer';
import { useFocusEffect } from '@react-navigation/native';
// No external converter needed
import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  Button,
  Dimensions,
  ScrollView,
  StyleSheet,
  View,
} from 'react-native';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';
import { useWebSocket } from '../websocketProvider';

const { width } = Dimensions.get('window');
const ITEM_MARGIN = 5;
const ITEMS_PER_ROW = 6;
const ITEM_WIDTH = (width - (ITEM_MARGIN * (ITEMS_PER_ROW + 1))) / ITEMS_PER_ROW;



export default () => {

  const renderVideoItem = (item: any, index: number) => (
    <View key={index} style={styles.gridItem}>
      <Button 
        title={`${item.Name} (${item.Size ?? "-"})`}
        color="#007AFF"
      />
    </View>
  );

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.container}>
        WebRTC, P2P
      </SafeAreaView>
    </SafeAreaProvider>
  );
};

const styles = StyleSheet.create({
    container: {
      flex: 1,
      backgroundColor: '#464646fd',
    },
});