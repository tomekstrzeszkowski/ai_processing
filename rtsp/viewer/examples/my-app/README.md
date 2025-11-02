# Testing

Serve the stream from the webcam. The application can be tested in two different tabs. A signaling server is required; you can find it in the broadcaster app. In one tab, click Receive, then in the other, click Send and grant the browser permission to access your camera. You should see the stream sent directly from the peer.

## Common Issues

In Firefox, you may encounter issues when testing on a local network. To fix this, go to about:config and set media.peerconnection.ice.loopback=true.