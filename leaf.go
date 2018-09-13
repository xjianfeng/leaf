package leaf

import (
	"server/leaf/cluster"
	"server/leaf/conf"
	"server/leaf/console"
	"server/leaf/log"
	"server/leaf/module"
	"os"
	"os/signal"
	"syscall"
)

func Run(mods ...module.Module) {
	// logger
	if conf.LogLevel != "" {
		logPath := conf.LogPath
		if conf.LogLevel == "debug" {
			logPath = ""
		}
		logger, err := log.New(conf.LogLevel, logPath, "leaf", conf.LogFlag)
		if err != nil {
			panic(err)
		}
		log.Export(logger)
		defer logger.Close()
	}

	log.Release("Leaf %v starting up", version)

	// module
	for i := 0; i < len(mods); i++ {
		module.Register(mods[i])
	}
	module.Init()

	// cluster
	cluster.Init()

	// console
	console.Init()

	// close
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGTERM)
	sig := <-c
	log.Release("Leaf closing down (signal: %v)", sig)
	console.Destroy()
	cluster.Destroy()
	module.Destroy()
}
