package main

func main() {
	setupLogger()
	initShortID()
	err:= initAssets()
	if err != nil {
		logger.Info("Cannot initialize assets: ", err)
		return
	}
	startBackLogTicker()
	err = initDirectives()
	if err != nil {
		logger.Info("Cannot initialize directives: ", err)
		return
	}
	startServer()
}
