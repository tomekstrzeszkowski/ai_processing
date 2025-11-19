package watcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"strzcam.com/broadcaster/connection"
	"strzcam.com/broadcaster/video"
)

type connInfo struct {
	conn     *websocket.Conn
	writeMux sync.Mutex
}
type Command int8

const (
	Idle Command = iota
	GetVideoList
	GetVideo
)

type Server struct {
	port    uint16
	Viewers []*connection.Viewer
	Frames  chan [][]byte
}

func NewServer(port uint16) (*Server, error) {
	receiver := &Server{
		port:   port,
		Frames: make(chan [][]byte, 1),
	}
	return receiver, nil
}

func (s *Server) BroadcastFrame(frames [][]byte) {
	log.Print("Broadcasting frame of size:", len(frames[0]))
	s.Frames <- frames
}
func (s *Server) AddViewer(v *connection.Viewer) {
	for _, existingViewer := range s.Viewers {
		if existingViewer.ID == v.ID {
			return
		}
	}
	s.Viewers = append(s.Viewers, v)
}
func (s *Server) GetViewer() *connection.Viewer {
	if len(s.Viewers) > 0 {
		return s.Viewers[0]
	}
	return nil
}
func (s *Server) getVideo(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	videoName := r.PathValue("name")
	viewer := s.GetViewer()
	videoData := viewer.GetVideo(videoName)
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	reader := bytes.NewReader(videoData)
	w.WriteHeader(http.StatusOK)
	_, err := io.Copy(w, reader)
	if err != nil {
		log.Printf("Error streaming video: %v", err)
		// Note: Can't write error header here as we've already written the success header
		return
	}
}
func (s *Server) getVideoList(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	startParam := r.URL.Query().Get("start")
	endParam := r.URL.Query().Get("end")
	if startParam == "" {
		startParam = "2011-11-10"
	}
	if endParam == "" {
		endParam = "2011-11-11"
	}
	start, _ := time.Parse("2006-01-02", startParam)
	end, _ := time.Parse("2006-01-02", endParam)
	var videoList []video.Video
	viewer := s.GetViewer()
	videoList = viewer.GetVideoList(start, end)
	json.NewEncoder(w).Encode(videoList)
}

func (s *Server) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}
func (s *Server) serveStream(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "close")

	mw := multipart.NewWriter(w)
	mw.SetBoundary("frame")

	for frames := range s.Frames {
		for _, frame := range frames {
			fmt.Println("Serving frame of size:", len(frame))
			img, _, err := image.Decode(bytes.NewReader(frame))
			if err != nil {
				log.Printf("Error decoding image: %v", err)
				continue
			}

			// Write JPEG frame to multipart stream
			if err := writeJPEGFrame(mw, img); err != nil {
				log.Printf("Error writing JPEG frame: %v", err)
				return
			}

			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

func writeJPEGFrame(mw *multipart.Writer, frame image.Image) error {
	// Create JPEG buffer
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, frame, &jpeg.Options{Quality: 80}); err != nil {
		return err
	}

	// Write multipart header
	header := textproto.MIMEHeader{}
	header.Set("Content-Type", "image/jpeg")
	header.Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

	part, err := mw.CreatePart(header)
	if err != nil {
		return err
	}

	// Write JPEG data
	if _, err := part.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}
func (s *Server) PrepareEndpoints() {

	// hlsConverter, _ := NewHLSConverter("./hls_output", s.Frames)
	// go hlsConverter.Start()

	// Serve HLS files
	//http.Handle("/hls/", http.StripPrefix("/hls/", http.FileServer(http.Dir("./hls_output"))))

	http.HandleFunc("/video-list", s.getVideoList)
	http.HandleFunc("/video/{name}", s.getVideo)
	http.HandleFunc("/stream", s.serveStream)

	// Serve static files for testing
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Video Stream</title>
</head>
<body>
    <h1>Live Video Stream` + fmt.Sprintf("%d", s.port) + `</h1>
	<a href="/stream">Stream</a>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})
}
func (s *Server) Start() {
	http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}
