package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"strzcam.com/broadcaster/web_rtc"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}
	signalingServerUrl := os.Getenv("SIGNALING_URL")
	web_rtc.RunLive(fmt.Sprintf("ws://%s/ws?userId=99", signalingServerUrl))
}
