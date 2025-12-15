from ultralytics import YOLO
from onnxruntime.quantization import quantize_dynamic, QuantType

name = "yolov8n"
model = YOLO(f"{name}.pt")

model.export(
    format="onnx",
    imgsz=320,          # Good choice for CPU (640, 416, 320)
    simplify=True,
    opset=11,           # 11 for broader compatibility
    dynamic=False,      # For CPU optimization
    half=False          # CPU doesn't benefit from FP16
)


quantize_dynamic(
    f"{name}.onnx",
    f"{name}320_int8.onnx",
    weight_type=QuantType.QUInt8
)