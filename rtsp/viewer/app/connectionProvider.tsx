import { useProtocol } from "@/app/protocolProvider";
import { createContext, useEffect, useRef, useState } from "react";
import { Animated } from "react-native";

type Props = {
  children: React.ReactNode;
};

type ConnectionContextType = {};
const ConnectionContext = createContext<ConnectionContextType | null>(null);

export function ConnectionProvider({ children }: Props) {
  const { isConnected, isConnecting } = useProtocol();
  const [shouldDisplay, setShouldDisplay] = useState(true);
  const pulseAnim = useRef(new Animated.Value(1)).current;
  const fadeAnim = useRef(new Animated.Value(1)).current;
  const opacityAnim = useRef(new Animated.Value(1)).current;

  useEffect(() => {
    if (isConnecting) {
      setShouldDisplay(true);
      fadeAnim.setValue(1);
      Animated.loop(
        Animated.sequence([
          Animated.timing(pulseAnim, {
            toValue: 1.1,
            duration: 800,
            useNativeDriver: true,
          }),
          Animated.timing(pulseAnim, {
            toValue: 1,
            duration: 800,
            useNativeDriver: true,
          }),
        ]),
      ).start();
      Animated.loop(
        Animated.sequence([
          Animated.timing(opacityAnim, {
            toValue: 0.5,
            duration: 800,
            useNativeDriver: true,
          }),
          Animated.timing(opacityAnim, {
            toValue: 1,
            duration: 800,
            useNativeDriver: true,
          }),
        ]),
      ).start();
    } else if (isConnected) {
      pulseAnim.stopAnimation();
      opacityAnim.stopAnimation();
      pulseAnim.setValue(1);
      opacityAnim.setValue(1);

      setTimeout(() => {
        Animated.timing(fadeAnim, {
          toValue: 0,
          duration: 500,
          useNativeDriver: true,
        }).start(() => {
          setShouldDisplay(false);
        });
      }, 2000);
    } else {
      setShouldDisplay(true);
      pulseAnim.stopAnimation();
      opacityAnim.stopAnimation();
      pulseAnim.setValue(1);
      opacityAnim.setValue(1);
      fadeAnim.setValue(1);
    }
  }, [isConnecting, isConnected]);

  return (
    <ConnectionContext.Provider value={{}}>
      {shouldDisplay && (
        <Animated.View
          style={{
            borderWidth: 1,
            borderColor: isConnected ? "#4CAF50" : "#f44336",
            transform: [{ scale: pulseAnim }],
            opacity: isConnecting ? opacityAnim : fadeAnim,
          }}
        />
      )}
      {children}
    </ConnectionContext.Provider>
  );
}
