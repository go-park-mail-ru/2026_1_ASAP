package s3log

import (
	"context"
	"time"

	"github.com/go-park-mail-ru/2026_1_ASAP/pkg/sqllog"
	minio "github.com/minio/minio-go/v7"
	"go.uber.org/zap"
)

func LogOp(ctx context.Context, logger *zap.Logger, op, objectKey string, start time.Time, err error, args []any) {
	if logger == nil {
		return
	}
	d := time.Since(start)
	fields := []zap.Field{
		zap.String("s3_op", op),
		zap.String("object", objectKey),
		zap.Duration("duration", d),
	}
	if len(args) > 0 {
		fields = append(fields, zap.Any("args", args))
	}
	switch {
	case err != nil && minio.ToErrorResponse(err).Code != "NoSuchKey":
		logger.Warn("s3", append(fields, zap.Error(err))...)
	case err != nil:
		logger.Debug("s3", append(fields, zap.NamedError("err", err))...)
	case d >= sqllog.SlowQueryThreshold:
		logger.Warn("s3 slow", fields...)
	default:
		logger.Debug("s3", fields...)
	}
}
