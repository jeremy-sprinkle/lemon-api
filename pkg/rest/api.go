package rest

import (
	"fmt"
	lemon_api "lemon/lemon-api"
	"math/rand"
	"net/http"
	"os"
	"strconv"
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

	s.engine.POST("api/feedback", s.InsertFeedback)
	s.engine.GET("api/feedback", s.GetFeedback)
	s.engine.PUT("api/feedback/:ID", s.MarkReadFeedback)

	s.engine.POST("api/save", s.NewUser)
	s.engine.PUT("api/save/:ID", s.UpdateUser)
	s.engine.GET("api/save/:ID", s.GetUser)
	s.engine.DELETE("api/save/:ID", s.DeleteUser)

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

func (s *Server) InsertFeedback(c *gin.Context) {
	var feedback lemon_api.Feedback
	if err := c.BindJSON(&feedback); err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"data": feedback,
		}).Error("Failed to bind JSON")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	_, err := s.database.InsertFeedback(feedback)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to insert feedback")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) GetFeedback(c *gin.Context) {
	data, err := s.database.GetFeedback()
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to get feedback from database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) MarkReadFeedback(c *gin.Context) {
	ID := c.Param("ID")
	if ID == "" {
		log.Error("No ID provided")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	feedbackID, err := strconv.ParseInt(ID, 10, 32)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to convert ID to Int")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	err = s.database.MarkReadFeedback(feedbackID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to mark read in database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) NewUser(c *gin.Context) {
	var user lemon_api.User
	if err := c.BindJSON(&user); err != nil {
		log.WithFields(log.Fields{
			"err":  err,
			"data": user,
		}).Error("Failed to bind JSON")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	token, err := s.database.NewUser(user)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to insert new user")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	c.JSON(http.StatusOK, token)
}

func (s *Server) GetUser(c *gin.Context) {
	ID := c.Param("ID")
	if ID == "" {
		log.Error("No ID provided")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	data, err := s.database.GetUser(ID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to get user from database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) UpdateUser(c *gin.Context) {
	ID := c.Param("ID")
	if ID == "" {
		log.Error("No ID provided")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	err := s.database.UpdateUser(ID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to update user in database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.AbortWithStatus(http.StatusOK)
}

func (s *Server) DeleteUser(c *gin.Context) {
	ID := c.Param("ID")
	if ID == "" {
		log.Error("No ID provided")
		c.AbortWithStatus(http.StatusBadRequest)
	}
	err := s.database.DeleteUser(ID)
	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Error("Failed to update user in database")
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.AbortWithStatus(http.StatusOK)
}
