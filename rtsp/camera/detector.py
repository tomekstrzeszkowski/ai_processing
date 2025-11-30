import cv2
from functools import cached_property
import numpy as np
from yolo_object import YoloObject, YOLO_MODEL_NAME_TO_SCALE_TO_ORIGINAL


YOLO_MODEL_NAME = "./yolo11n.onnx"


def calculate_iou(box1, box2):
    """
    Calculate Intersection over Union (IoU) of two bounding boxes
    Box format: [x0, y0, width, height]
    """
    x1, y1, w1, h1 = box1
    x2, y2, w2, h2 = box2
    
    # Calculate intersection rectangle
    x_left = max(x1, x2)
    y_top = max(y1, y2)
    x_right = min(x1 + w1, x2 + w2)
    y_bottom = min(y1 + h1, y2 + h2)
    
    if x_right < x_left or y_bottom < y_top:
        return 0.0
    
    # Calculate intersection area
    intersection_area = (x_right - x_left) * (y_bottom - y_top)
    
    # Calculate union area
    box1_area = w1 * h1
    box2_area = w2 * h2
    union_area = box1_area + box2_area - intersection_area
    
    if union_area == 0:
        return 0.0
    
    return intersection_area / union_area


class Detector:
    yolo_model_name = "./yolo11n.onnx"
    detect_only = (YoloObject.PERSON, YoloObject.CAR)
    detect_only_yolo_class_id = [yolo.value for yolo in detect_only]
    yolo_class_id_to_verbose = {yolo.value: yolo.name.lower() for yolo in detect_only}

    @property
    def _resize(self):
        return YOLO_MODEL_NAME_TO_SCALE_TO_ORIGINAL.get(self.yolo_model_name, 1)

    @cached_property
    def model(self):
        return cv2.dnn.readNetFromONNX(self.yolo_model_name)

    def _scale_input(self, original_image):
        [height, width, _] = original_image.shape
        length = max((height, width))
        image = np.zeros((length, length, 3), np.uint8)
        image[0:height, 0:width] = original_image

        # Calculate scale factor
        size = 640
        scale = length / size
        blob = cv2.dnn.blobFromImage(
            image, scalefactor=1 / 255, size=(size, size), swapRB=True
        )
        self.model.setInput(blob)
        outputs = self.model.forward()
        outputs = np.array([cv2.transpose(outputs[0])])
        return outputs, scale

    def detect_yolo_all(self, original_image):
        outputs, scale = self._scale_input(original_image)
        rows = outputs.shape[1]

        # Iterate through output to collect bounding boxes, confidence scores, and class IDs
        for i in range(rows):
            classes_scores = outputs[0][i][4:]
            (minScore, maxScore, minClassLoc, (x, maxClassIndex)) = cv2.minMaxLoc(
                classes_scores
            )
            if maxScore >= 0.34 and maxClassIndex in self.detect_only_yolo_class_id:
                center_x_norm = outputs[0][i][0]  # normalized center x (0-1)
                center_y_norm = outputs[0][i][1]  # normalized center y (0-1)
                width_norm = outputs[0][i][2]     # normalized width (0-1)
                height_norm = outputs[0][i][3]    # normalized height (0-1)
                
                # Convert to pixel coordinates in the 640x640 space, then scale to original
                center_x = center_x_norm * self._resize * scale
                center_y = center_y_norm * self._resize * scale
                box_width = width_norm * self._resize * scale
                box_height = height_norm * self._resize * scale
                
                # Convert center coordinates to top-left corner
                x0 = int(center_x - (box_width / 2))
                y0 = int(center_y - (box_height / 2))
                w = int(box_width)
                h = int(box_height)
                yield [x0, y0, w, h, maxClassIndex, maxScore]

    def detect_yolo_with_nms(self, original_image, nms_threshold=0.4):
        """
        Non-Maximum Suppression remove duplicates.

        Keep only coordinates with highest confidence among overlapping
        boxes.
        """
        outputs, scale = self._scale_input(original_image)
        rows = outputs.shape[1]

        # Collect all detections
        boxes = []
        confidences = []
        class_ids = []

        # Iterate through output to collect bounding boxes, confidence scores, and class IDs
        for i in range(rows):
            classes_scores = outputs[0][i][4:]
            (minScore, maxScore, minClassLoc, (x, maxClassIndex)) = cv2.minMaxLoc(
                classes_scores
            )
            if maxScore >= 0.3 and maxClassIndex in self.detect_only_yolo_class_id:
                center_x_norm = outputs[0][i][0]  # normalized center x (0-1)
                center_y_norm = outputs[0][i][1]  # normalized center y (0-1)
                width_norm = outputs[0][i][2]     # normalized width (0-1)
                height_norm = outputs[0][i][3]    # normalized height (0-1)
                
                # Convert to pixel coordinates in the 640x640 space, then scale to original
                center_x = center_x_norm * self._resize * scale
                center_y = center_y_norm * self._resize * scale
                box_width = width_norm * self._resize * scale
                box_height = height_norm * self._resize * scale
                
                # Convert center coordinates to top-left corner
                x0 = int(center_x - (box_width / 2))
                y0 = int(center_y - (box_height / 2))
                w = int(box_width)
                h = int(box_height)
                
                boxes.append([x0, y0, w, h])
                confidences.append(float(maxScore))
                class_ids.append(maxClassIndex)

        # Apply Non-Maximum Suppression
        indices = cv2.dnn.NMSBoxes(boxes, confidences, 0.3, nms_threshold)
        
        if len(indices) > 0:
            for i in indices.flatten():
                x0, y0, w, h = boxes[i]
                yield [x0, y0, w, h, class_ids[i], confidences[i]]


    def detect_yolo_with_averaging(self, original_image, iou_threshold=0.5):
        outputs, scale = self._scale_input(original_image)
        rows = outputs.shape[1]

        # Collect all detections
        detections = []

        # Iterate through output to collect bounding boxes, confidence scores, and class IDs
        for i in range(rows):
            classes_scores = outputs[0][i][4:]
            (minScore, maxScore, minClassLoc, (x, maxClassIndex)) = cv2.minMaxLoc(
                classes_scores
            )
            if maxScore >= 0.3 and maxClassIndex in self.detect_only_yolo_class_id:
                center_x_norm = outputs[0][i][0]  # normalized center x (0-1)
                center_y_norm = outputs[0][i][1]  # normalized center y (0-1)
                width_norm = outputs[0][i][2]     # normalized width (0-1)
                height_norm = outputs[0][i][3]    # normalized height (0-1)
                
                # Convert to pixel coordinates in the 640x640 space, then scale to original
                center_x = center_x_norm * self._resize * scale
                center_y = center_y_norm * self._resize * scale
                box_width = width_norm * self._resize * scale
                box_height = height_norm * self._resize * scale
                
                # Convert center coordinates to top-left corner
                x0 = int(center_x - (box_width / 2))
                y0 = int(center_y - (box_height / 2))
                w = int(box_width)
                h = int(box_height)
                
                detections.append({
                    'box': [x0, y0, w, h],
                    'class_id': maxClassIndex,
                    'confidence': maxScore,
                    'center_x': center_x,
                    'center_y': center_y
                })

        # Group overlapping detections by class
        grouped_detections = {}
        for detection in detections:
            class_id = detection['class_id']
            if class_id not in grouped_detections:
                grouped_detections[class_id] = []
            grouped_detections[class_id].append(detection)

        # Process each class separately
        for class_id, class_detections in grouped_detections.items():
            processed = [False] * len(class_detections)
            
            for i, detection in enumerate(class_detections):
                if processed[i]:
                    continue
                    
                # Find all overlapping detections
                overlapping = [detection]
                processed[i] = True
                
                for j, other_detection in enumerate(class_detections):
                    if processed[j] or i == j:
                        continue
                        
                    # Calculate IoU
                    iou = calculate_iou(detection['box'], other_detection['box'])
                    if iou > iou_threshold:
                        overlapping.append(other_detection)
                        processed[j] = True
                
                # Average the overlapping detections
                if len(overlapping) > 1:
                    # Method 1: Average coordinates
                    avg_center_x = sum(d['center_x'] for d in overlapping) / len(overlapping)
                    avg_center_y = sum(d['center_y'] for d in overlapping) / len(overlapping)
                    max_confidence = max(d['confidence'] for d in overlapping)
                    
                    # Use the box with highest confidence for width/height
                    best_detection = max(overlapping, key=lambda d: d['confidence'])
                    w, h = best_detection['box'][2], best_detection['box'][3]
                    
                    # Convert back to top-left coordinates
                    x0 = int(avg_center_x - w / 2)
                    y0 = int(avg_center_y - h / 2)
                    
                    yield [x0, y0, w, h, class_id, max_confidence]
                else:
                    # Single detection, return as is
                    x0, y0, w, h = detection['box']
                    yield [x0, y0, w, h, class_id, detection['confidence']]


    def detect_yolo_with_largest_box(self, original_image, iou_threshold=0.5):
        outputs, scale = self._scale_input(original_image)
        rows = outputs.shape[1]

        # Collect all detections
        detections = []

        # Iterate through output to collect bounding boxes, confidence scores, and class IDs
        for i in range(rows):
            classes_scores = outputs[0][i][4:]
            (minScore, maxScore, minClassLoc, (x, maxClassIndex)) = cv2.minMaxLoc(
                classes_scores
            )
            if maxScore >= 0.3 and maxClassIndex in self.detect_only_yolo_class_id:
                center_x_norm = outputs[0][i][0]  # normalized center x (0-1)
                center_y_norm = outputs[0][i][1]  # normalized center y (0-1)
                width_norm = outputs[0][i][2]     # normalized width (0-1)
                height_norm = outputs[0][i][3]    # normalized height (0-1)
                
                # Convert to pixel coordinates in the 640x640 space, then scale to original
                center_x = center_x_norm * self._resize * scale
                center_y = center_y_norm * self._resize * scale
                box_width = width_norm * self._resize * scale
                box_height = height_norm * self._resize * scale
                
                # Convert center coordinates to top-left corner
                x0 = int(center_x - (box_width / 2))
                y0 = int(center_y - (box_height / 2))
                w = int(box_width)
                h = int(box_height)
                
                detections.append({
                    'box': [x0, y0, w, h],
                    'class_id': maxClassIndex,
                    'confidence': maxScore,
                    'area': w * h
                })

        # Group overlapping detections by class
        grouped_detections = {}
        for detection in detections:
            class_id = detection['class_id']
            if class_id not in grouped_detections:
                grouped_detections[class_id] = []
            grouped_detections[class_id].append(detection)

        # Process each class separately
        for class_id, class_detections in grouped_detections.items():
            processed = [False] * len(class_detections)
            
            for i, detection in enumerate(class_detections):
                if processed[i]:
                    continue
                    
                # Find all overlapping detections
                overlapping = [detection]
                processed[i] = True
                
                for j, other_detection in enumerate(class_detections):
                    if processed[j] or i == j:
                        continue
                        
                    # Calculate IoU
                    iou = calculate_iou(detection['box'], other_detection['box'])
                    if iou > iou_threshold:
                        overlapping.append(other_detection)
                        processed[j] = True
                
                # Keep the largest box (or highest confidence if similar sizes)
                if len(overlapping) > 1:
                    # Sort by area (descending), then by confidence (descending)
                    best_detection = max(overlapping, key=lambda d: (d['area'], d['confidence']))
                    x0, y0, w, h = best_detection['box']
                    yield [x0, y0, w, h, class_id, best_detection['confidence']]
                else:
                    # Single detection, return as is
                    x0, y0, w, h = detection['box']
                    yield [x0, y0, w, h, class_id, detection['confidence']]

