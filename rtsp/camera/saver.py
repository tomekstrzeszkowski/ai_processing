import struct
import os
import cv2
import time


def write_frame_to_shared_memory(buffer, type_, shm_name="video_frame"):
    """Save frame buffer to shared memory."""
    data = buffer.tobytes()
    header = struct.pack("<bI", type_, len(data))
    shm_path = f"/dev/shm/{shm_name}"

    # Write frame data directly (no size header needed)
    temp_path = f"{shm_path}.tmp"
    with open(temp_path, "wb") as f:
        f.write(header)
        f.write(data)

    # Atomic rename
    os.rename(temp_path, shm_path)


class VideoSaver:
    video = None

    def __init__(self, camera_fps, width, height, file_name=""):
        if not file_name:
            file_name = f"{time.strftime('%Y-%m-%d %H_%M_%S')}.mp4"
        self.video = cv2.VideoWriter(
            file_name, cv2.VideoWriter_fourcc(*"mp4v"), camera_fps, (width, height)
        )

    def add_frame(self, frame):
        self.video.write(frame)

    def save(self):
        self.video.release()
