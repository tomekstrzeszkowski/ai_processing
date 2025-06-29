import { CapacitorConfig } from '@capacitor/cli';

const config = {
  appId: 'com.p2pvideo.client',
  appName: 'P2P Video Client',
  webDir: 'dist',
  server: {
    androidScheme: 'https'
  },
  plugins: {
    Camera: {
      permissions: ['camera', 'photos']
    },
    Microphone: {
      permissions: ['microphone']
    }
  }
};

export default config;
