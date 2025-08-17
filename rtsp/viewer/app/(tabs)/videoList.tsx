import { useFocusEffect } from '@react-navigation/native';
import React, { useCallback, useState } from 'react';
import {
  Button,
  Dimensions,
  StyleSheet,
  View
} from 'react-native';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';
import { useWebSocket } from '../websocketProvider';

const { width } = Dimensions.get('window');
const ITEM_MARGIN = 5;
const ITEMS_PER_ROW = 6;
const ITEM_WIDTH = (width - (ITEM_MARGIN * (ITEMS_PER_ROW + 1))) / ITEMS_PER_ROW;

export default () => {
  const { httpServerUrl } = useWebSocket();
  const [items, setItems] = useState([]);
  
  const fetchVideoList = async () => {
    try {
      const finalUrl = `${httpServerUrl}/video-list`;
      const response = await fetch(finalUrl, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });
      
      if (response.ok) {
        const data = await response.json();
        setItems(data??[]);
      }
    } catch (error) {
      console.error('Error fetching status:', error);
    }
  };

  useFocusEffect(
    useCallback(() => {
      fetchVideoList();
    }, [httpServerUrl])
  );

  const renderVideoItem = (item, index) => (
    <View key={index} style={styles.gridItem}>
      <Button 
        title={`${item.Name} (${item.Size ?? "-"})`}
        color="#007AFF"
        onPress={() => {
          // Add your button press handler here
          console.log('Selected:', item.Name);
        }}
      />
    </View>
  );

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.container}>
        <View style={styles.gridContainer}>
          {items.map((item, index) => renderVideoItem(item, index))}
        </View>
      </SafeAreaView>
    </SafeAreaProvider>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#464646fd',
  },
  gridContainer: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'space-around',
    paddingHorizontal: ITEM_MARGIN / 2,
    paddingTop: 10,
  },
  gridItem: {
    width: ITEM_WIDTH,
    marginBottom: ITEM_MARGIN,
    marginHorizontal: ITEM_MARGIN / 2,
  },
});