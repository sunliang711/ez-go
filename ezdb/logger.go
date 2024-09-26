package ezdb

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	glogger "gorm.io/gorm/logger"
)

type CustomLogger struct {
	glogger.Interface
}

func (c CustomLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	log.Printf("[INFO] "+msg, data...)
}

func (c CustomLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	log.Printf("[WARN] "+msg, data...)
}

func (c CustomLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	log.Printf("[ERROR] "+msg, data...)
}

func (c CustomLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	// 对于复杂的查询参数，你可能需要进一步处理
	// 这里假设所有参数都是简单的字符串、数字等
	for _, param := range glogger.ExplainSQL(sql, nil, "") {
		sql = strings.Replace(sql, "?", fmt.Sprintf("'%v'", param), 1)
	}

	if err != nil {
		log.Printf("[rows:%v] SQL: %s |  Elapsed: %v | Error: %v", rows, sql, elapsed, err)
	} else {
		log.Printf("[rows:%v] SQL: %s |  Elapsed: %v ", rows, sql, elapsed)
	}
}
