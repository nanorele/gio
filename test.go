package main

import "fmt"

type FrameEvent struct {}
type frameEvent struct { FrameEvent }

func main() {
	var evt interface{} = frameEvent{}
	switch evt.(type) {
	case FrameEvent:
		fmt.Println("Matched FrameEvent")
	case frameEvent:
		fmt.Println("Matched frameEvent")
	default:
		fmt.Println("Matched default")
	}
}
