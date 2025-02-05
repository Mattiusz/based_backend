package services

import (
	"net/http"

	"github.com/mattiusz/based_backend/internal/config"
)

type AuthService struct {
	config     *config.Config
	httpClient *http.Client
}

func NewAuthService(config *config.Config) (*AuthService, error) {

}
