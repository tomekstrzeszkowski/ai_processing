import cv2
import numpy as np
from dotenv import load_dotenv
import os
import time

load_dotenv()


class MotionDetector:
    detection_duration = float(os.getenv("MOTION_DURATION", "1.0"))
    deactive_duration = float(os.getenv("MOTION_DEACTIVE_DURATION", detection_duration))
    is_active_move = False
    first_target_state_time = None  # First time we saw the target state

    def __init__(self, min_area: int = 500, threshold: int = 25):
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

        for contour in contours:
            if cv2.contourArea(contour) < self.min_area:
                continue
            x, y, w, h = cv2.boundingRect(contour)
            yield True, (x, y, w, h)

    def detected_long(self, frame) -> bool:
        """Detect move and minimize random state changes.

        Logic:
        - If is_active_move=False: ALL detections must be True for detection_duration to activate
        - If is_active_move=True: ALL detections must be False for detection_duration to deactivate
        """
        is_motion_detected, _ = next(self.detect(frame), (False, tuple()))
        current_time = time.time()

        # Looking for motion to activate
        target_state = is_motion_detected  # Need True (motion)
        change_state_duration = self.detection_duration
        if self.is_active_move:
            # Looking for no motion to deactivate
            target_state = not is_motion_detected  # Need True (no motion)
            change_state_duration = self.deactive_duration

        # Check if we have the target state
        if target_state:
            # We have the state we're looking for
            if self.first_target_state_time is None:
                # First time seeing this target state
                self.first_target_state_time = current_time
            else:
                # Check if we've had this state long enough
                duration = current_time - self.first_target_state_time
                if duration >= change_state_duration:
                    # Toggle the state
                    self.is_active_move = not self.is_active_move
                    self.first_target_state_time = None
        else:
            # We don't have the target state - reset the timer
            self.first_target_state_time = None
        return self.is_active_move
