package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"airway-reservation/internal/pkg/constant"
	"airway-reservation/internal/pkg/database/interfaces"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type TransactionDB struct {
	db *interfaces.DB
}

func (t *TransactionDB) TransactionScope(ctx context.Context, fc func(context.Context, *interfaces.DB) error) (err error) {
	panicked := true
	if committer, ok := t.db.Gorm.Statement.ConnPool.(gorm.TxCommitter); ok && committer != nil {
		// nested transaction
		if !t.db.Gorm.DisableNestedTransaction {
			err := t.db.Gorm.SavePoint(fmt.Sprintf("sp%p", fc)).Error
			defer func() {
				// Make sure to rollback when panic, Block error or Commit error
				if panicked || err != nil {
					t.db.Gorm.RollbackTo(fmt.Sprintf("sp%p", fc))
				}
			}()
		}

		if err == nil {
			err = fc(ctx, t.db)
		}
	} else {
		tx := t.db.Gorm.Begin()

		defer func() {
			// Make sure to rollback when panic, Block error or Commit error
			if panicked || err != nil {
				tx.Rollback()
			}
		}()
		if err = tx.Error; err == nil {
			err = fc(ctx, &interfaces.DB{Gorm: tx})
		}

		if err == nil {
			err = tx.Commit().Error
		}
	}

	panicked = false
	return
}

func (t *TransactionDB) GetDB() *interfaces.DB {
	return t.db
}
func NewTransactionDBIF(g *gorm.DB) interfaces.TransactionDBIF {
	db := interfaces.DB{Gorm: g}
	return &TransactionDB{db: &db}
}

func SetDB(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, constant.TransactionDBContextKey, db)
}

func NewDB() *gorm.DB {

	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbName := os.Getenv("POSTGRES_DB")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("データベースとの接続に成功しました")
	return db
}

func CloseDB(db *gorm.DB) {
	sqlDB, _ := db.DB()
	if err := sqlDB.Close(); err != nil {
		log.Fatalln(err)
	}
	fmt.Println("データベースとの接続を切断しました")
}
