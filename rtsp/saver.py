import struct
import os


def write_frame_to_shared_memory(buffer, shm_name='video_frame'):
    """Save frame buffer to shared memory."""
    data = buffer.tobytes()
    shm_path = f'/dev/shm/{shm_name}'
    
    # Write frame data directly (no size header needed)
    temp_path = f'{shm_path}.tmp'
    with open(temp_path, 'wb') as f:
        f.write(data)
    
    # Atomic rename
    os.rename(temp_path, shm_path)