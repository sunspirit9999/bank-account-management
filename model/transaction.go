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

func (t *TransactionModel) GetTransactionState(transactionId string) bool {
	var transactionExist string
	err := t.DB.Raw("select transaction_id from transactions where transaction_id = ?", transactionId).Scan(&transactionExist).Error
	if err != nil {
		return false
	}

	if transactionExist != "" {
		return true
	}
	return false
}
