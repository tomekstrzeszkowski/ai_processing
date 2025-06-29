import { createLibp2p } from 'libp2p';
import { webSockets } from '@libp2p/websockets';
import { noise } from '@chainsafe/libp2p-noise';
import { yamux } from '@chainsafe/libp2p-yamux';
import { mplex } from '@libp2p/mplex';
import { circuitRelayTransport } from '@libp2p/circuit-relay-v2';
import { kadDHT } from '@libp2p/kad-dht';
import { bootstrap } from '@libp2p/bootstrap';
import { identify } from '@libp2p/identify';
import { ping } from '@libp2p/ping';
import { multiaddr } from '@multiformats/multiaddr';
import { fromString, toString } from 'uint8arrays';
import { CID } from 'multiformats/cid';
import { sha256 } from 'multiformats/hashes/sha2';
import * as raw from 'multiformats/codecs/raw';

const PROTOCOL_ID = '/get-frame/1.0.0';
const RENDEZVOUS = 'tstrz-b-p2p-app-v1.0.0';

class P2PVideoClient {
    constructor() {
        this.node = null;
        this.isRunning = false;
        this.eventListeners = new Map();
        this.frameBuffer = [];
        this.connectedPeers = new Set();
        this.maxFrameBuffer = 30;
        this.discoveryInterval = null;
        
        // Bootstrap peers for DHT (same as Go version)
        this.bootstrapPeers = [
            '/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN',
            '/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa',
            '/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb',
            '/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt',
            '/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ',
            // Add circuit relay for browser compatibility
            '/ip4/147.75.83.83/tcp/4001/ws/p2p/QmbFMke1KXqnYyBBWxB74N4c5SBnJMVAiMNRcGu6x1AwQH',
        ];
    }

    // Simple event emitter functionality
    on(event, callback) {
        if (!this.eventListeners.has(event)) {
            this.eventListeners.set(event, []);
        }
        this.eventListeners.get(event).push(callback);
    }

    emit(event, data) {
        if (this.eventListeners.has(event)) {
            this.eventListeners.get(event).forEach(callback => callback(data));
        }
    }

    async start() {
        if (this.isRunning) return;
        
        this.emit('status', 'Creating libp2p node...');
        
        try {
            // Create libp2p node (equivalent to MakeEnhancedHost in Go)
            this.node = await createLibp2p({
                addresses: {
                    listen: []  // Browsers can't listen on ports, only make outbound connections
                },
                transports: [
                    webSockets(),
                    circuitRelayTransport({
                        discoverRelays: 1,
                        hop: {
                            enabled: true,
                            active: true
                        }
                    })
                ],
                connectionEncryption: [noise()],
                streamMuxers: [yamux(), mplex()],
                services: {
                    identify: identify(),
                    ping: ping(),
                    dht: kadDHT({
                        clientMode: false,
                        validators: {},
                        selectors: {}
                    }),
                    bootstrap: bootstrap({
                        list: this.bootstrapPeers
                    })
                }
            });

            await this.node.start();
            this.isRunning = true;
            
            this.setupEventHandlers();
            this.setupStreamHandler();
            
            const multiaddrs = this.node.getMultiaddrs();
            this.emit('status', `P2P node started. Listening on: ${multiaddrs.map(ma => ma.toString()).join(', ')}`);
            
            this.emit('status', 'Starting peer discovery...');
            
            // Start peer discovery
            this.startPeerDiscovery();
            
        } catch (error) {
            this.isRunning = false;
            throw error;
        }
    }

    async startPeerDiscovery() {
        try {
            this.emit('status', 'Starting peer discovery...');
            console.log('=== PEER DISCOVERY DEBUG ===');
            
            // Wait for DHT to be ready
            await new Promise(resolve => setTimeout(resolve, 3000));
            
            // Log our peer ID
            console.log('Our Peer ID:', this.node.peerId.toString());
            this.emit('status', `Our ID: ${this.node.peerId.toString().substring(0, 12)}...`);
            
            // Check DHT status
            try {
                console.log('DHT Mode:', this.node.services.dht.mode);
                this.emit('status', `DHT Mode: ${this.node.services.dht.mode}`);
            } catch (e) {
                console.log('DHT status check failed:', e);
            }
            
            // Try simple rendezvous via DHT
            try {
                console.log('Attempting DHT rendezvous...');
                this.emit('status', 'Attempting DHT rendezvous...');
                
                const rendezvousBytes = new TextEncoder().encode(RENDEZVOUS);
                const hash = await sha256.digest(rendezvousBytes);
                const rendezvousCID = CID.createV1(raw.code, hash);
                console.log('Rendezvous key:', RENDEZVOUS);
                console.log('Rendezvous CID:', rendezvousCID.toString());
                
                // Provide our service
                console.log('Announcing on DHT...');
                await this.node.services.dht.provide(rendezvousCID);
                console.log('DHT announce successful');
                this.emit('status', 'Announced on DHT successfully');
                
                // Find providers
                console.log('Searching for providers...');
                this.emit('status', 'Searching for Go backend...');
                
                const providers = this.node.services.dht.findProviders(rendezvousCID);
                let providerCount = 0;
                
                for await (const provider of providers) {
                    providerCount++;
                    console.log(`Provider ${providerCount}:`, provider);
                    
                    // Handle different provider formats
                    let peerId = null;
                    if (provider.id) {
                        peerId = provider.id;
                    } else if (provider.peer) {
                        peerId = provider.peer;
                    } else if (typeof provider === 'string') {
                        peerId = provider;
                    } else {
                        console.log('Unknown provider format:', provider);
                        continue;
                    }
                    
                    console.log('Provider peer ID:', peerId.toString());
                    
                    if (peerId.toString() !== this.node.peerId.toString()) {
                        this.emit('status', `Found provider: ${peerId.toString().substring(0, 12)}...`);
                        console.log('Attempting to dial provider...');
                        
                        try {
                            const connection = await this.node.dial(peerId);
                            console.log('Connection successful!', connection);
                            this.emit('status', 'Connected to Go backend!');
                            
                            // Check if this peer supports our protocol
                            try {
                                const protocols = await this.node.peerStore.get(peerId);
                                console.log('Peer protocols:', protocols.protocols);
                                
                                // Check if it supports our video protocol
                                if (protocols.protocols.includes(PROTOCOL_ID)) {
                                    console.log(' Found Go backend with video protocol!');
                                    this.emit('status', ' Connected to Go video backend!');
                                } else {
                                    console.log('Connected peer does not support video protocol');
                                    this.emit('status', 'Connected to peer (not video backend)');
                                }
                            } catch (protocolError) {
                                console.log('Could not check peer protocols:', protocolError.message);
                            }
                            
                            return; // Success!
                        } catch (dialError) {
                            console.log('Dial failed:', dialError.message);
                            this.emit('status', `Dial failed: ${dialError.message}`);
                        }
                    }
                }
                
                console.log(`Found ${providerCount} total providers`);
                if (providerCount === 0) {
                    this.emit('status', 'No providers found on DHT');
                }
                
            } catch (dhtError) {
                console.log('DHT operation failed:', dhtError);
                this.emit('status', `DHT failed: ${dhtError.message}`);
            }
            
            // Fallback: try connecting to any available peers
            this.emit('status', 'Trying direct peer connections...');
            const allPeers = this.node.getPeers();
            console.log('Available peers:', allPeers.length);
            
            for (const peerId of allPeers) {
                if (!this.connectedPeers.has(peerId.toString())) {
                    console.log('Trying to connect to peer:', peerId.toString().substring(0, 12));
                    try {
                        await this.node.dial(peerId);
                        console.log('Connected to peer:', peerId.toString().substring(0, 12));
                        this.emit('status', `Connected to peer: ${peerId.toString().substring(0, 12)}...`);
                    } catch (error) {
                        console.log('Failed to connect to peer:', error.message);
                    }
                }
            }
            
            // Set up periodic discovery
            this.discoveryInterval = setInterval(async () => {
                console.log('=== PERIODIC DISCOVERY ===');
                const peers = this.node.getPeers();
                console.log('Current peers:', peers.length);
                this.emit('status', `Network peers: ${peers.length}`);
                
                // Try DHT search again
                try {
                    const rendezvousBytes = new TextEncoder().encode(RENDEZVOUS);
                    const hash = await sha256.digest(rendezvousBytes);
                    const rendezvousCID = CID.createV1(raw.code, hash);
                    const providers = this.node.services.dht.findProviders(rendezvousCID);
                    
                    for await (const provider of providers) {
                        // Handle different provider formats
                        let peerId = null;
                        if (provider.id) {
                            peerId = provider.id;
                        } else if (provider.peer) {
                            peerId = provider.peer;
                        } else if (typeof provider === 'string') {
                            peerId = provider;
                        } else {
                            continue;
                        }
                        
                        if (peerId.toString() !== this.node.peerId.toString() && 
                            !this.connectedPeers.has(peerId.toString())) {
                            console.log('New provider found:', peerId.toString().substring(0, 12));
                            try {
                                await this.node.dial(peerId);
                                console.log('Connected to new provider!');
                                this.emit('status', 'Connected to Go backend!');
                            } catch (error) {
                                console.log('Failed to connect to new provider:', error.message);
                            }
                        }
                    }
                } catch (error) {
                    console.log('Periodic DHT search failed:', error.message);
                }
                
            }, 20000); // Every 20 seconds
            
        } catch (error) {
            console.error('Peer discovery error:', error);
            this.emit('status', `Discovery error: ${error.message}`);
        }
    }

    setupEventHandlers() {
        // Handle peer connections (equivalent to HandleConnectedPeers in Go)
        this.node.addEventListener('peer:connect', (evt) => {
            const peerId = evt.detail.toString();
            this.connectedPeers.add(peerId);
            this.emit('peer-connected', peerId);
            this.emit('status', `Connected to ${this.connectedPeers.size} peer(s)`);
        });

        this.node.addEventListener('peer:disconnect', (evt) => {
            const peerId = evt.detail.toString();
            this.connectedPeers.delete(peerId);
            this.emit('peer-disconnected', peerId);
            this.emit('status', `Connected to ${this.connectedPeers.size} peer(s)`);
        });
    }

    setupStreamHandler() {
        // Set up stream handler (equivalent to StartListening in Go)
        this.node.handle(PROTOCOL_ID, ({ stream }) => {
            this.handleIncomingStream(stream);
        });
    }

    async handleIncomingStream(stream) {
        try {
            // Send all buffered frames (equivalent to Go's stream handler)
            for (const frame of this.frameBuffer) {
                await stream.write(frame);
            }
            await stream.close();
            
            // Clear frame buffer after sending
            this.frameBuffer = [];
            
        } catch (error) {
            console.error('Error handling incoming stream:', error);
        }
    }

    async connectToPeer(peerInfo) {
        try {
            await this.node.dial(peerInfo.id);
        } catch (error) {
            console.error(`Failed to connect to peer ${peerInfo.id}:`, error);
        }
    }

    async requestFramesFromPeer(peerId) {
        try {
            const stream = await this.node.dialProtocol(peerId, PROTOCOL_ID);
            
            // Read frames from the stream
            const chunks = [];
            for await (const chunk of stream.source) {
                chunks.push(chunk.subarray());
            }
            
            if (chunks.length > 0) {
                // Combine chunks and split JPEG frames
                const allData = new Uint8Array(chunks.reduce((acc, chunk) => acc + chunk.length, 0));
                let offset = 0;
                for (const chunk of chunks) {
                    allData.set(chunk, offset);
                    offset += chunk.length;
                }
                
                const frames = this.splitJPEGFrames(allData);
                frames.forEach(frame => {
                    this.emit('frame-received', frame);
                });
            }
            
            await stream.close();
            
        } catch (error) {
            console.error(`Error requesting frames from peer ${peerId}:`, error);
        }
    }

    splitJPEGFrames(data) {
        const frames = [];
        let start = 0;

        for (let i = 0; i < data.length - 1; i++) {
            // Look for JPEG start marker (0xFF 0xD8)
            if (data[i] === 0xFF && data[i + 1] === 0xD8 && i > start) {
                frames.push(data.slice(start, i));
                start = i;
            }
        }

        // Add the last frame
        if (start < data.length) {
            frames.push(data.slice(start));
        }

        return frames;
    }

    // Equivalent to BroadcastFrame in Go
    broadcastFrame(frameData) {
        console.log('GOT FRAME!')
        this.frameBuffer.push(frameData);
        
        // Keep buffer size manageable
        if (this.frameBuffer.length > this.maxFrameBuffer) {
            this.frameBuffer.shift();
        }
    }

    async shareStream(stream) {
        // Start capturing frames from the stream
        this.startFrameCapture(stream);
        
        // Start requesting frames from connected peers
        this.startFrameRequesting();
    }

    startFrameCapture(stream) {
        const video = document.createElement('video');
        video.srcObject = stream;
        video.play();
        
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        
        const captureFrame = () => {
            if (!this.isRunning || !stream.active) return;
            
            canvas.width = video.videoWidth;
            canvas.height = video.videoHeight;
            ctx.drawImage(video, 0, 0);
            
            // Convert to JPEG and broadcast
            canvas.toBlob((blob) => {
                if (blob) {
                    const reader = new FileReader();
                    reader.onload = () => {
                        const frameData = new Uint8Array(reader.result);
                        this.broadcastFrame(frameData);
                    };
                    reader.readAsArrayBuffer(blob);
                }
            }, 'image/jpeg', 0.8);
            
            // Adaptive frame rate based on connected peers
            const frameRate = this.connectedPeers.size > 0 ? 100 : 200; // 10 FPS or 5 FPS
            setTimeout(captureFrame, frameRate);
        };
        
        video.addEventListener('loadedmetadata', captureFrame);
    }

    startFrameRequesting() {
        // Periodically request frames from connected peers
        const requestFrames = () => {
            if (!this.isRunning) return;
            
            for (const peerId of this.connectedPeers) {
                this.requestFramesFromPeer(peerId).catch(err => {
                    console.error(`Error requesting frames from ${peerId}:`, err);
                });
            }
            
            // Request frames every second
            setTimeout(requestFrames, 1000);
        };
        
        setTimeout(requestFrames, 1000);
    }

    async connectToPeer(peerIdString, multiaddr = null) {
        try {
            console.log('Attempting manual connection to:', peerIdString);
            this.emit('status', `Connecting to ${peerIdString.substring(0, 12)}...`);
            
            let connection;
            if (multiaddr) {
                connection = await this.node.dial(multiaddr);
            } else {
                connection = await this.node.dial(peerIdString);
            }
            
            console.log('Manual connection successful!', connection);
            this.emit('status', 'Manual connection successful!');
            
            // Check protocols
            try {
                const protocols = await this.node.peerStore.get(connection.remotePeer);
                console.log('Manual peer protocols:', protocols.protocols);
                
                if (protocols.protocols.includes(PROTOCOL_ID)) {
                    console.log('🎉 Manual connection found Go backend!');
                    this.emit('status', '🎉 Manual connection to Go backend!');
                } else {
                    console.log('Manual peer does not support video protocol');
                }
            } catch (protocolError) {
                console.log('Could not check manual peer protocols:', protocolError.message);
            }
            
            return connection;
        } catch (error) {
            console.log('Manual connection failed:', error.message);
            this.emit('status', `Manual connection failed: ${error.message}`);
            throw error;
        }
    }

    async stop() {
        if (!this.isRunning) return;
        
        this.isRunning = false;
        this.connectedPeers.clear();
        this.frameBuffer = [];
        
        if (this.node) {
            await this.node.stop();
            this.node = null;
        }
        
        if (this.discoveryInterval) {
            clearInterval(this.discoveryInterval);
            this.discoveryInterval = null;
        }
        
        this.emit('status', 'Stopped');
    }

    getConnectedPeers() {
        return Array.from(this.connectedPeers);
    }

    getFrames() {
        return this.frameBuffer;
    }
}

export default P2PVideoClient;
