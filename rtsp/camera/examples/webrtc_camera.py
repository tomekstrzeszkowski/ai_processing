import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))

import asyncio
import cv2
import json
import numpy as np
from aiortc import RTCPeerConnection, RTCSessionDescription, VideoStreamTrack, RTCConfiguration, RTCIceServer, RTCIceCandidate
from aiortc.contrib.signaling import TcpSocketSignaling
from av import VideoFrame
import fractions
import os
from detector import Detector
from motion import MotionDetector
from dotenv import load_dotenv
from fps import FpsMonitor
from drawer import Drawer
from saver import write_frame_to_shared_memory, VideoSaver, read_frame_from_shared_memory
import websockets


load_dotenv()
SHOW_NOW_LABEL = bool(os.getenv("SHOW_NOW_LABEL", ""))


def process_frame(frame, detector, is_motion_detected):
    type_detected = -1
    drawer = Drawer(frame)
    if is_motion_detected:
        height, width = frame.shape[:2]
        scaled_frame = cv2.resize(frame, (int(width * 0.99), int(height * 0.99)))
        for x0, y0, w, h, type_detected, scale in detector.detect_yolo_with_nms(
            scaled_frame
        ):
            drawer.rectangle(detector.yolo_class_id_to_verbose[type_detected], x0, y0, w, h)
    drawer.label(is_motion_detected)
    return frame, type_detected

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

class CustomVideoStreamTrack(VideoStreamTrack):
    _current_frame: np.ndarray | None = None
    frame_count: int = 0
        
    def set_current_frame(self, frame: np.ndarray):
        """Update the current frame (no lock needed in single-threaded async)"""
        self._current_frame = frame.copy() if frame is not None else None
    
    async def recv(self):
        """Receive the next video frame"""
        self.frame_count += 1
        
        if self._current_frame is None:
            frame = np.zeros((480, 640, 3), dtype=np.uint8)
        else:
            frame = self._current_frame
        
        video_frame = VideoFrame.from_ndarray(frame, format="bgr24")
        video_frame.pts = self.frame_count
        video_frame.time_base = fractions.Fraction(1, 30)
        
        return video_frame


async def read_frame_async(video):
    """Non-blocking frame read using thread executor"""
    loop = asyncio.get_event_loop()
    return await loop.run_in_executor(None, video.read)

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
                    print(message)
                    for ice_data in message["ice"]:
                        candidate = RTCIceCandidate.from_sdp(ice_data["candidate"])
                        candidate.sdpMid = ice_data["sdpMid"]
                        candidate.sdpMLineIndex = ice_data["sdpMLineIndex"]
                        await pc.addIceCandidate(candidate)
    except websockets.exceptions.ConnectionClosed:
        print("Connection closed")

async def main():
    """Fully async main loop"""
    config = RTCConfiguration(iceServers=ICE_SERVERS)
    pc = RTCPeerConnection(config)
    video_track = CustomVideoStreamTrack()
    url = os.getenv("IP_CAM_URL", "0")
    
    # Initialize components
    detector = Detector()
    motion = MotionDetector(min_area=500, threshold=25)
    
    # Open video capture
    if url.startswith("rtsp"):
        video = cv2.VideoCapture(url, cv2.CAP_FFMPEG)
    else:
        video = cv2.VideoCapture(int(url))
    
    video.set(cv2.CAP_PROP_BUFFERSIZE, 1)
    video.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
    video.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)
    
    if not video.isOpened():
        print("Error: Could not connect to camera")
        return

    fps = FpsMonitor()
    fps.start()
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
                # Read frame asynchronously (doesn't block event loop)
                frames_to_read, frame = await read_frame_async(video)
                
                if not frames_to_read:
                    break
                
                fps.update_frame_count()
                if fps.is_skip_frame():
                    continue
                
                # Process frame (CPU-bound, but we accept this blocking)
                is_motion_detected = motion.detected_long(frame)
                frame, type_detected = process_frame(frame, detector, is_motion_detected)
                
                # Convert and update WebRTC
                processed_frame_bgr = cv2.cvtColor(np.array(frame), cv2.COLOR_RGB2BGR)
                video_track.set_current_frame(processed_frame_bgr)
                
                # Encode and write to shared memory
                success, buffer = cv2.imencode(".jpg", processed_frame_bgr)
                if success:
                    write_frame_to_shared_memory(buffer, type_detected, shm_name="video_frame")
                    del buffer
                
                # Yield control to event loop periodically
                await asyncio.sleep(0)  # Allows WebRTC tasks to run
                
        except KeyboardInterrupt:
            print("\nInterrupted")
        finally:
            #video.release()
            await pc.close()
            listener_task.cancel()


if __name__ == "__main__":
    asyncio.run(main())