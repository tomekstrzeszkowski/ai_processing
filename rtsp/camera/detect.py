import torch
import numpy as np
import cv2
import os
from PIL import Image, ImageDraw, ImageFilter, ImageFont
from detector.const import YoloObject, YOLO_MODEL_NAME_TO_SCALE_TO_ORIGINAL
from detector.detector import Detector
from motion import MotionDetector
from face import FaceDetector
from fps import FpsMonitor
from dotenv import load_dotenv
from saver import write_frame_to_shared_memory, VideoSaver
from drawer import Drawer
from datetime import datetime


load_dotenv()

SHOW_NOW_LABEL = bool(os.getenv("SHOW_NOW_LABEL", ""))
file_name = "video.mp4"


if __name__ == "__main__":
    BLUR_FACES = False
    SAVE_TO_SHM = bool(os.getenv("SAVE_TO_SHM", ""))
    SAVE_VIDEO = bool(os.getenv("SAVE_VIDEO", ""))
    device = torch.device("cuda:0" if torch.cuda.is_available() else "cpu")
    # face detector
    face = FaceDetector(device)
    # human detecter
    detector = Detector()
    motion = MotionDetector(min_area=500, threshold=25)
    font = ImageFont.load_default()
    video = cv2.VideoCapture(file_name)
    length = int(video.get(cv2.CAP_PROP_FRAME_COUNT))
    width = int(video.get(cv2.CAP_PROP_FRAME_WIDTH))
    height = int(video.get(cv2.CAP_PROP_FRAME_HEIGHT))
    camera_fps = video.get(cv2.CAP_PROP_FPS)
    filename_chunks = file_name.split(".")
    processed_name = (
        f"{'_'.join([*filename_chunks[:1], 'processed'])}.{filename_chunks[-1]}"
    )
    video_tracked = VideoSaver(camera_fps, width, height, processed_name)
    yolo_object_to_verbose = {y.value: y.name for y in YoloObject}
    type_detected = -1
    fps = FpsMonitor()
    while frames_to_read := True:
        frames_to_read, frame = video.read()
        if not frames_to_read:
            break
        frame = Image.fromarray(cv2.cvtColor(frame, cv2.COLOR_BGR2RGB))
        frame_array = np.array(frame)
        # scale for faster detections
        frame_array = cv2.resize(
            frame_array, (int(frame.size[0] * 0.99), int(frame.size[1] * 0.99))
        )
        draw = ImageDraw.Draw(frame)
        # detect
        should_process = fps.should_process()
        cached_detection = None if should_process else detector.last_detection
        motion_detected = motion.detected_long(frame_array) if should_process else False
        if not should_process:
            print(cached_detection)
        drawer = Drawer(frame_array)
        if motion_detected or cached_detection:
            type_detected = -1
            detection = cached_detection or detector.detect_yolo_with_largest_box(frame_array)
            for (x0, y0, w, h, type_detected, scale) in detection:
                drawer.rectangle(yolo_object_to_verbose[type_detected], x0, y0, w, h)
        drawer.label(motion_detected)
        frame_draw = drawer.get_from_array()

        # detect faces
        if BLUR_FACES:
            if (faces := face.detect(frame_array)) is None:
                continue
            draw.text((20, 50), f"Faces: {len(faces)}", fill="white", font=font)
            blurred = frame_draw.filter(ImageFilter.GaussianBlur(40))
            for face in faces:
                mask_box = make_ellipse_mask(frame_draw.size, face)
                frame_draw.paste(blurred, mask=mask_box)
                draw.rectangle(face.tolist(), outline="red")
        frame_bgr = cv2.cvtColor(np.array(frame_draw), cv2.COLOR_RGB2BGR)
        if SAVE_TO_SHM:
            success, buffer = cv2.imencode(".jpg", frame_bgr)
            if success:
                write_frame_to_shared_memory(
                    buffer, type_detected, shm_name=f"video_frame"
                )
        video_tracked.add_frame(cv2.resize(frame_bgr, (width, height)))
    video.release()
    video_tracked.save()
