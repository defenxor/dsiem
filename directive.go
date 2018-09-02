package main

func directiveChanController() {
	var total = 3000
	var dirchan []chan normalizedEvent
	logger.Info("Creating ", total, " directives.")
	eventChannel = make(chan normalizedEvent)
	for i := 0; i < total; i++ {
		dirchan = append(dirchan, make(chan normalizedEvent))
		go directive(i, dirchan[i])
		go func() {
			for {
				evt := <-eventChannel
				for i := range dirchan {
					dirchan[i] <- evt
				}
			}
		}()
	}
}

func directive(id int, c chan normalizedEvent) {
	logger.Info("started directive ", id)

	// should setup pipeline here with first input from chan c
	//stdout := processors.NewIoWriter(os.Stdout)
	//upperCaser := processors.NewFuncTransformer(func(d data.JSON) data.JSON {
	//	return data.JSON(strings.ToUpper(string(d)))
	//})
	//	pipeline := ratchet.NewPipeline(upperCaser, stdout)

	// Finally, run the Pipeline and wait for either an error or nil to be returned
	//	err := <-pipeline.Run()
	//	if err != nil {
	//		return
	//	}

	for {
		evt := <-c
		logger.Info("directive ", id, " received data from dirchan: ", evt)
	}
}
