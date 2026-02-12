package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"octopus/internal/model"
)

func newTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)

	return gormDB, mock
}

func TestMySQLRepository_SaveShortLink(t *testing.T) {
	db, mock := newTestDB(t)

	repo := &MySQLRepository{db: db}
	ctx := context.Background()

	t.Run("save short link successfully", func(t *testing.T) {
		sl := &model.ShortLink{
			ShortCode:   "ABCD",
			OriginalURL: "https://example.com",
			Status:      1,
		}

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `short_links`")).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.SaveShortLink(ctx, sl)
		assert.NoError(t, err)
	})

	t.Run("save short link with error", func(t *testing.T) {
		sl := &model.ShortLink{
			ShortCode:   "ABCD",
			OriginalURL: "https://example.com",
		}

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `short_links`")).
			WillReturnError(assert.AnError)
		mock.ExpectRollback()

		err := repo.SaveShortLink(ctx, sl)
		assert.Error(t, err)
	})
}

func TestMySQLRepository_GetShortLinkByCode(t *testing.T) {
	db, mock := newTestDB(t)

	repo := &MySQLRepository{db: db}
	ctx := context.Background()

	t.Run("get existing short link", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "short_code", "original_url", "params", "created_at", "expire_at", "status"}).
			AddRow(1, "ABCD", "https://example.com", nil, time.Now(), nil, 1)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `short_links` WHERE short_code = ? AND status = 1 ORDER BY `short_links`.`id` LIMIT ?")).
			WithArgs("ABCD", 1).
			WillReturnRows(rows)

		sl, err := repo.GetShortLinkByCode(ctx, "ABCD")
		assert.NoError(t, err)
		assert.NotNil(t, sl)
		assert.Equal(t, "ABCD", sl.ShortCode)
		assert.Equal(t, "https://example.com", sl.OriginalURL)
	})

	t.Run("get non-existent short link", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `short_links` WHERE short_code = ? AND status = 1 ORDER BY `short_links`.`id` LIMIT ?")).
			WithArgs("NONEXIST", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		sl, err := repo.GetShortLinkByCode(ctx, "NONEXIST")
		assert.Error(t, err)
		assert.Nil(t, sl)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestMySQLRepository_GetShortLinkByURL(t *testing.T) {
	db, mock := newTestDB(t)

	repo := &MySQLRepository{db: db}
	ctx := context.Background()

	t.Run("get by existing URL", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "short_code", "original_url", "params", "created_at", "expire_at", "status"}).
			AddRow(1, "ABCD", "https://example.com", nil, time.Now(), nil, 1)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `short_links` WHERE original_url = ? AND status = 1 ORDER BY `short_links`.`id` LIMIT ?")).
			WithArgs("https://example.com", 1).
			WillReturnRows(rows)

		sl, err := repo.GetShortLinkByURL(ctx, "https://example.com")
		assert.NoError(t, err)
		assert.NotNil(t, sl)
		assert.Equal(t, "ABCD", sl.ShortCode)
	})

	t.Run("get by non-existent URL", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `short_links` WHERE original_url = ? AND status = 1 ORDER BY `short_links`.`id` LIMIT ?")).
			WithArgs("https://nonexistent.com", 1).
			WillReturnError(gorm.ErrRecordNotFound)

		sl, err := repo.GetShortLinkByURL(ctx, "https://nonexistent.com")
		assert.Error(t, err)
		assert.Nil(t, sl)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})
}

func TestMySQLRepository_CheckExistsByCode(t *testing.T) {
	db, mock := newTestDB(t)

	repo := &MySQLRepository{db: db}
	ctx := context.Background()

	t.Run("code exists", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(1)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM `short_links` WHERE short_code = ?")).
			WithArgs("ABCD").
			WillReturnRows(rows)

		exists, err := repo.CheckExistsByCode(ctx, "ABCD")
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("code does not exist", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM `short_links` WHERE short_code = ?")).
			WithArgs("NONEXIST").
			WillReturnRows(rows)

		exists, err := repo.CheckExistsByCode(ctx, "NONEXIST")
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestMySQLRepository_SaveAccessLog(t *testing.T) {
	db, mock := newTestDB(t)

	repo := &MySQLRepository{db: db}
	ctx := context.Background()

	t.Run("save access log successfully", func(t *testing.T) {
		log := &model.AccessLog{
			ShortCode: "ABCD",
			ClientIP:  "192.168.1.1",
			UserAgent:  "Mozilla/5.0",
		}

		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `access_logs`")).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := repo.SaveAccessLog(ctx, log)
		assert.NoError(t, err)
	})
}

func TestMySQLRepository_GetAccessLogs(t *testing.T) {
	db, mock := newTestDB(t)

	repo := &MySQLRepository{db: db}
	ctx := context.Background()

	t.Run("get access logs with limit", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "short_code", "client_ip", "user_agent", "referer", "source", "access_time"}).
			AddRow(1, "ABCD", "192.168.1.1", "Mozilla/5.0", "https://google.com", "google", now).
			AddRow(2, "ABCD", "192.168.1.2", "Safari", "https://baidu.com", "baidu", now.Add(-time.Hour))

		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `access_logs` WHERE short_code = ? ORDER BY access_time DESC LIMIT ?")).
			WithArgs("ABCD", 10).
			WillReturnRows(rows)

		logs, err := repo.GetAccessLogs(ctx, "ABCD", 10)
		assert.NoError(t, err)
		assert.Len(t, logs, 2)
		assert.Equal(t, "ABCD", logs[0].ShortCode)
	})

	t.Run("get access logs without limit", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "short_code", "client_ip", "user_agent", "referer", "source", "access_time"}).
			AddRow(1, "ABCD", "192.168.1.1", "Mozilla/5.0", "https://google.com", "google", now)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `access_logs` WHERE short_code = ? ORDER BY access_time DESC")).
			WithArgs("ABCD").
			WillReturnRows(rows)

		logs, err := repo.GetAccessLogs(ctx, "ABCD", 0)
		assert.NoError(t, err)
		assert.Len(t, logs, 1)
	})
}

func TestMySQLRepository_GetTotalLinksCount(t *testing.T) {
	db, mock := newTestDB(t)

	repo := &MySQLRepository{db: db}
	ctx := context.Background()

	t.Run("get total count", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(100)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM `short_links`")).
			WillReturnRows(rows)

		count, err := repo.GetTotalLinksCount(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(100), count)
	})

	t.Run("get zero count", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(0)

		mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM `short_links`")).
			WillReturnRows(rows)

		count, err := repo.GetTotalLinksCount(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestMySQLRepository_CleanupExpiredLinks(t *testing.T) {
	db, mock := newTestDB(t)

	repo := &MySQLRepository{db: db}
	ctx := context.Background()

	t.Run("cleanup expired links", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `short_links` WHERE expire_at IS NOT NULL AND expire_at < ?")).
			WillReturnResult(sqlmock.NewResult(0, 5))
		mock.ExpectCommit()

		count, err := repo.CleanupExpiredLinks(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})

	t.Run("cleanup with no expired links", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `short_links` WHERE expire_at IS NOT NULL AND expire_at < ?")).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		count, err := repo.CleanupExpiredLinks(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}

func TestMySQLRepository_GetDB(t *testing.T) {
	db, _ := newTestDB(t)

	repo := &MySQLRepository{db: db}
	assert.Equal(t, db, repo.GetDB())
}

func TestMySQLRepository_Close(t *testing.T) {
	db, mock := newTestDB(t)

	repo := &MySQLRepository{db: db}

	// Expect the Close call on the underlying connection
	mock.ExpectClose()

	err := repo.Close()
	assert.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
