package rest

import (
	"fmt"
	lemon_api "lemon/lemon-api"
	"math/rand"
	"net/http"
	"os"
	"time"

	"lemon/lemon-api/pkg/config"
	"lemon/lemon-api/pkg/postgres"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type Server struct {
	config   *config.Config
	engine   *gin.Engine
	database *postgres.Service
}

func NewServer(cfg *config.Config, e *gin.Engine) *Server {
	rand.Seed(time.Now().UTC().UnixNano())

	return &Server{
		config: cfg,
		engine: e,
	}
}

func (s *Server) Initialise() {

	s.engine.POST("/api/lemon-test", s.LemonTest)
	s.engine.POST("api/new-feedback", s.InsertFeedback)

	if service, err := postgres.NewService(s.config); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to start database service")
		return
	} else {
		s.database = service
	}

	var filename = "logfile.log"
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	log.SetFormatter(&log.JSONFormatter{})
	if err != nil {
		fmt.Println(err)
	} else {
		log.SetOutput(f)
	}

}

func (s *Server) LemonTest(c *gin.Context) {
	log.WithFields(log.Fields{
		"TEST": "BOIIIIII",
	}).Info("This is text")

	logJSON := struct {
		Spoof int64 `json:"spoof"`
	}{
		Spoof: 69,
	}
	c.JSON(http.StatusOK, logJSON)
}

func (s *Server) InsertFeedback(c *gin.Context) {
	var feedback lemon_api.Feedback
	if err := c.BindJSON(&feedback); err != nil {
		log.WithFields(log.Fields{
			"err":err,
			"data":feedback,
		}).Error("Failed to bind JSON")
		c.AbortWithStatus(http.StatusBadRequest)
	}

	_, err := s.database.InsertFeedback(feedback)
	if err != nil {
		log.WithFields(log.Fields{
			"err":err,
		}).Error("Failed to insert feedback")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	c.AbortWithStatus(http.StatusOK)
}
