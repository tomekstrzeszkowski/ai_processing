import os
import cv2
import numpy as np
import time
from detector import Detector
from dotenv import load_dotenv
from saver import write_frame_to_shared_memory

load_dotenv()


def process_frame(frame, detector):
    processed_frame = frame.copy()
    height, width = processed_frame.shape[:2]
    scaled_frame = cv2.resize(
        processed_frame, (int(width * 0.99), int(height * 0.99))
    )
    detected_objects = 0
    type_ = -1
    for x0, y0, w, h, type_, scale in detector.detect_yolo_with_nms(scaled_frame):
        cv2.rectangle(processed_frame, (x0, y0), (x0 + w, y0 + h), (0, 255, 0), 1)
        cv2.putText(
            processed_frame,
            f"Detected {detector.yolo_class_id_to_verbose[type_]}!",
            (x0+10, y0+20),
            cv2.FONT_HERSHEY_SIMPLEX,
            0.5,
            (0, 255, 0),
            2,
        )
        detected_objects += 1
    if detected_objects:
        cv2.putText(
            processed_frame, 
            f"Objects: {detected_objects}", 
            (20, 20), 
            cv2.FONT_HERSHEY_SIMPLEX, 
            0.7, 
            (255, 255, 255),
            2,
        )
    
    return processed_frame, type_

def main():
    url = os.getenv("IP_CAM_URL", "copy .env.template")
    detector = Detector()
    print(f"Connecting to camera: {url} with AI model")

    
    # Create VideoCapture object
    os.environ["OPENCV_FFMPEG_CAPTURE_OPTIONS"] = "video_codec;h264_cuvid"
    video = cv2.VideoCapture(url, cv2.CAP_FFMPEG)
    
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
    fps = video.get(cv2.CAP_PROP_FPS)
    width = int(video.get(cv2.CAP_PROP_FRAME_WIDTH)/8)
    height = int(video.get(cv2.CAP_PROP_FRAME_HEIGHT)/8)
    
    print(f"Camera properties: {width}x{height} @ {fps} FPS")
    
    # Performance tracking
    frame_count = 0
    start_time = time.time()

    #optimize
    skip_frames = 10
    target_width = int(width * 4)
    target_height = int(height * 4)
    video.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
    video.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)
    video.set(cv2.CAP_PROP_FPS, 3)
    
    try:
        while True:
            frames_to_read, frame = video.read()
            frame_count += 1
            
            if not frames_to_read:
                print("Failed to grab frame")
                break

            if frame_count % (skip_frames + 1) != 0:
                continue
            small_frame = cv2.resize(frame, (target_width, target_height))
            processed_frame, type_ = process_frame(small_frame, detector)
            
            # Display frames
            cv2.imshow('Processed', processed_frame)
            processed_frame_bgr = cv2.cvtColor(np.array(processed_frame), cv2.COLOR_RGB2BGR)
            success, buffer = cv2.imencode('.jpg', processed_frame_bgr)
            if success:
                write_frame_to_shared_memory(
                    buffer, type_, shm_name=f"video_frame"
                )
            elapsed_time = time.time() - start_time
            if elapsed_time > 1.0:  # Update every second
                actual_fps = frame_count / elapsed_time
                print(f"Actual FPS: {actual_fps:.2f}")
                frame_count = 0
                start_time = time.time()
            
            # Break on 'q' key press
            key = cv2.waitKey(1) & 0xFF
            if key == ord('q'):
                break
            elif key == ord('s'):
                # Save current frame
                timestamp = int(time.time())
                cv2.imwrite(f'videotured_frame_{timestamp}.jpg', frame)
                print(f"Frame saved as videotured_frame_{timestamp}.jpg")
    
    except KeyboardInterrupt:
        print("Interrupted by user")
    
    finally:
        # Clean up
        video.release()
        cv2.destroyAllWindows()
        print("Camera connection closed")

if __name__ == "__main__":
    main()