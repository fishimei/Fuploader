package service

import (
	"Fuploader/internal/config"
	"Fuploader/internal/database"
	"Fuploader/internal/platform/baijiahao"
	"Fuploader/internal/platform/bilibili"
	"Fuploader/internal/platform/douyin"
	"Fuploader/internal/platform/kuaishou"
	"Fuploader/internal/platform/tencent"
	"Fuploader/internal/platform/tiktok"
	"Fuploader/internal/platform/xiaohongshu"
	"Fuploader/internal/types"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

type AccountService struct {
	db *gorm.DB
}

func NewAccountService(db *gorm.DB) *AccountService {
	return &AccountService{db: db}
}

func (s *AccountService) GetAccounts(ctx context.Context) ([]database.Account, error) {
	var accounts []database.Account
	result := s.db.Find(&accounts)
	if result.Error != nil {
		return nil, fmt.Errorf("query accounts failed: %w", result.Error)
	}
	return accounts, nil
}

func (s *AccountService) GetAccountByID(ctx context.Context, id int) (*database.Account, error) {
	var account database.Account
	result := s.db.First(&account, id)
	if result.Error != nil {
		return nil, fmt.Errorf("account not found: %w", result.Error)
	}
	return &account, nil
}

func (s *AccountService) AddAccount(ctx context.Context, platform string, name string) (*database.Account, error) {
	if err := os.MkdirAll(config.Config.CookiePath, 0755); err != nil {
		return nil, fmt.Errorf("create cookie directory failed: %w", err)
	}

	account := &database.Account{
		Platform: platform,
		Name:     name,
		Status:   config.AccountStatusInvalid,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(account).Error; err != nil {
			return fmt.Errorf("create account failed: %w", err)
		}

		account.CookiePath = filepath.Join(config.Config.CookiePath, fmt.Sprintf("%s_%d.json", platform, account.ID))

		if err := tx.Model(account).Update("cookie_path", account.CookiePath).Error; err != nil {
			return fmt.Errorf("update cookie path failed: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return account, nil
}

func (s *AccountService) DeleteAccount(ctx context.Context, id int) error {
	result := s.db.Delete(&database.Account{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete account failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

func (s *AccountService) UpdateAccount(ctx context.Context, account *database.Account) error {
	result := s.db.Save(account)
	if result.Error != nil {
		return fmt.Errorf("update account failed: %w", result.Error)
	}
	return nil
}

func (s *AccountService) ValidateAccount(ctx context.Context, id int) (bool, error) {
	var account database.Account
	result := s.db.First(&account, id)
	if result.Error != nil {
		return false, fmt.Errorf("account not found")
	}

	uploader := s.getUploader(account.Platform, uint(account.ID), account.CookiePath)
	valid, err := uploader.ValidateCookie(ctx)
	if err != nil {
		return false, err
	}

	if valid {
		account.Status = config.AccountStatusValid
	} else {
		account.Status = config.AccountStatusInvalid
	}
	s.db.Save(&account)

	return valid, nil
}

func (s *AccountService) LoginAccount(ctx context.Context, id int) error {
	var account database.Account
	result := s.db.First(&account, id)
	if result.Error != nil {
		return fmt.Errorf("account not found")
	}

	if account.CookiePath == "" {
		if err := os.MkdirAll(config.Config.CookiePath, 0755); err != nil {
			return fmt.Errorf("create cookie directory failed: %w", err)
		}
		account.CookiePath = s.GetCookiePath(account.Platform, uint(account.ID))
		if err := s.db.Model(&account).Update("cookie_path", account.CookiePath).Error; err != nil {
			return fmt.Errorf("update cookie path failed: %w", err)
		}
	}

	fmt.Printf("[DEBUG] LoginAccount - AccountID: %d, Platform: %s, CookiePath: %s\n",
		account.ID, account.Platform, account.CookiePath)

	uploader := s.getUploader(account.Platform, uint(account.ID), account.CookiePath)
	if err := uploader.Login(); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	account.Status = config.AccountStatusValid
	s.db.Save(&account)

	return nil
}

func (s *AccountService) ReloginAccount(ctx context.Context, id int) error {
	var account database.Account
	result := s.db.First(&account, id)
	if result.Error != nil {
		return fmt.Errorf("account not found")
	}

	if account.CookiePath == "" {
		if err := os.MkdirAll(config.Config.CookiePath, 0755); err != nil {
			return fmt.Errorf("create cookie directory failed: %w", err)
		}
		account.CookiePath = s.GetCookiePath(account.Platform, uint(account.ID))
		if err := s.db.Model(&account).Update("cookie_path", account.CookiePath).Error; err != nil {
			return fmt.Errorf("update cookie path failed: %w", err)
		}
	}

	if account.CookiePath != "" {
		if _, err := os.Stat(account.CookiePath); err == nil {
			if err := os.Remove(account.CookiePath); err != nil {
				return fmt.Errorf("remove old cookie file failed: %w", err)
			}
		}
	}

	uploader := s.getUploader(account.Platform, uint(account.ID), account.CookiePath)
	if err := uploader.Login(); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	account.Status = config.AccountStatusValid
	s.db.Save(&account)

	return nil
}

func (s *AccountService) getUploader(platform string, accountID uint, cookiePath string) types.Uploader {
	switch platform {
	case config.PlatformDouyin:
		return douyin.NewUploader(cookiePath)
	case config.PlatformTencent:
		return tencent.NewUploaderWithAccount(accountID)
	case config.PlatformKuaishou:
		return kuaishou.NewUploader(cookiePath)
	case config.PlatformTiktok:
		return tiktok.NewUploader(cookiePath)
	case config.PlatformXiaohongshu:
		return xiaohongshu.NewUploader(cookiePath)
	case config.PlatformBaijiahao:
		return baijiahao.NewUploader(cookiePath)
	case config.PlatformBilibili:
		return bilibili.NewUploader(cookiePath)
	default:
		return douyin.NewUploader(cookiePath)
	}
}

func (s *AccountService) GetCookiePath(platform string, accountID uint) string {
	return filepath.Join(config.Config.CookiePath, fmt.Sprintf("%s_%d.json", platform, accountID))
}
