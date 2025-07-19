FROM python:3.11-slim

RUN apt update && apt install ffmpeg libsm6 libxext6  -y

ENV PYTHONDONTWRITEBYTECODE=1
ENV PYTHONUNBUFFERED=1

WORKDIR /app

COPY requirements.txt .

RUN pip install --upgrade pip \
    && pip install -r requirements.txt

COPY . .

RUN rm -rf venv __pycache__

CMD ["python"]