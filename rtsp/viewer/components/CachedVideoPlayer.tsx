import React, { useEffect, useRef, useState } from 'react';
import { Animated, Easing, Image, View } from 'react-native';

interface CachedVideoPlayerProps {
  imageUri: string | null;
  frameCountRef: React.RefObject<number>;
  styles: any;
}

export const CachedVideoPlayer: React.FC<CachedVideoPlayerProps> = ({ imageUri, frameCountRef, styles }) => {
  const [displayUri, setDisplayUri] = useState<string | null>(imageUri);
  const [pendingUri, setPendingUri] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const fadeAnim = useRef<Animated.Value>(new Animated.Value(1));

  useEffect(() => {
    if (imageUri && imageUri !== displayUri) {
      setPendingUri(imageUri);
      setIsLoading(true);
    }
  }, [imageUri, displayUri]);

  const handleLoad = () => {
    if (pendingUri) {
      // Start crossfade animation
      fadeAnim.current.setValue(0);
      setIsLoading(false);
      Animated.timing(fadeAnim.current, {
        toValue: 1,
        duration: 200,
        easing: Easing.linear,
        useNativeDriver: true,
      }).start(() => {
        setDisplayUri(pendingUri);
        setPendingUri(null);
      });
    }
  };

  const handleError = (error: any) => {
    setPendingUri(null);
    setIsLoading(false);
  };

  // If no displayUri, show placeholder
  if (!displayUri && !pendingUri) {
    return (
      <View style={styles.noVideoContainer}>
        <Image
          source={{ uri: 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTEyIDJMMTMuMDkgOC4yNkwyMCA5TDEzLjA5IDE1Ljc0TDEyIDIyTDEwLjkxIDE1Ljc0TDQgOUwxMC45MSA4LjI2TDEyIDJaIiBmaWxsPSIjNjY2Ii8+Cjwvc3ZnPgo=' }}
          style={styles.placeholderIcon}
        />
      </View>
    );
  }

  return (
    <View style={styles.videoContainer}>
      {/* Current frame */}
      {displayUri && (
        <Image
          source={{ uri: displayUri }}
          style={[styles.video]}
          resizeMode="contain"
          key={displayUri}
        />
      )}
      {/* Crossfade to next frame */}
      {pendingUri && (
        <Animated.Image
          source={{ uri: pendingUri }}
          style={[styles.video, { position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, opacity: fadeAnim.current }]}
          resizeMode="contain"
          onLoad={handleLoad}
          onError={handleError}
          key={pendingUri}
        />
      )}
    </View>
  );
};
