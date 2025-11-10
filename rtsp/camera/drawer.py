import os
import cv2
from PIL import Image, ImageDraw, ImageFilter
from dotenv import load_dotenv
from datetime import datetime


load_dotenv()

SHOW_NOW_LABEL = bool(os.getenv("SHOW_NOW_LABEL", ""))


class Drawer:
    frame = None

    def __init__(self, frame):
        self.frame = frame
        self.rectagle_count = 0

    def label(self, motion_detected):
        now_label = (
            datetime.now().strftime("%Y-%m-%d %H:%M:%S") if SHOW_NOW_LABEL else ""
        )
        for bold, color in ((8, (0, 0, 0)), (4, (255, 255, 255))):
            cv2.putText(
                self.frame,
                f"{now_label} {self.rectagle_count}{'.' if motion_detected else ''}",
                (20, 20),
                cv2.FONT_HERSHEY_SIMPLEX,
                0.7,
                color,
                bold,
            )

    def rectangle(self, type_verbose, x0, y0, w, h):
        self.rectagle_count += 1
        cv2.rectangle(self.frame, (x0, y0), (x0 + w, y0 + h), (0, 255, 0), 2)
        for bold, color in ((5, (0, 0, 0)), (3, (255, 255, 255))):
            cv2.putText(
                self.frame,
                type_verbose.title(),
                (x0 + 10, y0 - 5),
                cv2.FONT_HERSHEY_SIMPLEX,
                0.8,
                color,
                bold,
            )

    def get_from_array(self):
        return Image.fromarray(self.frame)


def make_ellipse_mask(size, box, ellipse_blur=10):
    mask = Image.new("L", size, color=0)
    draw = ImageDraw.Draw(mask)
    x0, y0, x1, y1 = box.tolist()
    ellipse_size = [x0 * 0.95, y0 * 0.95, x1 * 1.05, y1 * 1.05]
    draw.ellipse(ellipse_size, fill=255)
    return mask.filter(ImageFilter.GaussianBlur(radius=ellipse_blur))

