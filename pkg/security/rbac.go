package security

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"lemon/lemon-api/pkg/config"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	ErrInvalidAccount       = errors.New("invalid account")
	ErrAccountAlreadyExists = errors.New("account already exists")
	ErrInvalidToken         = errors.New("invalid token")
	ErrTokenExpired         = errors.New("token has expired")
	ErrTokenNotYetValid     = errors.New("token not yet valid")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrAccountLocked        = errors.New("account locked")
	ErrAccountNotVerified   = errors.New("account not verified")
)

func Authenticate(cfg *config.Config, redirectOnFailure bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.Security.Enforce {
			token := ""

			if cookie, err := c.Cookie("haia-token"); err == nil {
				token = cookie
			} else {
				header := c.GetHeader("Authorization")
				headerParts := strings.Split(header, " ")

				if headerParts[0] != "Bearer" {
					log.Warn("invalid header")
					if redirectOnFailure {
						c.Redirect(http.StatusTemporaryRedirect, cfg.Security.Redirect)
						c.Abort()
					} else {
						c.AbortWithStatus(http.StatusUnauthorized)
					}

					return
				}

				token = headerParts[1]

				if token == "" {
					log.Warn("unauthorised: missing token")
					if redirectOnFailure {
						c.Redirect(http.StatusTemporaryRedirect, cfg.Security.Redirect)
						c.Abort()
					} else {
						c.AbortWithStatus(http.StatusUnauthorized)
					}

					return
				}
			}

			tkn, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, nil
				}

				return []byte(cfg.Security.Secret), nil
			})
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Warn("unauthorised: parsing token")
				if redirectOnFailure {
					c.Redirect(http.StatusTemporaryRedirect, cfg.Security.Redirect)
					c.Abort()
				} else {
					c.AbortWithStatus(http.StatusUnauthorized)
				}

				return
			}

			if claims, ok := tkn.Claims.(jwt.MapClaims); ok && tkn.Valid {
				now := time.Now().UTC().Unix()
				expiry := int64(claims["exp"].(float64))
				nbf := int64(claims["nbf"].(float64))

				if now > expiry {
					log.Warn("unauthorised: expired token")
					if redirectOnFailure {
						c.Redirect(http.StatusTemporaryRedirect, cfg.Security.Redirect)
						c.Abort()
					} else {
						c.AbortWithStatus(http.StatusUnauthorized)
					}

					return
				}

				if now < nbf {
					log.Warn("unauthorised: token not yet valid")
					if redirectOnFailure {
						c.Redirect(http.StatusTemporaryRedirect, cfg.Security.Redirect)
						c.Abort()
					} else {
						c.AbortWithStatus(http.StatusUnauthorized)
					}

					return
				}

				jwtID := string(claims["id"].(string))
				jwtName := string(claims["name"].(string))

				c.Set("jwt_id", jwtID)
				c.Set("jwt_email", jwtName)
			}
		}

		c.Next()
	}
}

func RBAC(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
