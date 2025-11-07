package service

import (
	"context"
	"github.com/inzarubin80/Warden/internal/model"
)

func (s *PokerService) Authorization(ctx context.Context, accessToken string) (*model.Claims, error) {

	return s.accessTokenService.ValidateToken(accessToken)

}
