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
)

func Authenticate(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.Security.Enforce {
			token := ""

			if cookie, err := c.Cookie("lemon-token"); err == nil {
				token = cookie
			} else {
				header := c.GetHeader("Authorization")
				headerParts := strings.Split(header, " ")

				if headerParts[0] != "Bearer" {
					log.Warn("invalid header")
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}

				token = headerParts[1]

				if token == "" {
					log.Warn("unauthorised: missing token")
					c.AbortWithStatus(http.StatusUnauthorized)
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
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			if claims, ok := tkn.Claims.(jwt.MapClaims); ok && tkn.Valid {
				now := time.Now().UTC().Unix()
				expiry := int64(claims["exp"].(float64))
				nbf := int64(claims["nbf"].(float64))

				if now > expiry {
					log.Warn("unauthorised: expired token")
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}

				if now < nbf {
					log.Warn("unauthorised: token not yet valid")
					c.AbortWithStatus(http.StatusUnauthorized)
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

func GetTokenAccountID(cfg *config.Config, token string) (*string, error) {
	tkn, err := VerifyToken(cfg, token)
	if err != nil {
		return nil, err
	}

	if claims, ok := tkn.Claims.(jwt.MapClaims); ok && tkn.Valid {
		if accountID, ok := claims["id"].(string); ok {
			return &accountID, nil
		}
	}

	return nil, ErrInvalidToken
}

func VerifyToken(cfg *config.Config, token string) (*jwt.Token, error) {
	token = strings.Split(token, "Bearer ")[1]
	tkn, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}

		return []byte(cfg.Security.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := tkn.Claims.(jwt.MapClaims); ok && tkn.Valid {
		now := time.Now().Unix()
		expiry := int64(claims["exp"].(float64))
		nbf := int64(claims["nbf"].(float64))
		if now > expiry {
			return nil, ErrTokenExpired
		}

		if now < nbf {
			return nil, ErrTokenNotYetValid
		}
	}

	return tkn, nil
}
