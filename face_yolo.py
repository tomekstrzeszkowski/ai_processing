from facenet_pytorch import MTCNN
import torch
import numpy as np
import cv2
from PIL import Image, ImageDraw, ImageFilter, ImageFont
from IPython import display
import yolo_object


file_name = "xc.mov"


def detect_people_yolo8(net, original_image):
    [height, width, _] = original_image.shape
    # Prepare a square image for inference
    length = max((height, width))
    image = np.zeros((length, length, 3), np.uint8)
    image[0:height, 0:width] = original_image

    # Calculate scale factor
    scale = length / 640

    blob = cv2.dnn.blobFromImage(
        image, scalefactor=1 / 255, size=(640, 640), swapRB=True
    )
    net.setInput(blob)

    # Perform inference
    outputs = net.forward()

    # Prepare output array
    output_layers = np.array([cv2.transpose(outputs[0])])

    # Prepare output array
    outputs = np.array([cv2.transpose(outputs[0])])
    rows = outputs.shape[1]

    # Iterate through output to collect bounding boxes, confidence scores, and class IDs
    for i in range(rows):
        classes_scores = outputs[0][i][4:]
        (minScore, maxScore, minClassLoc, (x, maxClassIndex)) = cv2.minMaxLoc(
            classes_scores
        )
        if maxScore >= 0.3 and maxClassIndex in (yolo_object.PERSON, yolo_object.CAR):
            yield [
                int((outputs[0][i][0] - (0.5 * outputs[0][i][2])) * scale),
                int((outputs[0][i][1] - (0.5 * outputs[0][i][3])) * scale),
                int(outputs[0][i][2] * scale),
                int(outputs[0][i][3] * scale),
            ]


def detect_people_yolo3(net, image):
    height, width = image.shape[0], image.shape[1]
    blob = cv2.dnn.blobFromImage(image, 1 / 255.0, (416, 416), swapRB=True, crop=False)
    net.setInput(blob)

    # Perform forward propagation
    output_layer_name = net.getUnconnectedOutLayersNames()
    output_layers = net.forward(output_layer_name)

    # Loop over the output layers
    for output in output_layers:
        # Loop over the detections
        for detection in output:
            # Extract the class ID and confidence of the current detection
            scores = detection[5:]
            class_id = np.argmax(scores)
            confidence = scores[class_id]

            # Only keep detections with a high confidence
            if class_id == 0 and confidence > 0.9:
                # Object detected
                center_x = int(detection[0] * width)
                center_y = int(detection[1] * height)
                w = int(detection[2] * width)
                h = int(detection[3] * height)

                # Rectangle coordinates
                x = int(center_x - w / 2)
                y = int(center_y - h / 2)

                # Add the detection to the list of people
                yield (x, y, w, h)


def make_ellipse_mask(size, box, ellipse_blur=10):
    mask = Image.new("L", size, color=0)
    draw = ImageDraw.Draw(mask)
    x0, y0, x1, y1 = box.tolist()
    ellipse_size = [x0 * 0.95, y0 * 0.95, x1 * 1.05, y1 * 1.05]
    draw.ellipse(ellipse_size, fill=255)
    return mask.filter(ImageFilter.GaussianBlur(radius=ellipse_blur))


device = torch.device("cuda:0" if torch.cuda.is_available() else "cpu")
# face detector
mtcnn = MTCNN(keep_all=True, device=device)
# human detecter
# net = cv2.dnn.readNet("./yolov3.weights", "./yolov3.cfg")
# hog = cv2.HOGDescriptor()
# hog.setSVMDetector(cv2.HOGDescriptor_getDefaultPeopleDetector())
# model = torch.hub.load('ultralytics/yolov5', 'yolov5s', pretrained=True)
net = cv2.dnn.readNetFromONNX("./yolov8n.onnx")
font = ImageFont.truetype("arial.ttf", 36)
# video = mmcv.VideoReader(file_name)
# frames = [Image.fromarray(cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)) for frame in video]
frames_tracked = []
video = cv2.VideoCapture(file_name)
length = int(video.get(cv2.CAP_PROP_FRAME_COUNT))
width = int(video.get(cv2.CAP_PROP_FRAME_WIDTH))
height = int(video.get(cv2.CAP_PROP_FRAME_HEIGHT))
fps = video.get(cv2.CAP_PROP_FPS)
i = 0
while frames_to_read := True:
    i += 1
    frames_to_read, frame = video.read()
    if not frames_to_read:
        break
    frame = Image.fromarray(cv2.cvtColor(frame, cv2.COLOR_BGR2RGB))
    print(f"{i}/{length}")
    # create a new movie with detections only
    is_detected = True
    frame_draw = frame.copy()
    frame_array = np.array(frame_draw)
    # scale for faster detections
    frame_array = cv2.resize(
        frame_array, (int(frame_draw.size[0] * 0.99), int(frame_draw.size[1] * 0.99))
    )
    draw = ImageDraw.Draw(frame_draw)
    # detect humans
    humans = list(detect_people_yolo8(net, frame_array))
    # humans, _ = hog.detectMultiScale(
    #     frame_array, winStride=(4, 4), padding=(3, 3), scale=1.1
    # )
    if humans is not None:
        is_detected = True
        for x0, y0, w, h in humans:
            cv2.rectangle(frame_array, (x0, y0), (x0 + w, y0 + h), (0, 0, 255), 2)
        frame_draw = Image.fromarray(frame_array)
        draw = ImageDraw.Draw(frame_draw)
        draw.text((20, 20), f"Humans: {len(humans)}", color="white", font=font)

    # detect faces
    # faces, _ = mtcnn.detect(frame_array)
    # if faces is None:
    #     continue
    # draw.text((20, 50), f"Faces: {len(faces)}", color="white", font=font)

    # blurred = frame_draw.filter(ImageFilter.GaussianBlur(40))
    # for face in faces:
    #     is_detected = True
    #     mask_box = make_ellipse_mask(frame_draw.size, face)
    #     frame_draw.paste(blurred, mask=mask_box)
    #     draw.rectangle(face.tolist(), outline="red")
    # frames with detections
    if is_detected:
        frames_tracked.append(frame_draw.resize((width, height), Image.BILINEAR))

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
