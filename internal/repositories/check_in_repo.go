package repositories

import (
	"ggcode/internal/models"
	"time"

	"gorm.io/gorm"
)

// 打卡相关
// 实现 struct 可为 checkInRepository

type CheckInRepository interface {
	CountCheckInToday(userID uint, today time.Time, count *int64) error
	CreateCheckInRecord(userID uint, today time.Time) error
	GetUserCheckInByDate(userID uint, date time.Time, checkIn *models.UserCheckIn) error
	GetLatestUserCheckIn(userID uint, checkIn *models.UserCheckIn) error
	CreateUserCheckIn(checkIn *models.UserCheckIn) error
	UpdateUserCheckIn(checkIn *models.UserCheckIn) error
	// 统计用户一年内每日打卡记录
	GetUserYearlyCheckInStats(userID uint, startDate, endDate time.Time) ([]models.CheckInStat, error)
	// 统计总打卡天数
	GetTotalCheckInDays(userID uint) (int64, error)
}

type checkInRepository struct {
	db *gorm.DB
}

func NewCheckInRepository(db *gorm.DB) CheckInRepository {
	return &checkInRepository{db: db}
}

var _ CheckInRepository = &checkInRepository{nil}

// CountCheckInToday 查询用户今日是否已打卡
func (r *checkInRepository) CountCheckInToday(userID uint, today time.Time, count *int64) error {
	return r.db.Model(&models.UserCheckIn{}).
		Where("user_id = ? AND DATE(check_date) = DATE(?)", userID, today).
		Count(count).Error
}

// CreateCheckInRecord 创建打卡记录
func (r *checkInRepository) CreateCheckInRecord(userID uint, today time.Time) error {
	record := models.UserCheckIn{
		UserID:    userID,
		CheckDate: today,
	}
	return r.db.Create(&record).Error
}

// GetUserCheckInByDate 查询用户某天的打卡记录
func (r *checkInRepository) GetUserCheckInByDate(userID uint, date time.Time, checkIn *models.UserCheckIn) error {
	return r.db.Where("user_id = ? AND check_date = ?", userID, date).First(checkIn).Error
}

// GetLatestUserCheckIn 查询用户最近的一条打卡记录
func (r *checkInRepository) GetLatestUserCheckIn(userID uint, checkIn *models.UserCheckIn) error {
	return r.db.Where("user_id = ?", userID).Order("check_date DESC").First(checkIn).Error
}

// CreateUserCheckIn 创建打卡记录
func (r *checkInRepository) CreateUserCheckIn(checkIn *models.UserCheckIn) error {
	return r.db.Create(checkIn).Error
}

// UpdateUserCheckIn 更新打卡记录
func (r *checkInRepository) UpdateUserCheckIn(checkIn *models.UserCheckIn) error {
	return r.db.Save(checkIn).Error
}

func (r *checkInRepository) GetUserYearlyCheckInStats(userID uint, startDate, endDate time.Time) ([]models.CheckInStat, error) {
	var checkInStats []models.CheckInStat
	err := r.db.Table("user_check_ins").
		Select("DATE(check_date) as date, study_count").
		Where("user_id = ? AND DATE(check_date) >= DATE(?) AND DATE(check_date) < DATE(?)", userID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Order("date").
		Scan(&checkInStats).Error
	return checkInStats, err
}

func (r *checkInRepository) GetTotalCheckInDays(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.UserCheckIn{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
