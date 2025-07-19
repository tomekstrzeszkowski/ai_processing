from facenet_pytorch import MTCNN
import torch
import numpy as np
import cv2
from PIL import Image, ImageDraw, ImageFilter, ImageFont
from yolo_object import YoloObject, YOLO_MODEL_NAME_TO_SCALE_TO_ORIGINAL
from detector import Detector
from saver import write_frame_to_shared_memory

file_name = "video.mp4"


def make_ellipse_mask(size, box, ellipse_blur=10):
    mask = Image.new("L", size, color=0)
    draw = ImageDraw.Draw(mask)
    x0, y0, x1, y1 = box.tolist()
    ellipse_size = [x0 * 0.95, y0 * 0.95, x1 * 1.05, y1 * 1.05]
    draw.ellipse(ellipse_size, fill=255)
    return mask.filter(ImageFilter.GaussianBlur(radius=ellipse_blur))


if __name__ == "__main__":
    BLUR_FACES = False
    device = torch.device("cuda:0" if torch.cuda.is_available() else "cpu")
    # face detector
    mtcnn = MTCNN(keep_all=True, device=device)
    # human detecter
    detector = Detector()
    font = ImageFont.load_default()
    frames_tracked = []
    video = cv2.VideoCapture(file_name)
    length = int(video.get(cv2.CAP_PROP_FRAME_COUNT))
    width = int(video.get(cv2.CAP_PROP_FRAME_WIDTH))
    height = int(video.get(cv2.CAP_PROP_FRAME_HEIGHT))
    fps = video.get(cv2.CAP_PROP_FPS)
    i = 0
    yolo_object_to_verbose = {y.value: y.name for y in YoloObject}
    while frames_to_read := True:
        i += 1
        frames_to_read, frame = video.read()
        if not frames_to_read:
            break
        frame = Image.fromarray(cv2.cvtColor(frame, cv2.COLOR_BGR2RGB))
        print(f"{i}/{length}")
        frame_draw = frame.copy()
        frame_array = np.array(frame_draw)
        # scale for faster detections
        frame_array = cv2.resize(
            frame_array, (int(frame_draw.size[0] * 0.99), int(frame_draw.size[1] * 0.99))
        )
        draw = ImageDraw.Draw(frame_draw)
        # detect humans
        detected = 0
        for x0, y0, w, h, type_, scale in detector.detect_yolo_with_largest_box(frame_array):
            detected += 1
            cv2.rectangle(frame_array, (x0, y0), (x0 + w, y0 + h), (0, 0, 255), 2)
            cv2.putText(
                frame_array,
                f"Detected {yolo_object_to_verbose[type_]}",
                (x0, y0),
                cv2.FONT_HERSHEY_SIMPLEX,
                0.7,
                (255, 255, 255),
                2,
            )
        cv2.putText(
            frame_array,
            f"Objects: {detected}",
            (20, 20),
            cv2.FONT_HERSHEY_SIMPLEX,
            0.7,
            (255, 255, 255),
            2,
        )
        frame_draw = Image.fromarray(frame_array)

        # detect faces
        if BLUR_FACES:
            faces, _ = mtcnn.detect(frame_array)
            if faces is None:
                continue
            draw.text((20, 50), f"Faces: {len(faces)}", fill="white", font=font)
            blurred = frame_draw.filter(ImageFilter.GaussianBlur(40))
            for face in faces:
                mask_box = make_ellipse_mask(frame_draw.size, face)
                frame_draw.paste(blurred, mask=mask_box)
                draw.rectangle(face.tolist(), outline="red")
        frame_bgr = cv2.cvtColor(np.array(frame_draw), cv2.COLOR_RGB2BGR)
        success, buffer = cv2.imencode('.jpg', frame_bgr)
        if success:
            write_frame_to_shared_memory(buffer)
        frames_tracked.append(frame_draw.resize((width, height), Image.BILINEAR))


    dim = frames_tracked[0].size
    fourcc = cv2.VideoWriter_fourcc(*"mp4v")
    filename_chunks = file_name.split(".")
    processed_name = (
        f"{'_'.join([*filename_chunks[:1], 'processed'])}.{filename_chunks[-1]}"
    )
    video_tracked = cv2.VideoWriter(processed_name, fourcc, fps, dim)
    for frame in frames_tracked:
        video_tracked.write(cv2.cvtColor(np.array(frame), cv2.COLOR_RGB2BGR))
    video_tracked.release()