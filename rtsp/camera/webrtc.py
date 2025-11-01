import asyncio
import cv2
import json
import numpy as np
from aiortc import RTCPeerConnection, RTCSessionDescription, VideoStreamTrack, RTCConfiguration, RTCIceServer, RTCIceCandidate
from aiortc.contrib.signaling import TcpSocketSignaling
from av import VideoFrame
import fractions
import os
from dotenv import load_dotenv
from saver import write_frame_to_shared_memory, VideoSaver, read_frame_from_shared_memory
import websockets
from PIL import Image
from io import BytesIO


load_dotenv()

ICE_SERVERS = (
    RTCIceServer(urls=[
        "stun:stun.l.google.com:19302",
        "stun:stun2.l.google.com:19302",
        "stun:stun3.l.google.com:19302",
        'stun:stun.1und1.de:3478',
        'stun:stun.avigora.com:3478',
        'stun:stun.avigora.fr:3478',
    ]),
    RTCIceServer(
        urls="turn:global.turn.twilio.com:3478?transport=udp",
        username="dc2d2894d5a9023620c467b0e71cfa6a35457e6679785ed6ae9856fe5bdfa269",
        credential="tE2DajzSJwnsSbc123"
    ),
    RTCIceServer(
        urls=['turn:openrelay.metered.ca:80', 'turn:openrelay.metered.ca:443'],
        username='openrelayproject',
        credential='openrelayproject'
    ),
    RTCIceServer(
        urls='turn:openrelay.metered.ca:443?transport=tcp',
        username='openrelayproject',
        credential='openrelayproject'
    ),
)

class ShmVideoStreamTrack(VideoStreamTrack):
    frame_count: int = 0
    _current_frame: np.ndarray | None = None

    def read_frame(self):
        data, type_ = read_frame_from_shared_memory()
        self._current_frame = data

    async def recv(self):
        self.frame_count += 1
        img = Image.open(BytesIO(self._current_frame))
        video_frame = VideoFrame.from_image(img)
        video_frame.pts = self.frame_count
        video_frame.time_base = fractions.Fraction(1, 30)
        return video_frame


async def listen(websocket, pc):
    try:
        async for message in websocket:
            message = json.loads(message)
            match message["type"]:
                case "offer":
                    pass
                case "answer":
                    await pc.setRemoteDescription(RTCSessionDescription(**message))
                case "ice":
                    for ice_data in message["ice"]:
                        candidate = RTCIceCandidate.from_sdp(ice_data["candidate"])
                        candidate.sdpMid = ice_data["sdpMid"]
                        candidate.sdpMLineIndex = ice_data["sdpMLineIndex"]
                        await pc.addIceCandidate(candidate)
    except websockets.exceptions.ConnectionClosed:
        print("Connection closed")

async def main():
    config = RTCConfiguration(iceServers=ICE_SERVERS)
    pc = RTCPeerConnection(config)
    video_track = ShmVideoStreamTrack()

    async with websockets.connect("ws://localhost:8080/ws?userId=53") as websocket:
        listener_task = asyncio.create_task(listen(websocket, pc))
        try:
            pc.addTrack(video_track)
            @pc.on("connectionstatechange")
            async def on_connectionstatechange():
                print(f"Connection state: {pc.connectionState}")
            
            @pc.on("icecandidate")
            async def on_icecandidate(event):
                if not event.candidate:
                    return
            offer = await pc.createOffer()
            await pc.setLocalDescription(offer)
            await websocket.send(json.dumps({
                "sdp": pc.localDescription.sdp,
                "type": pc.localDescription.type
            }).encode() + b"\n")
            while True:
                video_track.read_frame()
                await asyncio.sleep(0)
        finally:
            await pc.close()
            listener_task.cancel()


if __name__ == "__main__":
    asyncio.run(main())