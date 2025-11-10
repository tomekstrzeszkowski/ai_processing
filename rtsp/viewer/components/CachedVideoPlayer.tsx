import { useWebSocket } from '@/app/websocketProvider';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Text, View } from 'react-native';

interface CachedVideoPlayerProps {
  styles: any;
  isConnected: boolean;
}

export const CachedVideoPlayer: React.FC<CachedVideoPlayerProps> = ({ 
  styles,
  isConnected
}) => {
  const { imageUri } = useWebSocket();
  const [isLoading, setIsLoading] = useState(false);
  const [hasInitialized, setHasInitialized] = useState(false);
  
  const containerRef = useRef<HTMLDivElement | null>(null);
  const currentImageRef = useRef<HTMLImageElement | null>(null);
  const pendingImageRef = useRef<HTMLImageElement | null>(null);
  const isTransitioningRef = useRef(false);
  
  // Track the last processed URI to avoid duplicate processing
  const lastProcessedUriRef = useRef<string | null>(null);

  // Aggressive cleanup function
  const destroyImage = useCallback((img: HTMLImageElement | null) => {
    if (!img) return;
    
    try {
      // Cancel any pending loads
      img.onload = null;
      img.onerror = null;
      img.onabort = null;
      
      // Remove from DOM first
      if (img.parentNode) {
        img.parentNode.removeChild(img);
      }
      
      // Clear the src to release the base64 data from memory
      // This is CRITICAL for releasing base64 URIs
      img.removeAttribute('src');
      img.removeAttribute('srcset');
      img.src = '';
      
    } catch (error) {
      console.warn('Error destroying image:', error);
    }
  }, []);

  // Load new image with proper cleanup
  const loadNewFrame = useCallback((uri: string) => {
    // Skip if already processing this URI or if transitioning
    if (uri === lastProcessedUriRef.current || isTransitioningRef.current) {
      return;
    }
    
    // Skip if we're still loading
    if (isLoading) {
      return;
    }
    
    lastProcessedUriRef.current = uri;
    isTransitioningRef.current = true;
    setIsLoading(true);
    
    // Clean up any pending image
    if (pendingImageRef.current) {
      destroyImage(pendingImageRef.current);
      pendingImageRef.current = null;
    }
    
    // Create new image
    const newImg = document.createElement('img');
    pendingImageRef.current = newImg;
    
    newImg.onload = () => {
      if (!containerRef.current || pendingImageRef.current !== newImg) {
        destroyImage(newImg);
        return;
      }
      
      // Style the new image
      newImg.style.cssText = `
        width: 100%;
        height: 100%;
        object-fit: contain;
        position: absolute;
        top: 0;
        left: 0;
        opacity: 0;
        transition: opacity 150ms ease-in-out;
      `;
      
      // Add to container
      containerRef.current.appendChild(newImg);
      
      // Get reference to old image before updating
      const oldImg = currentImageRef.current;
      currentImageRef.current = newImg;
      pendingImageRef.current = null;
      
      // Force reflow
      newImg.offsetHeight;
      
      // Fade in new image
      const IMAGE_FADE = 800
      requestAnimationFrame(() => {
        newImg.style.opacity = '1';
        
        // Fade out old image simultaneously (if exists)
        if (oldImg && oldImg.parentNode) {
          oldImg.style.transition = `opacity ${IMAGE_FADE}ms ease-in-out`;
          oldImg.style.opacity = '0';
        }
        
        setIsLoading(false);
        isTransitioningRef.current = false;
        setHasInitialized(true);
      });
      
      // Clean up old image AFTER new one is fully visible
      if (oldImg) {
        setTimeout(() => {
          destroyImage(oldImg);
        }, IMAGE_FADE); // Wait for fade transition to complete
      }
    };
    
    newImg.onerror = () => {
      console.error('Failed to load image');
      destroyImage(newImg);
      if (pendingImageRef.current === newImg) {
        pendingImageRef.current = null;
      }
      setIsLoading(false);
      isTransitioningRef.current = false;
      lastProcessedUriRef.current = null; // Allow retry
    };
    
    // Set src last to start loading
    newImg.src = uri;
  }, [isLoading, destroyImage]);

  // Handle new frames
  useEffect(() => {
    if (imageUri && containerRef.current) {
      loadNewFrame(imageUri);
    }
  }, [imageUri, loadNewFrame]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      destroyImage(currentImageRef.current);
      destroyImage(pendingImageRef.current);
      currentImageRef.current = null;
      pendingImageRef.current = null;
    };
  }, [destroyImage]);

  return (
    <View style={styles.videoContainer}>
      <div 
        style={{
          ...styles.videoContainer,
          position: 'relative',
          overflow: 'hidden'
        }} 
        ref={containerRef}
      >
        {isLoading && !hasInitialized && (
          <div style={{
            position: 'absolute',
            top: '50%',
            left: '50%',
            transform: 'translate(-50%, -50%)',
            zIndex: 1,
            color: '#666'
          }}>
            Loading...
          </div>
        )}
      </div>
      {!imageUri && (
        <View style={styles.noVideoContainer}>
          <Text style={styles.noVideoText}>
            {isConnected ? 'Waiting for video frames...' : 'Connect to server to view stream'}
          </Text>
        </View>
      )}
    </View>
  );
};