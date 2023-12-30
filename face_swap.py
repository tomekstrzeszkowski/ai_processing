from facenet_pytorch import MTCNN
import torch
import numpy as np
import mmcv, cv2
from PIL import Image, ImageDraw, ImageFilter, ImageFont
from IPython import display


file_name = "orch2.mkv"


device = torch.device('cuda:0' if torch.cuda.is_available() else 'cpu')
# face detector
mtcnn = MTCNN(keep_all=True, device=device)
# human detecter
#net = cv2.dnn.readNet("./yolov3.weights", "./yolov3.cfg")
# hog = cv2.HOGDescriptor() 
# hog.setSVMDetector(cv2.HOGDescriptor_getDefaultPeopleDetector())
#model = torch.hub.load('ultralytics/yolov5', 'yolov5s', pretrained=True)
net = cv2.dnn.readNetFromONNX("./yolov8s.onnx")
face_img = Image.open('bar.png')
font = ImageFont.truetype("arial.ttf", 36)
video = mmcv.VideoReader(file_name)
frames = [
    Image.fromarray(cv2.cvtColor(frame, cv2.COLOR_BGR2RGB))
    for frame in video[:50]
]
frames_tracked = []
for i, frame in enumerate(frames):
    print(f"{i} / {len(frames)}")
    # create a new movie with detections only
    is_detected = True
    frame_draw = frame.copy()
    frame_array = np.array(frame_draw)
    # scale for faster detections
    frame_array = cv2.resize(
        frame_array, (int(frame_draw.size[0] * 0.99), int(frame_draw.size[1] * 0.99))
    )
    draw = ImageDraw.Draw(frame_draw)
    # detect faces
    faces, _ = mtcnn.detect(frame_array)
    if faces is None:
        continue
    for face in faces:
        x0, y0, x1, y1 = face.tolist()
        face_copy = face_img.copy()
        print((int(x1-x0), int(y1-y0)))
        face_copy = face_copy.resize((int(x1-x0), int(y1-y0)), Image.ANTIALIAS)
        frame_draw.paste(face_copy, (int(x0), int(y0)))
        #draw.rectangle(face.tolist(), outline="red")
    frames_tracked.append(
        frame_draw.resize((640, 380), Image.BILINEAR)
    )

dim = frames_tracked[0].size
fourcc = cv2.VideoWriter_fourcc(*'FMP4')
filename_chunks = file_name.split('.')
processed_name = f"{'_'.join([*filename_chunks[:1], 'processed'])}.{filename_chunks[-1]}"
video_tracked = cv2.VideoWriter(processed_name, fourcc, 24.0, dim)
for frame in frames_tracked:
    video_tracked.write(cv2.cvtColor(np.array(frame), cv2.COLOR_RGB2BGR))
video_tracked.release()
