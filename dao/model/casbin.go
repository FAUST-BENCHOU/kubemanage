package model

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	adapter "github.com/casbin/gorm-adapter/v3"
)

func init() {
	RegisterInitializer(CasbinInitOrder, &casbinDateBase{})
}

type casbinDateBase struct{}

func (c casbinDateBase) TableName() string {
	var entity adapter.CasbinRule
	return entity.TableName()
}

func (c casbinDateBase) MigrateTable(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).AutoMigrate(&adapter.CasbinRule{})
}

func (c casbinDateBase) InitData(ctx context.Context, db *gorm.DB) error {
	ok, err := c.IsInitData(ctx, db)
	if err != nil {
		return err
	}
	if !ok {
		// 首次初始化，插入所有 Casbin 规则
		return db.WithContext(ctx).Create(CasbinApi).Error
	}
	// 数据库已初始化，增量添加新的 Casbin 规则（仅管理员角色的新接口权限）
	// 检查 CasbinApi 中定义的规则是否都在数据库中，如果不在就插入
	for _, rule := range CasbinApi {
		// 只处理管理员角色的规则
		if rule.V0 == "111" {
			var existingRule adapter.CasbinRule
			result := db.WithContext(ctx).Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?",
				rule.Ptype, rule.V0, rule.V1, rule.V2).First(&existingRule)
			if result.Error != nil {
				// 规则不存在，插入新规则
				if err := db.WithContext(ctx).Create(&rule).Error; err != nil {
					// 忽略重复键错误（可能并发插入）
					continue
				}
			}
		}
	}
	return nil
}

func (c casbinDateBase) IsInitData(ctx context.Context, db *gorm.DB) (bool, error) {
	// TODO 管理员用户判断登陆接口是否具有权限
	if errors.Is(gorm.ErrRecordNotFound, db.WithContext(ctx).Where(adapter.CasbinRule{Ptype: "p", V0: "222", V1: "/api/user/login", V2: "POST"}).
		First(&adapter.CasbinRule{}).Error) { // 判断是否存在数据
		return false, nil
	}
	return true, nil
}

func (c casbinDateBase) TableCreated(ctx context.Context, db *gorm.DB) bool {
	return db.WithContext(ctx).Migrator().HasTable(&adapter.CasbinRule{})
}
