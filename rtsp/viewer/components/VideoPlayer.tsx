import React, { useState } from 'react';

interface VideoPlayerProps {
  videoUrl: string;
}

export const VideoPlayer: React.FC<VideoPlayerProps> = ({ videoUrl }) => {
  const [error, setError] = useState<string | null>(null);

  if (!videoUrl) {
    return <div className="p-4">No video selected</div>;
  }

  return (
    <div className="p-4">
      <h2 className="text-xl font-bold mb-4">Video Player</h2>
      {error ? (
        <div className="text-red-500">{error}</div>
      ) : (
        <video 
          src={videoUrl} 
          controls 
          className="w-full max-w-2xl"
          onError={(e) => {
            console.error('Video error:', e);
            setError('Error playing video');
          }}
        />
      )}
    </div>
  );
};