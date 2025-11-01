import os
import cv2
import numpy as np
import time
import re
from detector import Detector
from motion import MotionDetector
from dotenv import load_dotenv
from saver import write_frame_to_shared_memory, VideoSaver
from datetime import datetime
from fps import FpsMonitor
from drawer import Drawer

load_dotenv()
SHOW_NOW_LABEL = bool(os.getenv("SHOW_NOW_LABEL", ""))


def process_frame(frame, detector, is_motion_detected):
    type_detected = -1
    drawer = Drawer(frame)
    if is_motion_detected:
        height, width = frame.shape[:2]
        scaled_frame = cv2.resize(frame, (int(width * 0.99), int(height * 0.99)))
        for x0, y0, w, h, type_detected, scale in detector.detect_yolo_with_nms(
            scaled_frame
        ):
            drawer.rectangle(detector.yolo_class_id_to_verbose[type_detected], x0, y0, w, h)
    drawer.label(is_motion_detected)
    return frame, type_detected


def main():
    url = os.getenv("IP_CAM_URL", "copy .env.template")
    display_preview = bool(os.getenv("DISPLAY_PREVIEW", ""))
    SAVE_TO_SHM = bool(os.getenv("SAVE_TO_SHM", ""))
    SAVE_VIDEO = bool(os.getenv("SAVE_VIDEO", ""))
    detector = Detector()
    motion = MotionDetector(min_area=500, threshold=25)
    url_clean = re.sub(r"(rtsp:\/\/.+:)(.+)@", r"\1***@", url)
    print(f"Connecting to camera: {url_clean} with AI model")
    if capture_options := os.getenv("OPENCV_FFMPEG_CAPTURE_OPTIONS", ""):
        os.environ["OPENCV_FFMPEG_CAPTURE_OPTIONS"] = capture_options

    if url.startswith("rtsp"):
        video = cv2.VideoCapture(url, cv2.CAP_FFMPEG)
    else:
        video = cv2.VideoCapture(int(url))
    # Set buffer size to reduce latency
    video.set(cv2.CAP_PROP_BUFFERSIZE, 1)
    video.set(cv2.CAP_PROP_OPEN_TIMEOUT_MSEC, 1 * 10_000)

    # Check if camera opened successfully
    if not video.isOpened():
        print("Error: Could not connect to camera")
        return None

    print("Successfully connected to camera")
    if video is None:
        return

    # Get camera properties
    camera_fps = video.get(cv2.CAP_PROP_FPS)
    width = int(video.get(cv2.CAP_PROP_FRAME_WIDTH) / 2)
    height = int(video.get(cv2.CAP_PROP_FRAME_HEIGHT) / 2)
    video.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
    video.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)
    video.set(cv2.CAP_PROP_FPS, 3)
    video_tracked = None
    print(f"Camera properties: {width=}x{height=} @ {camera_fps=}")
    has_detected = False
    fps = FpsMonitor(camera_fps)
    fps.start()
    try:
        while True:
            frames_to_read, camera_frame = video.read()
            if not frames_to_read:
                print("Failed to grab frame")
                continue
            fps.update_frame_count()
            if fps.is_skip_frame():
                continue
            is_motion_detected = motion.detected_long(camera_frame)
            frame, type_detected = process_frame(
                camera_frame, detector, is_motion_detected
            )
            fps.mark_processed()
            if fps.update_elapsed_time():
                print(f"{fps.get_current()=:.2f}")
            if display_preview:
                cv2.imshow("Processed", frame)
            if SAVE_TO_SHM:
                success, buffer = cv2.imencode(".jpg", frame)
                if success:
                    write_frame_to_shared_memory(
                        buffer, type_detected, shm_name=f"video_frame"
                    )
                del buffer
            if SAVE_VIDEO:
                found_object = type_detected != -1
                has_detected = found_object or has_detected
                if found_object or is_motion_detected:
                    if not video_tracked:
                        video_tracked = VideoSaver(
                            fps.get_current(),
                            width,
                            height,
                        )
                    video_tracked.add_frame(cv2.resize(frame, (width, height)))
                elif video_tracked:
                    if has_detected:
                        video_tracked.save()
                    else:
                        video_tracked.remove()
                    video_tracked = None
                    has_detected = False
            if not display_preview:
                continue
            # Break on 'q' key press
            key = cv2.waitKey(1) & 0xFF
            if key == ord("q"):
                break
            elif key == ord("s"):
                timestamp = int(time.time())
                cv2.imwrite(f"videotured_frame_{timestamp}.jpg", frame)
                print(f"[{timestamp}] Frame saved as video_frame.jpg")
    except KeyboardInterrupt:
        print("Interrupted by user")
    finally:
        video.release()
        if display_preview:
            cv2.destroyAllWindows()
        print("Camera connection closed")


if __name__ == "__main__":
    main()
