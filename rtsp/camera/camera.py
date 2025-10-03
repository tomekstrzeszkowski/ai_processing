import os
import cv2
import numpy as np
import time
import re
from detector import Detector
from motion import MotionDetector
from dotenv import load_dotenv
from saver import write_frame_to_shared_memory
from datetime import datetime
from fps import FpsMonitor

load_dotenv()
SHOW_NOW_LABEL = bool(os.getenv("SHOW_NOW_LABEL", ""))


def process_frame(frame, detector, is_motion_detected):
    types_counted = 0
    type_detected = -1
    if is_motion_detected:
        height, width = frame.shape[:2]
        scaled_frame = cv2.resize(frame, (int(width * 0.99), int(height * 0.99)))
        for x0, y0, w, h, type_detected, scale in detector.detect_yolo_with_nms(
            scaled_frame
        ):
            cv2.rectangle(frame, (x0, y0), (x0 + w, y0 + h), (0, 255, 0), 1)
            cv2.putText(
                frame,
                f"Detected {detector.yolo_class_id_to_verbose[type_detected]}!",
                (x0 + 10, y0 + 20),
                cv2.FONT_HERSHEY_SIMPLEX,
                0.5,
                (0, 255, 0),
                2,
            )
            types_counted += 1
    now_label = datetime.now().strftime("%Y-%m-%d %H:%M:%S") if SHOW_NOW_LABEL else ""
    cv2.putText(
        frame,
        f"{now_label} detected: {types_counted}{'.' if is_motion_detected else ''}",
        (20, 20),
        cv2.FONT_HERSHEY_SIMPLEX,
        0.7,
        (255, 255, 255),
        2,
    )
    return frame, type_detected


def main():
    url = os.getenv("IP_CAM_URL", "copy .env.template")
    display_preview = bool(os.getenv("DISPLAY_PREVIEW", ""))
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
    width = int(video.get(cv2.CAP_PROP_FRAME_WIDTH) / 8)
    height = int(video.get(cv2.CAP_PROP_FRAME_HEIGHT) / 8)

    print(f"Camera properties: {width=}x{height=} @ {camera_fps=}")

    # Performance tracking
    frame_count = 0
    fps = FpsMonitor()
    fps.start()

    # optimize
    target_width = int(width * 4)
    target_height = int(height * 4)
    video.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
    video.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)
    video.set(cv2.CAP_PROP_FPS, 3)
    type_detected = -1
    try:
        while True:
            frames_to_read, frame = video.read()
            if not frames_to_read:
                print("Failed to grab frame")
                break
            fps.update_frame_count()
            if fps.is_skip_frame():
                continue
            is_motion_detected = motion.detected_long(frame)
            frame, type_detected = process_frame(frame, detector, is_motion_detected)
            if display_preview:
                cv2.imshow("Processed", frame)
            processed_frame_bgr = cv2.cvtColor(np.array(frame), cv2.COLOR_RGB2BGR)
            success, buffer = cv2.imencode(".jpg", processed_frame_bgr)
            if success:
                write_frame_to_shared_memory(
                    buffer, type_detected, shm_name=f"video_frame"
                )
                del buffer
            actual_fps = fps.update_elapsed_time()
            if actual_fps:
                print(f"{actual_fps=:.2f}")
            if not display_preview:
                continue
            # Break on 'q' key press
            key = cv2.waitKey(1) & 0xFF
            if key == ord("q"):
                break
            elif key == ord("s"):
                # Save current frame
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
