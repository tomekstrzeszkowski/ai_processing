import { SignalingMessage } from "./message";


export class WebSocketSignalingClient {
    userId: number;
    serverUrl: string;
    ws: WebSocket | null;
    messageHandlers: {
        offer: Array<(message: SignalingMessage) => void>;
        ice: Array<(message: SignalingMessage) => void>;
        answer: Array<(message: SignalingMessage) => void>;
    };
    constructor(userId: number, serverUrl: string) {
        this.userId = userId;
        this.serverUrl = `${serverUrl}?userId=${userId}`;
        this.ws = null;
        this.messageHandlers = {
            offer: [],
            ice: [],
            answer: [],
        };
    }

    connect() {
        return new Promise<void>((resolve, reject) => {
            this.ws = new WebSocket(this.serverUrl);
            this.ws.addEventListener("open", () => {
                console.log('WebSocket connected');
                resolve();
            });
            this.ws.addEventListener("error", error => {
                console.error('WebSocket error', error);
                reject();
            });
            this.ws.addEventListener("close", () => {
                console.log('WebSocket disconnected');
            });
            this.ws.addEventListener("message", event => {
                try {
                    const message: SignalingMessage = JSON.parse(event.data);
                    const messageType = message.type as keyof typeof this.messageHandlers;
                    const handlers = this.messageHandlers[messageType] || [];
                    handlers.forEach(handler => handler(message));
                } catch (error) {
                    console.error('Error parsing message:', error);
                }
            }); 
        });
    }

    send(message: SignalingMessage) {
        if (this.ws?.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        } else {
            console.error('WebSocket is not OPEN');
        }
    }

    sendOffer(offer: SignalingMessage) {
        this.send(offer);
    }

    sendAnswer(answer: SignalingMessage) {
        this.send(answer);
    }

    sendIceCandidates(candidates: RTCIceCandidate[]) {
        this.send({
            type: 'ice',
            ice: candidates
        });
    }

    onOffer(handler: (message: SignalingMessage) => void) {
        this.messageHandlers.offer.push(handler);
    }

    onAnswer(handler: (message: SignalingMessage) => void) {
        this.messageHandlers.answer.push(handler);
    }

    onIce(handler: (message: SignalingMessage) => void) {
        this.messageHandlers.ice.push(handler);
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
        }
    }
}