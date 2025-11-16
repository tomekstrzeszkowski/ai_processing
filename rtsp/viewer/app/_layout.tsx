import { DarkTheme, DefaultTheme, ThemeProvider } from '@react-navigation/native';
import { useFonts } from 'expo-font';
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import 'react-native-reanimated';

import { useColorScheme } from '@/hooks/useColorScheme';

import { HttpProvider } from '@/app/httpProvider';
import { ProtocolProvider } from '@/app/protocolProvider';
import { ToastProvider } from '@/app/toastProvider';
import { WebRtcProvider } from '@/app/webRtcProvider';
import { WebSocketProvider } from '@/app/websocketProvider';
import {
  Text,
  View
} from 'react-native';

export default function RootLayout() {
  const colorScheme = useColorScheme();
  const [loaded] = useFonts({
    SpaceMono: require('../assets/fonts/SpaceMono-Regular.ttf'),
  });

  if (!loaded) {
    return (
      <View><Text>Still Loading...</Text></View>
    );
  }

  return (
    <ToastProvider>
      <ProtocolProvider>
        <HttpProvider>
          <WebRtcProvider>
            <WebSocketProvider>
              <ThemeProvider value={colorScheme === 'dark' ? DarkTheme : DefaultTheme}>
                <Stack>
                  <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
                  <Stack.Screen name="+not-found" />
                </Stack>
                <StatusBar style="auto" />
              </ThemeProvider>
            </WebSocketProvider>
          </WebRtcProvider>
        </HttpProvider>
     </ProtocolProvider>
    </ToastProvider>
  );
}
