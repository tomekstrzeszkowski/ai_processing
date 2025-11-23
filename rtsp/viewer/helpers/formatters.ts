export function formatBytes(bytes: number, decimals = 1): string {
  if (bytes === 0) return '0 Bytes';
  if (!bytes || bytes < 0) return 'Invalid size';
  
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const value = bytes / Math.pow(k, i);
  return `${value.toFixed(decimals)} ${sizes[i]}`;
}
export function formatTime(seconds: number): string {
  const totalSeconds = Math.floor(seconds);
  
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const secs = totalSeconds % 60;
  
  // Format seconds with leading zero
  const formattedSecs = secs.toString().padStart(2, '0');
  
  if (hours > 0) {
    // Format: H:MM:SS
    const formattedMins = minutes.toString().padStart(2, '0');
    return `${hours}:${formattedMins}:${formattedSecs}`;
  } else {
    // Format: M:SS
    return `${minutes}:${formattedSecs}`;
  }
}