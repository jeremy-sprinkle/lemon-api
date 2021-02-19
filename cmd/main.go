package main

import (
	"fmt"

	"lemon/lemon-api/pkg/config"
	"lemon/lemon-api/pkg/rest"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Info("Started Lemon API Server")

	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.WithFields(log.Fields{
			"path":  "config.json",
			"error": err,
		}).Error("error loading config")
		return
	}

	webEngine := gin.New()
	if webEngine == nil {
		log.WithFields(log.Fields{
			"error": "unable to create gin engine",
		}).Fatal("Unable to create Gin engine")
		return
	}

	w := rest.NewServer(cfg, webEngine)
	if w == nil {
		log.Fatal("Unable to create web server")
		return
	}

	w.Initialise()

	log.WithFields(log.Fields{
		"port": cfg.API.Port,
	}).Info("Lemon API Listening")
	err = webEngine.Run(fmt.Sprintf("%v:%v", "0.0.0.0", cfg.API.Port))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("unable to start HTTP interface")
	}
}
