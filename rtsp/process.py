import cv2
import numpy as np
import time


def process_frame(frame):
    """Process individual frame - add your processing logic here"""
    # Example processing: Convert to grayscale
    gray = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)
    
    # Example: Apply Gaussian blur
    blurred = cv2.GaussianBlur(gray, (15, 15), 0)
    
    # Example: Edge detection
    edges = cv2.Canny(blurred, 50, 150)
    
    # Example: Find contours
    contours, _ = cv2.findContours(edges, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)
    
    # Draw contours on original frame
    processed_frame = frame.copy()
    cv2.drawContours(processed_frame, contours, -1, (0, 255, 0), 2)
    
    # Add text overlay
    cv2.putText(processed_frame, f"Contours: {len(contours)}", 
                (10, 30), cv2.FONT_HERSHEY_SIMPLEX, 1, (0, 255, 0), 2)
    
    return processed_frame, gray, edges

def main():
    url = "rtsp://admin:pass-here!@192.168.1.192:554/cam/realmonitor?channel=1&subtype=0"
    print(f"Connecting to camera: {url}")
    
    # Create VideoCapture object
    cap = cv2.VideoCapture(url, cv2.CAP_FFMPEG)
    
    # Set buffer size to reduce latency
    cap.set(cv2.CAP_PROP_BUFFERSIZE, 1)
    
    # Set timeout (in milliseconds)
    cap.set(cv2.CAP_PROP_OPEN_TIMEOUT_MSEC, 1 * 1000)
    
    # Check if camera opened successfully
    if not cap.isOpened():
        print("Error: Could not connect to camera")
        return None
    
    print("Successfully connected to camera")
    if cap is None:
        return
    
    # Get camera properties
    fps = cap.get(cv2.CAP_PROP_FPS)
    width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH)/8)
    height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT)/8)
    
    print(f"Camera properties: {width}x{height} @ {fps} FPS")
    
    # Performance tracking
    frame_count = 0
    start_time = time.time()
    
    try:
        while True:
            # Read frame from camera
            ret, frame = cap.read()
            
            if not ret:
                print("Failed to grab frame")
                break
            
            # Process the frame
            processed_frame, gray, edges = process_frame(frame)
            
            # Display frames
            cv2.imshow('Original', frame)
            # cv2.imshow('Processed', processed_frame)
            # cv2.imshow('Grayscale', gray)
            # cv2.imshow('Edges', edges)
            
            # Performance calculation
            frame_count += 1
            elapsed_time = time.time() - start_time
            if elapsed_time > 1.0:  # Update every second
                actual_fps = frame_count / elapsed_time
                print(f"Actual FPS: {actual_fps:.2f}")
                frame_count = 0
                start_time = time.time()
            
            # Break on 'q' key press
            key = cv2.waitKey(1) & 0xFF
            if key == ord('q'):
                break
            elif key == ord('s'):
                # Save current frame
                timestamp = int(time.time())
                cv2.imwrite(f'captured_frame_{timestamp}.jpg', frame)
                print(f"Frame saved as captured_frame_{timestamp}.jpg")
    
    except KeyboardInterrupt:
        print("Interrupted by user")
    
    finally:
        # Clean up
        cap.release()
        cv2.destroyAllWindows()
        print("Camera connection closed")

if __name__ == "__main__":
    main()