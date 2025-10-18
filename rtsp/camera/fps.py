import time
import os
from dotenv import load_dotenv

load_dotenv()


class FpsMonitor:
    start_time = None
    processed_count = 0  # Only count processed frames
    total_count = 0  # Count all frames
    skip_frames = int(os.getenv("SKIP_FRAMES", "10"))

    def __init__(self, camera_fps=0.0):
        self.camera_fps = camera_fps
        self.last_fps = camera_fps

    def start(self):
        self.start_time = time.time()

    def update_frame_count(self):
        self.total_count += 1

    def mark_processed(self):
        """Call this after actually processing a frame"""
        self.processed_count += 1

    def update_elapsed_time(self):
        elapsed_time = time.time() - self.start_time
        if elapsed_time >= 1.0:  # Update every second
            actual_fps = self.processed_count / elapsed_time
            self.last_fps = actual_fps
            self.processed_count = 0
            self.total_count = 0
            self.start_time = time.time()
            return True
        return False

    def is_skip_frame(self):
        return self.total_count % (self.skip_frames + 1) != 0

    def get_current(self):
        return self.last_fps or self.camera_fps / (self.skip_frames + 1)
