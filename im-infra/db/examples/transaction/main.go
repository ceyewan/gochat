package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/db"
	"gorm.io/gorm"
)

// Account 账户模型
type Account struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null;index"`
	AccountNo string `gorm:"uniqueIndex;size:50;not null"`
	Balance   int64  `gorm:"not null;default:0;comment:余额(分)"`
	Status    int    `gorm:"default:1;comment:账户状态 1=active 0=frozen"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Transaction 交易记录模型
type TransactionLog struct {
	ID        uint   `gorm:"primaryKey"`
	TxnID     string `gorm:"uniqueIndex;size:50;not null"`
	FromID    uint   `gorm:"index;comment:转出账户ID"`
	ToID      uint   `gorm:"index;comment:转入账户ID"`
	Amount    int64  `gorm:"not null;comment:交易金额(分)"`
	Type      string `gorm:"size:20;not null;comment:交易类型"`
	Status    string `gorm:"size:20;not null;comment:交易状态"`
	CreatedAt time.Time
}

func main() {
	ctx := context.Background()

	// 创建自定义日志器
	logger := clog.Namespace("db-transaction-example")

	// 创建 MySQL 配置
	cfg := db.MySQLConfig("gochat:gochat_pass_2024@tcp(localhost:3306)/gochat_dev?charset=utf8mb4&parseTime=True&loc=Local")

	// 使用 New 函数创建数据库实例，并注入 Logger
	provider, err := db.New(ctx, cfg, db.WithLogger(logger))
	if err != nil {
		log.Fatalf("创建数据库实例失败: %v", err)
	}
	defer provider.Close()

	logger.Info("开始事务操作示例")

	// 自动迁移
	if err := provider.AutoMigrate(ctx, &Account{}, &TransactionLog{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	gormDB := provider.DB(ctx)

	// === 初始化测试数据 ===
	logger.Info("初始化测试账户")
	accounts := []Account{
		{UserID: 1, AccountNo: "ACC001", Balance: 100000}, // 1000.00 元
		{UserID: 2, AccountNo: "ACC002", Balance: 50000},  // 500.00 元
		{UserID: 3, AccountNo: "ACC003", Balance: 0},      // 0.00 元
	}

	for _, account := range accounts {
		if err := gormDB.WithContext(ctx).Create(&account).Error; err != nil {
			logger.Error("创建账户失败", clog.Err(err))
		} else {
			logger.Info("创建账户成功",
				clog.String("accountNo", account.AccountNo),
				clog.Int64("balance", account.Balance))
		}
	}

	// === 示例1: 基础事务操作 ===
	logger.Info("=== 示例1: 基础事务操作 ===")
	err = provider.Transaction(ctx, func(tx *gorm.DB) error {
		// 在事务中创建多个相关记录
		user4Account := Account{
			UserID:    4,
			AccountNo: "ACC004",
			Balance:   80000,
		}

		if err := tx.Create(&user4Account).Error; err != nil {
			return err
		}

		// 创建初始交易记录
		txnLog := TransactionLog{
			TxnID:  "TXN001",
			ToID:   user4Account.ID,
			Amount: 80000,
			Type:   "deposit",
			Status: "completed",
		}

		if err := tx.Create(&txnLog).Error; err != nil {
			return err
		}

		logger.Info("事务中成功创建账户和交易记录")
		return nil
	})

	if err != nil {
		logger.Error("基础事务失败", clog.Err(err))
	} else {
		logger.Info("基础事务操作成功")
	}

	// === 示例2: 转账事务（成功场景）===
	logger.Info("=== 示例2: 转账事务（成功场景）===")
	transferAmount := int64(10000) // 100.00 元
	fromAccountNo := "ACC001"
	toAccountNo := "ACC002"

	err = performTransfer(provider, logger, fromAccountNo, toAccountNo, transferAmount)
	if err != nil {
		logger.Error("转账失败", clog.Err(err))
	} else {
		logger.Info("转账成功")
	}

	// === 示例3: 转账事务（余额不足，回滚场景）===
	logger.Info("=== 示例3: 转账事务（余额不足，回滚场景）===")
	transferAmount = int64(200000) // 2000.00 元（余额不足）
	fromAccountNo = "ACC002"
	toAccountNo = "ACC003"

	logger.Info("开始执行转账事务（预期失败场景）")
	err = performTransfer(provider, logger, fromAccountNo, toAccountNo, transferAmount)
	if err != nil {
		logger.Error("转账失败（预期失败）", clog.Err(err))
		logger.Info("确认事务已回滚，检查账户余额是否恢复")
	} else {
		logger.Info("转账成功（不应该到这里）")
	}

	// === 示例4: 嵌套事务 ===
	logger.Info("=== 示例4: 嵌套事务 ===")
	err = provider.Transaction(ctx, func(tx *gorm.DB) error {
		// 外层事务：批量创建账户
		newAccounts := []Account{
			{UserID: 5, AccountNo: "ACC005", Balance: 30000},
			{UserID: 6, AccountNo: "ACC006", Balance: 40000},
		}

		for _, account := range newAccounts {
			if err := tx.Create(&account).Error; err != nil {
				return err
			}
		}

		// 内层事务：为新账户创建交易记录
		return tx.Transaction(func(tx2 *gorm.DB) error {
			for i, account := range newAccounts {
				txnLog := TransactionLog{
					TxnID:  fmt.Sprintf("TXN_BATCH_%d", i+1),
					ToID:   account.ID,
					Amount: account.Balance,
					Type:   "initial_deposit",
					Status: "completed",
				}
				if err := tx2.Create(&txnLog).Error; err != nil {
					return err
				}
			}
			logger.Info("嵌套事务中创建交易记录成功")
			return nil
		})
	})

	if err != nil {
		logger.Error("嵌套事务失败", clog.Err(err))
	} else {
		logger.Info("嵌套事务操作成功")
	}

	// === 示例5: 事务中的错误处理和重试 ===
	logger.Info("=== 示例5: 事务中的错误处理和重试 ===")
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = provider.Transaction(ctx, func(tx *gorm.DB) error {
			// 模拟可能失败的操作
			if rand.Float32() < 0.7 && attempt < 3 { // 70%概率失败，但第3次必成功
				return errors.New("模拟网络错误")
			}

			// 成功的操作
			account := Account{
				UserID:    7,
				AccountNo: fmt.Sprintf("ACC_RETRY_%d", attempt),
				Balance:   25000,
			}
			return tx.Create(&account).Error
		})

		if err != nil {
			logger.Warn("事务尝试失败，准备重试",
				clog.Int("attempt", attempt),
				clog.Err(err))
			time.Sleep(100 * time.Millisecond) // 等待重试
		} else {
			logger.Info("事务重试成功", clog.Int("attempt", attempt))
			break
		}
	}

	// === 查看最终结果 ===
	logger.Info("=== 查看最终结果 ===")
	var finalAccounts []Account
	if err := gormDB.WithContext(ctx).Find(&finalAccounts).Error; err != nil {
		logger.Error("查询最终账户失败", clog.Err(err))
	} else {
		logger.Info("最终账户列表", clog.Int("count", len(finalAccounts)))
		for _, account := range finalAccounts {
			logger.Info("账户信息",
				clog.String("accountNo", account.AccountNo),
				clog.Int64("balance", account.Balance),
				clog.Int("status", account.Status))
		}
	}

	var finalTxns []TransactionLog
	if err := gormDB.WithContext(ctx).Find(&finalTxns).Error; err != nil {
		logger.Error("查询最终交易记录失败", clog.Err(err))
	} else {
		logger.Info("最终交易记录", clog.Int("count", len(finalTxns)))
		for _, txn := range finalTxns {
			logger.Info("交易记录",
				clog.String("txnID", txn.TxnID),
				clog.String("type", txn.Type),
				clog.Int64("amount", txn.Amount),
				clog.String("status", txn.Status))
		}
	}

	logger.Info("事务操作示例完成")
}

// performTransfer 执行转账操作
func performTransfer(provider db.Provider, logger clog.Logger, fromAccountNo, toAccountNo string, amount int64) error {
	return provider.Transaction(context.Background(), func(tx *gorm.DB) error {
		// 1. 查询转出账户（加锁）
		var fromAccount Account
		if err := tx.Where("account_no = ?", fromAccountNo).First(&fromAccount).Error; err != nil {
			return fmt.Errorf("转出账户不存在: %w", err)
		}

		// 2. 检查余额
		logger.Info("检查转出账户余额",
			clog.String("accountNo", fromAccountNo),
			clog.Int64("currentBalance", fromAccount.Balance),
			clog.Int64("transferAmount", amount))
		if fromAccount.Balance < amount {
			logger.Warn("余额不足，事务将回滚",
				clog.String("accountNo", fromAccountNo),
				clog.Int64("currentBalance", fromAccount.Balance),
				clog.Int64("transferAmount", amount))
			return fmt.Errorf("余额不足，当前余额: %d, 转账金额: %d", fromAccount.Balance, amount)
		}

		// 3. 查询转入账户
		var toAccount Account
		if err := tx.Where("account_no = ?", toAccountNo).First(&toAccount).Error; err != nil {
			return fmt.Errorf("转入账户不存在: %w", err)
		}

		// 4. 扣减转出账户余额
		if err := tx.Model(&fromAccount).Update("balance", fromAccount.Balance-amount).Error; err != nil {
			return fmt.Errorf("扣减转出账户余额失败: %w", err)
		}

		// 5. 增加转入账户余额
		if err := tx.Model(&toAccount).Update("balance", toAccount.Balance+amount).Error; err != nil {
			return fmt.Errorf("增加转入账户余额失败: %w", err)
		}

		// 6. 创建交易记录
		txnLog := TransactionLog{
			TxnID:  fmt.Sprintf("TXN_%d", time.Now().UnixNano()),
			FromID: fromAccount.ID,
			ToID:   toAccount.ID,
			Amount: amount,
			Type:   "transfer",
			Status: "completed",
		}

		if err := tx.Create(&txnLog).Error; err != nil {
			return fmt.Errorf("创建交易记录失败: %w", err)
		}

		logger.Info("转账操作成功",
			clog.String("from", fromAccountNo),
			clog.String("to", toAccountNo),
			clog.Int64("amount", amount),
			clog.String("txnID", txnLog.TxnID))

		return nil
	})
}
