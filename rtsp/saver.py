import struct
import os


def write_frame_to_shared_memory(buffer, shm_name='video_frame'):
    """Save frame buffer to shared memory.

    Convert back: tail -c +5 video_frame > video_frame.jpg
    """
    data = buffer.tobytes()
    shm_path = f'/dev/shm/{shm_name}'
    
    # Write frame data with size header
    frame_data = struct.pack('!I', len(data)) + data
    
    # Write atomically using temporary file
    temp_path = f'{shm_path}.tmp'
    with open(temp_path, 'wb') as f:
        f.write(frame_data)
    
    # Atomic rename
    os.rename(temp_path, shm_path)
