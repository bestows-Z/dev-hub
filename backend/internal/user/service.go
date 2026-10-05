package user

import (
	"context"
	"fmt"
)

type Service struct {
	repository Repository
}

func NewService(
	repository Repository,
) *Service {
	return &Service{
		repository: repository,
	}
}

// GetByID 查询用户资料。
func (s *Service) GetByID(
	ctx context.Context,
	id uint64,
) (*Response, error) {
	u, err := s.repository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get user: %w",
			err,
		)
	}

	result := ToResponse(u)

	return &result, nil
}
