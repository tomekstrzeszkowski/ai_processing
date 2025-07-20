FROM python:3.11-slim

RUN apt update && apt install ffmpeg libsm6 libxext6  -y

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

# avoid coping it each time docker is re-builded
COPY *.onnx .
COPY *.mp4 .
COPY requirements.txt .

RUN pip install --upgrade pip \
    && pip install -r requirements.txt

COPY *.py .

CMD ["python"]