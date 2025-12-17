import cv2
import numpy as np
from dotenv import load_dotenv
import os

load_dotenv()


class MotionDetector:
    detection_duration = float(os.getenv("MOTION_DURATION", "1.0"))
    deactive_duration = float(os.getenv("MOTION_DEACTIVE_DURATION", detection_duration))
    is_active_move = False
    first_target_state_frame = None 
    frame_count = 0

    def __init__(self, min_area: int = 500, threshold: int = 25, fps: float = 15.0):
        self.min_area = min_area
        self.fps = fps
        self.scale_factor = 0.25
        self.background_subtractor = cv2.createBackgroundSubtractorMOG2(
            history=500, varThreshold=threshold, detectShadows=False
        )
        self.detection_frames = int(self.detection_duration * fps)
        self.deactive_frames = int(self.deactive_duration * fps)
        
        self.kernel = np.ones((5, 5), np.uint8)
        
        # Pre-calculate scaled min area (do once instead of every frame)
        self.scaled_min_area = self.min_area * (self.scale_factor ** 2)
        
        # Pre-calculate scale inverse
        self.scale_inv = 1.0 / self.scale_factor
        self.last_detection_result = False
        self.process_every_n_frames = 2  # Process every 2nd frame
        
    def _has_significant_motion(self, fg_mask) -> bool:
        """Fast motion check without full contour analysis."""
        # Count non-zero pixels - much faster than findContours
        motion_pixels = cv2.countNonZero(fg_mask)
        # Rough heuristic: if enough pixels changed, there's motion
        return motion_pixels > self.scaled_min_area * 0.5


    def detected_long(self, frame) -> bool:
        """Optimized motion detection with frame skipping and fast path."""
        self.frame_count += 1
        
        # Skip frames for efficiency
        if self.frame_count % self.process_every_n_frames != 0:
            return self.is_active_move
        
        # Fast path: downscale and check for motion without full contour analysis
        frame_small = cv2.resize(
            frame, 
            None, 
            fx=self.scale_factor, 
            fy=self.scale_factor, 
            interpolation=cv2.INTER_LINEAR
        )
        fg_mask = self.background_subtractor.apply(frame_small)
        fg_mask = cv2.morphologyEx(fg_mask, cv2.MORPH_OPEN, self.kernel)
        fg_mask = cv2.morphologyEx(fg_mask, cv2.MORPH_CLOSE, self.kernel)
        
        self.last_detection_result = self._has_significant_motion(fg_mask)
        
        if self.frame_count >= 1000:
            self.frame_count = 0
            self.first_target_state_frame = None

        # State machine logic
        if self.is_active_move:
            # Looking for no motion to deactivate
            if not self.is_motion_detected:
                if self.first_target_state_frame is None:
                    self.first_target_state_frame = self.frame_count
                elif (self.frame_count - self.first_target_state_frame) >= self.deactive_frames:
                    self.is_active_move = False
                    self.first_target_state_frame = None
            else:
                self.first_target_state_frame = None
        else:
            # Looking for motion to activate
            if is_motion_detected:
                if self.first_target_state_frame is None:
                    self.first_target_state_frame = self.frame_count
                elif (self.frame_count - self.first_target_state_frame) >= self.detection_frames:
                    self.is_active_move = True
                    self.first_target_state_frame = None
            else:
                self.first_target_state_frame = None
            
        return self.is_active_move