import P2PVideoClient from './p2p-client.js';
import './styles.css';

class App {
    constructor() {
        this.client = new P2PVideoClient();
        this.initializeElements();
        this.setupEventListeners();
        this.setupClientEventListeners();
    }

    initializeElements() {
        this.elements = {
            startBtn: document.getElementById('startBtn'),
            stopBtn: document.getElementById('stopBtn'),
            shareBtn: document.getElementById('shareBtn'),
            status: document.getElementById('status'),
            localVideo: document.getElementById('localVideo'),
            remoteCanvas: document.getElementById('remoteCanvas'),
            peerList: document.getElementById('peerList'),
            logContainer: document.getElementById('logContainer')
        };
    }

    setupEventListeners() {
        this.elements.startBtn.addEventListener('click', () => this.startP2P());
        this.elements.stopBtn.addEventListener('click', () => this.stopP2P());
        this.elements.shareBtn.addEventListener('click', () => this.shareScreen());
    }

    setupClientEventListeners() {
        this.client.on('status', (status) => {
            this.updateStatus(status);
        });

        this.client.on('peer-connected', (peerId) => {
            this.log(`Peer connected: ${peerId}`);
            this.updatePeerList();
        });

        this.client.on('peer-disconnected', (peerId) => {
            this.log(`Peer disconnected: ${peerId}`);
            this.updatePeerList();
        });

        this.client.on('frame-received', (frameData) => {
            this.displayFrame(frameData);
        });

        this.client.on('error', (error) => {
            this.log(`Error: ${error.message}`, 'error');
        });
    }

    async startP2P() {
        try {
            this.elements.startBtn.disabled = true;
            this.log('Starting P2P discovery...');
            
            await this.client.start();
            
            this.elements.stopBtn.disabled = false;
            this.elements.shareBtn.disabled = false;
            this.log('P2P client started successfully');
        } catch (error) {
            this.log(`Failed to start P2P client: ${error.message}`, 'error');
            this.elements.startBtn.disabled = false;
        }
    }

    async stopP2P() {
        try {
            this.log('Stopping P2P client...');
            await this.client.stop();
            
            this.elements.startBtn.disabled = false;
            this.elements.stopBtn.disabled = true;
            this.elements.shareBtn.disabled = true;
            
            this.updateStatus('Stopped');
            this.log('P2P client stopped');
        } catch (error) {
            this.log(`Error stopping P2P client: ${error.message}`, 'error');
        }
    }

    async shareScreen() {
        try {
            this.log('Starting screen share...');
            const stream = await navigator.mediaDevices.getDisplayMedia({
                video: { mediaSource: 'screen' },
                audio: true
            });
            
            this.elements.localVideo.srcObject = stream;
            await this.client.shareStream(stream);
            
            this.log('Screen sharing started');
        } catch (error) {
            this.log(`Screen share failed: ${error.message}`, 'error');
        }
    }

    updateStatus(status) {
        this.elements.status.textContent = status;
    }

    updatePeerList() {
        const peers = this.client.getConnectedPeers();
        if (peers.length === 0) {
            this.elements.peerList.innerHTML = '<div>No peers connected</div>';
        } else {
            this.elements.peerList.innerHTML = peers
                .map(peer => `<div class="peer-item">${peer}</div>`)
                .join('');
        }
    }

    displayFrame(frameData) {
        const canvas = this.elements.remoteCanvas;
        const ctx = canvas.getContext('2d');
        
        // Create image from frame data
        const img = new Image();
        img.onload = () => {
            canvas.width = img.width;
            canvas.height = img.height;
            ctx.drawImage(img, 0, 0);
        };
        
        // Convert frame data to blob URL
        const blob = new Blob([frameData], { type: 'image/jpeg' });
        img.src = URL.createObjectURL(blob);
    }

    log(message, type = 'info') {
        const timestamp = new Date().toLocaleTimeString();
        const logEntry = document.createElement('div');
        logEntry.className = 'log-entry';
        logEntry.innerHTML = `<span class="log-timestamp">[${timestamp}]</span>${message}`;
        
        if (type === 'error') {
            logEntry.style.color = '#fc8181';
        }
        
        this.elements.logContainer.appendChild(logEntry);
        this.elements.logContainer.scrollTop = this.elements.logContainer.scrollHeight;
        
        // Keep only last 100 log entries
        while (this.elements.logContainer.children.length > 100) {
            this.elements.logContainer.removeChild(this.elements.logContainer.firstChild);
        }
    }
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    const app = new App();
    // Expose for testing
    window.app = app;
    window.p2pClient = app.client;
});
