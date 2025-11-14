package main

import "strzcam.com/broadcaster/web_rtc"

func main() {
	web_rtc.RunLive("ws://localhost:7070/ws?userId=99")
}
