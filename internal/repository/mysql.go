package repository

import (
	"context"
	"time"

	"octopus/internal/config"
	"octopus/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// MySQLRepository handles MySQL operations
type MySQLRepository struct {
	db *gorm.DB
}

// NewMySQLRepository creates a new MySQL repository
func NewMySQLRepository(cfg *config.MySQLConfig) *MySQLRepository {
	// Configure GORM logger
	var gormLogger logger.Interface
	if zerolog.GlobalLevel() > zerolog.DebugLevel {
		gormLogger = logger.Default.LogMode(logger.Silent)
	} else {
		gormLogger = logger.Default.LogMode(logger.Info)
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MySQL")
	}

	// Auto migrate tables
	if err := db.AutoMigrate(&model.ShortLink{}, &model.AccessLog{}); err != nil {
		log.Fatal().Err(err).Msg("Failed to migrate database")
	}

	log.Info().Msg("MySQL connected successfully")

	return &MySQLRepository{db: db}
}

// GetDB returns the GORM DB instance
func (r *MySQLRepository) GetDB() *gorm.DB {
	return r.db
}

// SaveShortLink saves a short link to MySQL
func (r *MySQLRepository) SaveShortLink(ctx context.Context, sl *model.ShortLink) error {
	return r.db.WithContext(ctx).Create(sl).Error
}

// GetShortLinkByCode retrieves a short link by short code
func (r *MySQLRepository) GetShortLinkByCode(ctx context.Context, shortCode string) (*model.ShortLink, error) {
	var sl model.ShortLink
	err := r.db.WithContext(ctx).
		Where("short_code = ? AND status = 1", shortCode).
		First(&sl).Error
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// GetShortLinkByURL retrieves a short link by original URL (for deduplication)
func (r *MySQLRepository) GetShortLinkByURL(ctx context.Context, url string) (*model.ShortLink, error) {
	var sl model.ShortLink
	err := r.db.WithContext(ctx).
		Where("original_url = ? AND status = 1", url).
		First(&sl).Error
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// CheckExistsByCode checks if a short code exists
func (r *MySQLRepository) CheckExistsByCode(ctx context.Context, shortCode string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ShortLink{}).
		Where("short_code = ?", shortCode).
		Count(&count).Error
	return count > 0, err
}

// SaveAccessLog saves an access log to MySQL
func (r *MySQLRepository) SaveAccessLog(ctx context.Context, accessLog *model.AccessLog) error {
	return r.db.WithContext(ctx).Create(accessLog).Error
}

// GetAccessLogs retrieves access logs for a short code
func (r *MySQLRepository) GetAccessLogs(ctx context.Context, shortCode string, limit int) ([]model.AccessLog, error) {
	var logs []model.AccessLog
	query := r.db.WithContext(ctx).
		Where("short_code = ?", shortCode).
		Order("access_time DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// GetTotalLinksCount returns the total count of short links
func (r *MySQLRepository) GetTotalLinksCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.ShortLink{}).Count(&count).Error
	return count, err
}

// CleanupExpiredLinks removes expired short links
func (r *MySQLRepository) CleanupExpiredLinks(ctx context.Context) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Where("expire_at IS NOT NULL AND expire_at < ?", now).
		Delete(&model.ShortLink{})
	return result.RowsAffected, result.Error
}

// Close closes the database connection
func (r *MySQLRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
