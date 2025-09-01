import React, { useState } from 'react';
import {
  Button,
  ScrollView,
  StyleSheet
} from 'react-native';

interface VideoPlayerProps {
  videoUrl: string;
  name: string;
  scrollView?: React.RefObject<ScrollView | null>;
}

export const VideoPlayer: React.FC<VideoPlayerProps> = ({ videoUrl, name, scrollView }) => {
  const [error, setError] = useState<string>("");

  if (!videoUrl) {
    return <div>No video selected</div>;
  }

  return (
    <div style={styles.container}>
      <Button 
        title={`${name} ⬆️`}
        color="#4CAF50"
        onPress={() => {
          scrollView?.current?.scrollTo({
          y: 0,
          animated: false
        });
        }}
      />

      {error ? (
        <div>{error}</div>
      ) : (
        <video 
          src={videoUrl} 
          controls 
          width={styles.videoSize.width}
          height={styles.videoSize.height}
          onError={(e) => {
            console.error('Video error:', e);
            setError('Error playing video');
          }}
        />
      )}
    </div>
  );
};

const styles = StyleSheet.create({
  videoSize: {
    width: '100%',
    height: '100%',
  },
  container: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'flex-start',
  },
});