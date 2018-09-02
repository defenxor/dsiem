package main

var eventChannel chan normalizedEvent

func main() {
	setupLogger()
	directiveChanController()
	startServer()
}
