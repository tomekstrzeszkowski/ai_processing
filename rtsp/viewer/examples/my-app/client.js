class WebSocketSignalingClient {
    constructor(userId, serverUrl = 'ws://localhost:7070/ws') {
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
        return new Promise((resolve, reject) => {
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
                    const message = JSON.parse(event.data);
                    const handlers = this.messageHandlers[message.type] || [];
                    handlers.forEach(handler => handler(message));
                } catch (error) {
                    console.error('Error parsing message:', error);
                }
            }); 
        });
    }

    send(message) {
        if (this.ws?.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        } else {
            console.error('WebSocket is not OPEN');
        }
    }

    sendOffer(offer) {
        this.send(offer);
    }

    sendAnswer(answer) {
        this.send(answer);
    }

    sendIceCandidates(candidates) {
        this.send({
            type: 'ice',
            ice: candidates
        });
    }

    onOffer(handler) {
        this.messageHandlers.offer.push(handler);
    }

    onAnswer(handler) {
        this.messageHandlers.answer.push(handler);
    }

    onIce(handler) {
        this.messageHandlers.ice.push(handler);
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
        }
    }
}