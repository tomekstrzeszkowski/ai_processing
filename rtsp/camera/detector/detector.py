import onnxruntime as ort
import numpy as np
import cv2
import multiprocessing
from functools import cached_property
from detector.const import YoloObject, YOLO_MODEL_NAME_TO_SCALE_TO_ORIGINAL, ACTIVE_MODEL
from detector.utils import calculate_iou


class Detector:
    yolo_model_name = ACTIVE_MODEL
    detect_only = (YoloObject.PERSON, YoloObject.CAR)
    detect_only_yolo_class_id = [yolo.value for yolo in detect_only]
    yolo_class_id_to_verbose = {yolo.value: yolo.name.lower() for yolo in detect_only}
    last_detection = None

    @property
    def _resize(self):
        return YOLO_MODEL_NAME_TO_SCALE_TO_ORIGINAL.get(self.yolo_model_name, 1)[0]

    @property
    def _export_size(self):
        return YOLO_MODEL_NAME_TO_SCALE_TO_ORIGINAL.get(self.yolo_model_name, 1)[1]


    @cached_property
    def model(self):
        session_options = ort.SessionOptions()
        num_threads = multiprocessing.cpu_count()
        session_options.intra_op_num_threads = num_threads
        session_options.inter_op_num_threads = num_threads
        return ort.InferenceSession(
            self.yolo_model_name,
            providers=['CPUExecutionProvider']
        )

    @cached_property
    def input_name(self):
        """Get the input tensor name"""
        return self.model.get_inputs()[0].name

    def _scale_input(self, original_image):
        [height, width, _] = original_image.shape
        length = max((height, width))
        image = np.zeros((length, length, 3), np.uint8)
        image[0:height, 0:width] = original_image

        size = self._export_size
        scale = length / size
        
        blob = cv2.dnn.blobFromImage(
            image, scalefactor=1 / 255, size=(size, size), swapRB=True
        )
        outputs = self.model.run(None, {self.input_name: blob})
        outputs = np.array(outputs[0])
        if len(outputs.shape) == 3 and outputs.shape[1] < outputs.shape[2]:
            outputs = np.transpose(outputs, (0, 2, 1))
        
        return outputs, scale

    def detect_yolo_all(self, original_image):
        """Detect all objects without filtering duplicates"""
        outputs, scale = self._scale_input(original_image)
        
        # outputs shape: [1, N, 85] for YOLOv5 or [1, N, 84] for YOLOv8/v11
        batch_size, num_detections, num_features = outputs.shape
        self.last_detection = []

        # Iterate through detections
        for i in range(num_detections):
            detection = outputs[0][i]
            if num_features == 85:  # YOLOv5 format
                center_x_norm = detection[0]
                center_y_norm = detection[1]
                width_norm = detection[2]
                height_norm = detection[3]
                objectness = detection[4]
                classes_scores = detection[5:]
                max_class_score = np.max(classes_scores)
                maxScore = objectness * max_class_score
                maxClassIndex = int(np.argmax(classes_scores))
            else:  # YOLOv8/v11 format (84 values)
                center_x_norm = detection[0]
                center_y_norm = detection[1]
                width_norm = detection[2]
                height_norm = detection[3]
                classes_scores = detection[4:]
                maxScore = np.max(classes_scores)
                maxClassIndex = int(np.argmax(classes_scores))
            
            if maxScore >= 0.34 and maxClassIndex in self.detect_only_yolo_class_id:
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
                last_result = [x0, y0, w, h, maxClassIndex, float(maxScore)]
                self.last_detection.append(last_result)
                yield last_result

    def detect_yolo_with_nms(self, original_image, nms_threshold=0.4):
        """
        Non-Maximum Suppression remove duplicates.

        Keep only coordinates with highest confidence among overlapping
        boxes.
        """
        outputs, scale = self._scale_input(original_image)
        
        batch_size, num_detections, num_features = outputs.shape

        # Collect all detections
        boxes = []
        confidences = []
        class_ids = []
        self.last_detection = []

        # Iterate through detections
        for i in range(num_detections):
            detection = outputs[0][i]
            
            # Handle both YOLOv5 (with objectness) and YOLOv8/v11 (without objectness)
            if num_features == 85:  # YOLOv5 format
                center_x_norm = detection[0]
                center_y_norm = detection[1]
                width_norm = detection[2]
                height_norm = detection[3]
                objectness = detection[4]
                classes_scores = detection[5:]
                max_class_score = np.max(classes_scores)
                maxScore = objectness * max_class_score
                maxClassIndex = int(np.argmax(classes_scores))
            else:  # YOLOv8/v11 format (84 values)
                center_x_norm = detection[0]
                center_y_norm = detection[1]
                width_norm = detection[2]
                height_norm = detection[3]
                classes_scores = detection[4:]
                maxScore = np.max(classes_scores)
                maxClassIndex = int(np.argmax(classes_scores))
            
            if maxScore >= 0.3 and maxClassIndex in self.detect_only_yolo_class_id:
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
        if len(boxes) > 0:
            indices = cv2.dnn.NMSBoxes(boxes, confidences, 0.3, nms_threshold)
            
            if len(indices) > 0:
                for i in indices.flatten():
                    x0, y0, w, h = boxes[i]
                    last_result = [x0, y0, w, h, class_ids[i], confidences[i]]
                    self.last_detection.append(last_result)
                    yield last_result

    def detect_yolo_with_averaging(self, original_image, iou_threshold=0.5):
        """Average overlapping detections for smoother results"""
        outputs, scale = self._scale_input(original_image)
        
        batch_size, num_detections, num_features = outputs.shape

        # Collect all detections
        detections = []
        self.last_detection = []

        # Iterate through detections
        for i in range(num_detections):
            detection = outputs[0][i]
            
            # Handle both YOLOv5 (with objectness) and YOLOv8/v11 (without objectness)
            if num_features == 85:  # YOLOv5 format
                center_x_norm = detection[0]
                center_y_norm = detection[1]
                width_norm = detection[2]
                height_norm = detection[3]
                objectness = detection[4]
                classes_scores = detection[5:]
                max_class_score = np.max(classes_scores)
                maxScore = objectness * max_class_score
                maxClassIndex = int(np.argmax(classes_scores))
            else:  # YOLOv8/v11 format (84 values)
                center_x_norm = detection[0]
                center_y_norm = detection[1]
                width_norm = detection[2]
                height_norm = detection[3]
                classes_scores = detection[4:]
                maxScore = np.max(classes_scores)
                maxClassIndex = int(np.argmax(classes_scores))
            
            if maxScore >= 0.3 and maxClassIndex in self.detect_only_yolo_class_id:
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
                    'confidence': float(maxScore),
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
                    last_result = [x0, y0, w, h, class_id, max_confidence]
                    self.last_detection.append(last_result)
                    yield last_result
                else:
                    # Single detection, return as is
                    x0, y0, w, h = detection['box']
                    last_result = [x0, y0, w, h, class_id, detection['confidence']]
                    self.last_detection.append(last_result)
                    yield last_result

    def detect_yolo_with_largest_box(self, original_image, iou_threshold=0.5):
        """Keep only the largest box among overlapping detections"""
        outputs, scale = self._scale_input(original_image)
        
        batch_size, num_detections, num_features = outputs.shape

        # Collect all detections
        detections = []
        self.last_detection = []

        # Iterate through detections
        for i in range(num_detections):
            detection = outputs[0][i]
            
            # Handle both YOLOv5 (with objectness) and YOLOv8/v11 (without objectness)
            if num_features == 85:  # YOLOv5 format
                center_x_norm = detection[0]
                center_y_norm = detection[1]
                width_norm = detection[2]
                height_norm = detection[3]
                objectness = detection[4]
                classes_scores = detection[5:]
                max_class_score = np.max(classes_scores)
                maxScore = objectness * max_class_score
                maxClassIndex = int(np.argmax(classes_scores))
            else:  # YOLOv8/v11 format (84 values)
                center_x_norm = detection[0]
                center_y_norm = detection[1]
                width_norm = detection[2]
                height_norm = detection[3]
                classes_scores = detection[4:]
                maxScore = np.max(classes_scores)
                maxClassIndex = int(np.argmax(classes_scores))
            
            if maxScore >= 0.3 and maxClassIndex in self.detect_only_yolo_class_id:
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
                    'confidence': float(maxScore),
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
                    last_result = [x0, y0, w, h, class_id, best_detection['confidence']]
                    self.last_detection.append(last_result)
                    yield last_result
                else:
                    # Single detection, return as is
                    x0, y0, w, h = detection['box']
                    last_result = [x0, y0, w, h, class_id, detection['confidence']]
                    self.last_detection.append(last_result)
                    yield last_result