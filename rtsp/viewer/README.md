# Go connection issue

WIP

# P2P Video Client

A JavaScript-based peer-to-peer video streaming client ported from Go, designed for mobile deployment using Capacitor.

## Features

- 🔗 **P2P Connectivity**: Direct peer-to-peer connections using js-libp2p (same as Go version)
- 📱 **Mobile Ready**: Optimized for mobile devices with Capacitor
- 🎥 **Video Streaming**: Real-time JPEG frame sharing between peers
- 🔍 **Peer Discovery**: Automatic mDNS peer discovery (same protocol as Go version)
- 📺 **Screen Sharing**: Share your screen with connected peers
- 🎨 **Modern UI**: Beautiful, responsive interface with dark mode support

## Architecture

This client is a direct port of the Go libp2p implementation:
- **js-libp2p** for P2P connections (equivalent to Go's libp2p)
- **mDNS** for peer discovery (same as Go version)
- **Custom protocol** `/get-frame/1.0.0` for frame streaming
- **Canvas API** for JPEG frame processing
- **Capacitor** for mobile deployment

The JavaScript version maintains the same P2P architecture as the Go version:
- Uses the same bootstrap peers for DHT
- Same mDNS service tag: `tstrz-b-p2p-app-v1.0.0`
- Same protocol ID: `/get-frame/1.0.0`
- Same frame buffering and broadcasting logic

## Quick Start

### Web Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build
```

### Mobile Deployment

```bash
# Initialize Capacitor
npm run mobile:init

# Add mobile platforms
npm run mobile:add-ios     # For iOS
npm run mobile:add-android # For Android

# Build and sync
npm run mobile:build

# Open in native IDEs
npm run mobile:open-ios     # Opens Xcode
npm run mobile:open-android # Opens Android Studio
```

## Usage

1. **Start P2P Discovery**: Click "Start P2P Discovery" to begin looking for peers
2. **Share Screen**: Click "Share Screen" to broadcast your screen to connected peers
3. **View Connections**: Monitor connected peers in the peer list
4. **View Logs**: Check the logs section for connection status and debugging info

## Peer Discovery

Currently uses mDNS for peer discovery.

## Mobile Considerations

### iOS
- Requires camera/microphone permissions in Info.plist
- WebRTC works well in WKWebView (Capacitor's default)
- Background processing limitations apply

### Android
- Requires CAMERA and RECORD_AUDIO permissions
- WebRTC supported in modern WebView versions
- Consider battery optimization settings

## Production Deployment

For production use, you'll need to:

1. **Set up a signaling server** (replace localStorage discovery)
2. **Configure STUN/TURN servers** for NAT traversal
3. **Implement proper authentication** and security measures
4. **Add error handling** and reconnection logic
5. **Optimize for mobile networks** (adaptive bitrate, etc.)

## Comparison with Go Version

| Feature | Go Version | JavaScript Version |
|---------|------------|-------------------|
| P2P Library | libp2p | js-libp2p |
| Peer Discovery | mDNS | mDNS |
| Video Processing | Go routines | Canvas API |
| Mobile Support | None | Capacitor |
| WebSocket Server | Built-in | Not needed |

## Development Notes

- The current peer discovery uses mDNS for simplicity
- In production, implement a proper signaling server
- Frame processing is done client-side using Canvas API
- Mobile permissions are handled by Capacitor plugins

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test on both web and mobile
5. Submit a pull request

## License

MIT License - see LICENSE file for details
