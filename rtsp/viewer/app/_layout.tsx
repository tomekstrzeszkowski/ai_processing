import { DarkTheme, DefaultTheme, ThemeProvider } from '@react-navigation/native';
import { useFonts } from 'expo-font';
import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import 'react-native-reanimated';

import { useColorScheme } from '@/hooks/useColorScheme';

import { ConnectionProvider } from '@/app/connectionProvider';
import { HttpProvider } from '@/app/httpProvider';
import { NavigationListener } from '@/app/navigationListener';
import { P2pProvider } from '@/app/p2pProvider';
import { ProtocolProvider } from '@/app/protocolProvider';
import { ToastProvider } from '@/app/toastProvider';
import { WebRtcProvider } from '@/app/webRtcProvider';
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
            <P2pProvider>
              <ThemeProvider value={colorScheme === 'dark' ? DarkTheme : DefaultTheme}>
                <ConnectionProvider>
                  <NavigationListener />
                  <Stack>
                    <Stack.Screen name="(tabs)" options={{ headerShown: false }} />
                    <Stack.Screen name="+not-found" />
                  </Stack>
                  <StatusBar style="auto" />
                </ConnectionProvider>
              </ThemeProvider>
            </P2pProvider>
          </WebRtcProvider>
        </HttpProvider>
     </ProtocolProvider>
    </ToastProvider>
  );
}
