import os
from dotenv import load_dotenv

load_dotenv()


class FpsMonitor:
    def __init__(self, camera_fps=30.0):
        self.skip_frames = int(os.getenv("SKIP_FRAMES", "10"))
        self.frame_counter = 0
        self.current_fps = camera_fps

    def should_process(self):
        self.frame_counter += 1
        return (self.frame_counter % self.skip_frames) != 0

    def get_current(self):
        return self.current_fps
