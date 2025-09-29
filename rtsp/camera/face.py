from facenet_pytorch import MTCNN


class FaceDetector:
	def __init__(self, device):
		self.mtcnn = MTCNN(keep_all=True, device=device)
	def detect(self, frame):
		return mtcnn.detect(frame_array)[0]