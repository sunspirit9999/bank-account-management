package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Transaction struct {
	TransactionId string `gorm:"primaryKey"`
	Sender        string
	Receiver      string
	Amount        float64
	CreatedTime   time.Time
	Type          string
}

type TransactionModel struct {
	Transaction *Transaction
	DB          *gorm.DB
}

func NewTransactionModel(db *gorm.DB) *TransactionModel {
	return &TransactionModel{
		Transaction: &Transaction{},
		DB:          db,
	}
}

func (t *TransactionModel) Save(tx *Transaction) error {
	tx.CreatedTime = time.Now()
	err := t.DB.Create(tx).Error
	if err != nil {
		return fmt.Errorf("failed to save transaction : %v", err)
	}
	return nil
}
