export interface SignalingMessage {
    type: string;
    sdp?: string;
    ice?: RTCIceCandidate[];
}