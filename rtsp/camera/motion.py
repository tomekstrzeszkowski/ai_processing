import cv2
import numpy as np
from dotenv import load_dotenv
import os

load_dotenv()


class MotionDetector:
    detection_duration = float(os.getenv("MOTION_DURATION", "1.0"))
    deactive_duration = float(os.getenv("MOTION_DEACTIVE_DURATION", detection_duration))
    is_active_move = False
    first_target_state_frame = None  # Frame count when we first saw target state
    frame_count = 0

    def __init__(self, min_area: int = 500, threshold: int = 25, fps: float = 30.0):
        self.min_area = min_area
        self.fps = fps
        self.background_subtractor = cv2.createBackgroundSubtractorMOG2(
            history=500, varThreshold=threshold, detectShadows=False
        )
        
        # Convert durations to frame counts (more efficient than time.time())
        self.detection_frames = int(self.detection_duration * fps)
        self.deactive_frames = int(self.deactive_duration * fps)

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
        """Detect move and minimize random state changes using frame counting."""
        is_motion_detected, _ = next(self.detect(frame), (False, tuple()))
        self.frame_count += 1
        if self.frame_count > 1000:
            self.frame_count = 0
            self.first_target_state_frame = None
        # Looking for motion to activate
        target_state = is_motion_detected
        change_state_frames = self.detection_frames
        if self.is_active_move:
            # Looking for no motion to deactivate
            target_state = not is_motion_detected
            change_state_frames = self.deactive_frames
        if target_state:
            if self.first_target_state_frame is None:
                self.first_target_state_frame = self.frame_count
            else:
                # Check if we've had this state long enough (in frames)
                frames_elapsed = self.frame_count - self.first_target_state_frame
                if frames_elapsed >= change_state_frames:
                    # Toggle the state
                    self.is_active_move = not self.is_active_move
                    self.first_target_state_frame = None
        else:
            self.first_target_state_frame = None
            
        return self.is_active_move