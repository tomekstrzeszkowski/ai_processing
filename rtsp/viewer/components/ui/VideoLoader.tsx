import { LinearGradient } from 'expo-linear-gradient';
import { useEffect, useRef, useState } from 'react';
import { Animated, StyleSheet, Text, View } from 'react-native';

const height = 800;
export const VideoLoader = () => {
  const shimmerAnim = useRef(new Animated.Value(0)).current;
  const [containerWidth, setContainerWidth] = useState(0);

  useEffect(() => {
    if (containerWidth === 0) return; // Wait until we have the width

    shimmerAnim.setValue(0);
    
    const animation = Animated.loop(
      Animated.sequence([
        Animated.timing(shimmerAnim, {
          toValue: 2,
          duration: 1900,
          useNativeDriver: true,
        }),
        Animated.timing(shimmerAnim, {
          toValue: 5,
          duration: 1000,
          useNativeDriver: true,
        }),
      ])
    );
    
    animation.start();

    return () => {
      animation.stop();
      shimmerAnim.setValue(0);
    };
  }, [shimmerAnim, containerWidth]);

  const translateX = shimmerAnim.interpolate({
    inputRange: [0, 1],
    outputRange: [-height - 200, containerWidth],
  });

  const AnimatedLinearGradient = Animated.createAnimatedComponent(LinearGradient);

  return (
    <View 
      style={styles.container}
      onLayout={(event) => {
        const { width } = event.nativeEvent.layout;
        setContainerWidth(width);
      }}
    >
      <AnimatedLinearGradient
        colors={['transparent', 'rgba(238, 238, 238, 0.1)', 'transparent']}
        start={{ x: 0, y: 0.3 }}
        end={{ x: 1, y: 0.7 }}
        style={[
          styles.shimmer,
          {
            transform: [
              { translateX },
              { rotate: '0deg' },
            ],
          },
        ]}
      />
      <View style={styles.textContainer}>
        <Text style={styles.text}>Waiting for stream...</Text>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    width: "100%",
    height: "100%",
    backgroundColor: "#1a1a1a",
    borderRadius: 8,
    overflow: "hidden",
    justifyContent: "center",
    alignItems: "center",
    minWidth: 10,
    minHeight: height,
  },
  shimmer: {
    position: "absolute",
    top: 0,
    left: 0,
    height: "100%",
    width: height,
  },
  textContainer: {
    display: "flex",
    alignSelf: "center",
  },
  text: {
    color: "#b9b9b9",
  },
});