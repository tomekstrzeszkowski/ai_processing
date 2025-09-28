import React, { useCallback, useEffect, useRef, useState } from 'react';

interface CachedVideoPlayerProps {
  imageUri: string | null;
  frameCountRef: React.RefObject<number>;
  styles: any;
}

export const CachedVideoPlayer: React.FC<CachedVideoPlayerProps> = ({ imageUri, frameCountRef, styles }) => {
  const [displayUri, setDisplayUri] = useState<string | null>(null); // Start as null
  const [isLoading, setIsLoading] = useState(false);
  const [hasInitialized, setHasInitialized] = useState(false); // Track initialization
  
  // Use refs to directly manipulate DOM and avoid React keeping references
  const containerRef = useRef<HTMLDivElement | null>(null);
  
  // Keep track of created images for cleanup
  const imageRefsSet = useRef<Set<HTMLImageElement>>(new Set());
  const forceCleanupImage = useCallback((imgElement: HTMLImageElement | null) => {
    if (!imgElement) return;
    
    try {
      // Remove from our tracking set
      imageRefsSet.current.delete(imgElement);
      
      // Clear all image properties that hold memory
      imgElement.src = '';
      imgElement.srcset = '';
      imgElement.onload = null;
      imgElement.onerror = null;
      imgElement.onabort = null;
      
      // Force remove from DOM if still attached
      if (imgElement.parentNode) {
        imgElement.parentNode.removeChild(imgElement);
      }
      
      // Set to null to break references
      imgElement = null;
    } catch (error) {
      console.warn('Image cleanup error:', error);
    }
  }, []);

  // Cleanup all tracked images
  const cleanupAllImages = useCallback(() => {
    imageRefsSet.current.forEach(img => {
      forceCleanupImage(img);
    });
    imageRefsSet.current.clear();
  }, [forceCleanupImage]);

  const performCrossfade = useCallback((newImg: HTMLImageElement, oldImg: HTMLImageElement | null) => {
    if (!containerRef.current) return;
    
    // Setup new image styles
    newImg.style.width = '100%';
    newImg.style.height = '100%';
    newImg.style.objectFit = 'contain';
    newImg.style.position = 'absolute';
    newImg.style.top = '0';
    newImg.style.left = '0';
    newImg.style.opacity = '0';
    newImg.style.transition = 'opacity 100ms linear';
    
    // Add new image to container
    containerRef.current.appendChild(newImg);
    
    // Force reflow to ensure transition works
    newImg.offsetHeight;
    
    // Fade in new image
    requestAnimationFrame(() => {
      newImg.style.opacity = '1';
      
      // If there's an old image, fade it out simultaneously
      if (oldImg && oldImg.parentNode) {
        oldImg.style.transition = 'opacity 100ms linear';
        oldImg.style.opacity = '0';
        
        // Remove old image after transition
        setTimeout(() => {
          forceCleanupImage(oldImg);
        }, 120); // Slightly longer than transition
      }
    });
  }, [forceCleanupImage]);

  const createAndLoadImage = useCallback((uri: string, onSuccess: (img: HTMLImageElement) => void, onError: () => void) => {
    const img = document.createElement('img');
    imageRefsSet.current.add(img);
    
    img.onload = () => {
      onSuccess(img);
    };
    
    img.onerror = () => {
      forceCleanupImage(img);
      onError();
    };
    img.src = uri;
    return img;
  }, [forceCleanupImage]);

  const createInitialImage = useCallback((uri: string) => {
    if (!containerRef.current) return;
    
    setIsLoading(true);
    
    createAndLoadImage(
      uri,
      (img: HTMLImageElement) => {
        if (containerRef.current) {
          // Style the initial image
          img.style.width = '100%';
          img.style.height = '100%';
          img.style.objectFit = 'contain';
          img.style.position = 'absolute';
          img.style.top = '0';
          img.style.left = '0';
          img.style.opacity = '1';
          
          containerRef.current.appendChild(img);
          setDisplayUri(uri);
          setHasInitialized(true);
          setIsLoading(false);
        }
      },
      () => {
        console.error('Failed to load initial image:', uri);
        setIsLoading(false);
        // Don't set hasInitialized to true on error
      }
    );
  }, [createAndLoadImage]);
  useEffect(() => {
    if (imageUri && !hasInitialized && containerRef.current) {
      createInitialImage(imageUri);
    }
  }, [imageUri, hasInitialized, createInitialImage]);

  // Handle subsequent image changes with crossfade
  useEffect(() => {
    if (imageUri && hasInitialized && imageUri !== displayUri && containerRef.current) {
      setIsLoading(true);
      const currentImg = containerRef.current.querySelector('img') as HTMLImageElement | null;
      createAndLoadImage(
        imageUri,
        (newImg: HTMLImageElement) => {
          // Success - perform crossfade
          if (containerRef.current) {
            performCrossfade(newImg, currentImg);
            setDisplayUri(imageUri);
            setIsLoading(false);
          }
        },
        () => {
          // Error
          console.error('Failed to load image:', imageUri);
          setIsLoading(false);
        }
      );
    }
  }, [imageUri, displayUri, hasInitialized, createAndLoadImage, performCrossfade]);

  // CRITICAL: Cleanup on unmount
  useEffect(() => {
    return () => {
      cleanupAllImages();
    };
  }, [cleanupAllImages]);

  if (!imageUri) {
    return (
      <div style={styles.noVideoContainer}>
        <img
          src="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMjQiIGhlaWdodD0iMjQiIHZpZXdCb3g9IjAgMCAyNCAyNCIgZmlsbD0ibm9uZSIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj4KPHBhdGggZD0iTTEyIDJMMTMuMDkgOC4yNkwyMCA5TDEzLjA5IDE1Ljc0TDEyIDIyTDEwLjkxIDE1Tjc0TDQgOUwxMC45MSA4LjI2TDEyIDJaIiBmaWxsPSIjNjY2Ii8+Cjwvc3ZnPgo="
          style={styles.placeholderIcon}
          alt="placeholder"
        />
      </div>
    );
  }

  return (
    <div 
      style={{
        ...styles.videoContainer,
        position: 'relative',
        overflow: 'hidden'
      }} 
      ref={containerRef}
    >
      {isLoading && (
        <div style={{
          position: 'absolute',
          top: '50%',
          left: '50%',
          transform: 'translate(-50%, -50%)',
          zIndex: 1
        }}>
          Loading...
        </div>
      )}
    </div>
  );
};