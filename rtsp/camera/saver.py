import struct
import os
import cv2
import time


def write_frame_to_shared_memory(buffer, type_, width, height, shm_name="video_frame"):
    """Save frame buffer to shared memory."""
    data = buffer.tobytes()
    header = struct.pack("<bII", type_, width, height)
    shm_path = f"/dev/shm/{shm_name}"
    temp_path = f"{shm_path}.tmp"
    with open(temp_path, "wb") as f:
        f.write(header)
        f.write(data)
    # Atomic rename
    os.rename(temp_path, shm_path)


def read_frame_from_shared_memory(shm_name="video_frame"):
    with open(f"/dev/shm/{shm_name}", "rb") as f:
        header = f.read(9)
        type_, width, height = struct.unpack("<bII", header)
        data = f.read()
    return data, type_


class VideoSaver:
    video = None
    name = ""

    def __init__(self, camera_fps, width, height, file_name=""):
        if not file_name:
            file_name = f"{time.strftime('%Y-%m-%d %H_%M_%S')}.mp4"
        self.video = cv2.VideoWriter(
            file_name, cv2.VideoWriter_fourcc(*"mp4v"), camera_fps, (width, height)
        )
        self.name = file_name

    def add_frame(self, frame):
        self.video.write(frame)

    def save(self):
        self.video.release()

    def remove(self):
        os.remove(self.name)
