import cv2
import numpy as np
from collections import deque
from dotenv import load_dotenv
import os


load_dotenv()

class MotionDetector:
    history_ticks = int(os.getenv("HISTORY_TICKS", "120"))
    history = deque([False] * history_ticks, maxlen=history_ticks)
    is_active_move = False

    def __init__(self, min_area: int=500, threshold: int=25):
        self.min_area = min_area
        self.background_subtractor = cv2.createBackgroundSubtractorMOG2(
            history=500, varThreshold=threshold, detectShadows=False
        )

    def resize_history(self, new_size: int) -> None:
        if new_size == self.history_ticks:
            return
        if self.history_ticks > new_size:
            # Trim from the left, keeping the most recent items
            for _ in range(self.history_ticks - new_size):
                self.history.popleft()
        else:
            # Pad on the left
            self.history.extendleft([self.is_active_move] * (new_size - self.history_ticks))
        self.history_ticks = new_size

    def detect(self, frame):
        fg_mask = self.background_subtractor.apply(frame)
        # Remove noise
        kernel = np.ones((5, 5), np.uint8)
        fg_mask = cv2.morphologyEx(fg_mask, cv2.MORPH_OPEN, kernel)
        fg_mask = cv2.morphologyEx(fg_mask, cv2.MORPH_CLOSE, kernel)
        # Find contours
        contours, _ = cv2.findContours(
            fg_mask, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE
        )
        motion_detected = False
        motion_boxes = []

        for contour in contours:
            if cv2.contourArea(contour) < self.min_area:
                continue
            motion_detected = True
            x, y, w, h = cv2.boundingRect(contour)
            yield True, (x, y, w, h)

    def detected_long(self, frame) -> bool:
        """Detect move and minimalize random state change."""
        is_motion_detected, _ = next(self.detect(frame), (False, tuple()))
        self.history.append(is_motion_detected)
        active_move = any(self.history) if self.is_active_move else all(self.history)
        if active_move and not self.is_active_move:
            self.is_active_move = True
        if not active_move and self.is_active_move:
            self.is_active_move = False
        return active_move
