import time
import os
from dotenv import load_dotenv

load_dotenv()


class FpsMonitor:
    start_time = None
    frame_count = 0
    skip_frames = int(os.getenv("SKIP_FRAMES", "10"))
    
    def start(self):
        self.start_time = time.time()

    def update_frame_count(self):
        self.frame_count += 1

    def update_elapsed_time(self):
        elapsed_time = time.time() - self.start_time
        actual_fps = 0
        if elapsed_time > 1.0:  # Update every second
            actual_fps = self.frame_count / elapsed_time
            self.frame_count = 0
            self.start_time = time.time()
        return actual_fps

    def is_skip_frame(self):
        return self.frame_count % (self.skip_frames + 1) != 0