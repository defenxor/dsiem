package main

var progDir string

func init() {
	d, _ := getDir()
	progDir = d
}

func main() {
	setupLogger()
	initShortID()
	err := initAssets()
	if err != nil {
		logger.Info("Cannot initialize assets: ", err)
		return
	}
	err = initDirectives()
	if err != nil {
		logger.Info("Cannot initialize directives: ", err)
		return
	}
	startBackLogTicker()
	startServer()
}
