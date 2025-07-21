import struct
import os


def write_frame_to_shared_memory(buffer, type_, shm_name='video_frame'):
    """Save frame buffer to shared memory.""" 
    data = buffer.tobytes()
    header = struct.pack('<bI', type_, len(data))
    shm_path = f'/dev/shm/{shm_name}'
    
    # Write frame data directly (no size header needed)
    temp_path = f'{shm_path}.tmp'
    with open(temp_path, 'wb') as f:
        f.write(header)
        f.write(data)
    
    # Atomic rename
    os.rename(temp_path, shm_path)