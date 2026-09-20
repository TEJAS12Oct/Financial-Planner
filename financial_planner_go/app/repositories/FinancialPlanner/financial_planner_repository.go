package FinancialPlanner

import (
	dao "FinancialPlannerGo/app/domain/dao/financialPlanner"
	"fmt"

	"gorm.io/gorm"
)

type FinancialPlannerRepository interface {
	Save(report *dao.FinancialPlannerReport) error
	Update(report *dao.FinancialPlannerReport) error
	SaveSelectedGoals(goals []dao.FinancialPlannerSelectedGoal) error
	GetReports(userID uint) ([]dao.FinancialPlannerReport, error)
	GetByID(reportID uint) (*dao.FinancialPlannerReport, error)
}

type financialPlannerRepository struct {
	db *gorm.DB
}

func NewFinancialPlannerRepository(db *gorm.DB) FinancialPlannerRepository {
	return &financialPlannerRepository{db: db}
}

func (r *financialPlannerRepository) Save(report *dao.FinancialPlannerReport) error {
	return r.db.Create(report).Error
}

func (r *financialPlannerRepository) Update(report *dao.FinancialPlannerReport) error {
	return r.db.Save(report).Error
}

func (r *financialPlannerRepository) SaveSelectedGoals(
	goals []dao.FinancialPlannerSelectedGoal,
) error {
	if len(goals) == 0 {
		return nil
	}

	result := r.db.Create(
		&goals,
	)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != int64(len(goals)) {
		return fmt.Errorf(
			"expected to insert %d selected goals, but inserted %d",
			len(goals),
			result.RowsAffected,
		)
	}

	return nil
}

func (r *financialPlannerRepository) GetReports(userID uint) ([]dao.FinancialPlannerReport, error) {
	var reports []dao.FinancialPlannerReport

	query := r.db.
		Model(&dao.FinancialPlannerReport{}).
		Order("created_at DESC")

	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Find(&reports).Error; err != nil {
		return nil, err
	}

	return reports, nil
}

func (r *financialPlannerRepository) GetByID(reportID uint) (*dao.FinancialPlannerReport, error) {
	var report dao.FinancialPlannerReport
	if err := r.db.First(&report, reportID).Error; err != nil {
		return nil, err
	}
	return &report, nil
}
