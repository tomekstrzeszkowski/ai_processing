from facenet_pytorch import MTCNN
import torch
import numpy as np
import mmcv, cv2
from PIL import Image, ImageDraw, ImageFilter, ImageFont
from IPython import display
from collections import namedtuple


Coordinates = namedtuple("Coordinates", ["x0", "y0", "w", "h"])

file_name = "video.mp4"

device = torch.device("cuda:0" if torch.cuda.is_available() else "cpu")
# face detector
mtcnn = MTCNN(keep_all=True, device=device)
# human detecter
hog = cv2.HOGDescriptor()
hog.setSVMDetector(cv2.HOGDescriptor_getDefaultPeopleDetector())

video = cv2.VideoCapture(file_name)
length = int(video.get(cv2.CAP_PROP_FRAME_COUNT))
width = int(video.get(cv2.CAP_PROP_FRAME_WIDTH))
height = int(video.get(cv2.CAP_PROP_FRAME_HEIGHT))
fps = video.get(cv2.CAP_PROP_FPS)

frames_tracked = []
font = ImageFont.truetype("arial.ttf", 36)


def make_ellipse_mask(size, coordinates, ellipse_blur=10):
    mask = Image.new("L", size, color=0)
    draw = ImageDraw.Draw(mask)
    x0, y0, w, h = coordinates
    ellipse_size = [x0 * 0.95, y0 * 0.95, w * 1.05, h * 1.05]
    draw.ellipse(ellipse_size, fill=255)
    return mask.filter(ImageFilter.GaussianBlur(radius=ellipse_blur))


while frames_to_read := True:
    frames_to_read, frame = video.read()
    if not frames_to_read:
        break
    frame = Image.fromarray(cv2.cvtColor(frame, cv2.COLOR_BGR2RGB))
    frame_draw = frame.copy()
    frame_array = np.array(frame_draw)
    # scale for faster detections
    scale = 0.5
    frame_array = cv2.resize(
        frame_array, (int(frame_draw.size[0] * scale), int(frame_draw.size[1] * scale))
    )
    draw = ImageDraw.Draw(frame_draw)

    # detect faces
    faces, _ = mtcnn.detect(frame_array)
    if faces is None:
        continue
    draw.text((20, 50), f"Faces: {len(faces)}", color="white", font=font)
    blurred = frame_draw.filter(ImageFilter.GaussianBlur(40))
    for face in faces:
        coordinates = Coordinates(*[x / scale for x in face.tolist()])
        mask_box = make_ellipse_mask(frame_draw.size, coordinates)
        frame_draw.paste(blurred, mask=mask_box)
        draw.rectangle(coordinates, outline="red")
    # detect humans
    humans, _ = hog.detectMultiScale(frame_array)
    if humans is not None:
        for human in humans:
            x0, y0, w, h = Coordinates(*[x / scale for x in human.tolist()])
            draw.rectangle([(x0, y0), (x0 + w, y0 + h)], outline="blue", width=4)

        draw.text((20, 20), f"Humans: {len(humans)}", color="white", font=font)
    frames_tracked.append(frame_draw.resize((640, 380), Image.BILINEAR))

dim = frames_tracked[0].size
fourcc = cv2.VideoWriter_fourcc(*"FMP4")
filename_chunks = file_name.split(".")
processed_name = (
    f"{'_'.join([*filename_chunks[:1], 'processed'])}.{filename_chunks[-1]}"
)
video_tracked = cv2.VideoWriter(processed_name, fourcc, fps, dim)
for frame in frames_tracked:
    video_tracked.write(cv2.cvtColor(np.array(frame), cv2.COLOR_RGB2BGR))
video_tracked.release()
