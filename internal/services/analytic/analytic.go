package analytic

import (
	"context"
	"fmt"

	domain "github.com/go-park-mail-ru/2026_1_ASAP/internal/domain/analytic"
	dtoAnalytic "github.com/go-park-mail-ru/2026_1_ASAP/internal/dto/analytic"
)

type AnalyticRepo interface {
	GetUserAnalytic(ctx context.Context, userID int64) (domain.ComplaintAnalytic, error)
}

type AnalyticService struct {
	repo AnalyticRepo
}

func (a AnalyticService) GetUserComplaintAnalytic(ctx context.Context, request dtoAnalytic.RequestComplaintAnalytic) (dtoAnalytic.ResponseComplaintAnalytic, error) {
	res, err := a.repo.GetUserAnalytic(ctx, request.UserID)
	if err != nil {
		return dtoAnalytic.ResponseComplaintAnalytic{}, fmt.Errorf("get user complaint analytic: %w", err)
	}
	return dtoAnalytic.ResponseComplaintAnalytic{
		CountStatus: dtoAnalytic.CountStatus{
			CountStatusClosed: res.CountStatus.CountStatusClosed,
			CountStatusInWork: res.CountStatus.CountStatusInWork,
			CountStatusOpened: res.CountStatus.CountStatusOpened,
		},
		CountType: dtoAnalytic.CountType{
			CountBug:     res.CountType.CountBug,
			CountProduct: res.CountType.CountProduct,
			CountUpgrade: res.CountType.CountUpgrade,
		},
	}, nil
}

func NewAnalyticService(repo AnalyticRepo) *AnalyticService {
	return &AnalyticService{repo: repo}
}
