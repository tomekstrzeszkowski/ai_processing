import cv2
import numpy as np
from collections import deque


class MotionDetector:
    HISTORY_TICKS = int(30 * 0.5)
    history = deque([False] * HISTORY_TICKS, maxlen=HISTORY_TICKS)

    def __init__(self, min_area=500, threshold=25):
        self.min_area = min_area
        self.background_subtractor = cv2.createBackgroundSubtractorMOG2(
            history=500, varThreshold=threshold, detectShadows=False
        )

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

    def detected_long(self, frame):
        is_motion_detected, _ = next(self.detect(frame), (False, tuple()))
        self.history.append(is_motion_detected)
        return all(self.history)
