package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	initBalance    float64 = 50000
	minimumBalance float64 = 50000
)

type Account struct {
	AccountId   string `gorm:"primaryKey"`
	Username    string `gorm:"unique"`
	Password    string
	CreatedTime time.Time
	Balance     float64
	State       int32
	Token       string
}

type AccountModel struct {
	Account *Account
	DB      *gorm.DB
}

func NewAccountModel(db *gorm.DB) *AccountModel {
	return &AccountModel{
		Account: &Account{},
		DB:      db,
	}
}

func (a *AccountModel) Register(account *Account) error {
	if a.UserNameExist(account.Username) {
		return errors.New("username existed")
	}

	account.CreatedTime = time.Now()
	account.AccountId = uuid.NewString()
	account.Balance = initBalance

	err := a.DB.Create(account).Error
	if err != nil {
		return err
	}
	return nil
}

// func (a *AccountModel) CheckValidLogin(username string, password string) error {

// 	return nil
// }

func (a *AccountModel) UserNameExist(username string) bool {
	var usernameExist string
	err := a.DB.Raw("select username from accounts where username = ?", username).Scan(&usernameExist).Error
	if err != nil {
		return false
	}

	if usernameExist != "" {
		return true
	}
	return false
}

func (a *AccountModel) GetHashedPasswordByUsername(username string) string {
	var hashedPassword string
	err := a.DB.Raw("select password from accounts where username = ?", username).Scan(&hashedPassword).Error
	if err != nil {
		return ""
	}
	return hashedPassword
}

func (a *AccountModel) GetAccountIdByToken(token string) (string, error) {
	var accountId string
	err := a.DB.Raw("select account_id from accounts where token = ?", token).Scan(&accountId).Error
	if err != nil {
		return "", err
	}
	return accountId, nil
}

func (a *AccountModel) GetAccountIdByUserName(username string) (string, error) {
	var accountId string
	err := a.DB.Raw("select account_id from accounts where username = ?", username).Scan(&accountId).Error
	if err != nil {
		return "", err
	}
	return accountId, nil
}

func (a *AccountModel) GetList() ([]Account, error) {
	var accounts []Account
	err := a.DB.Table("accounts").Omit("password", "token").Find(&accounts).Error
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func (a *AccountModel) SaveToken(token, username string) error {
	err := a.DB.Exec("update accounts set token = ? where username = ?", token, username).Error
	if err != nil {
		return errors.New("failed to save account's token")
	}
	return nil
}

// func (a *AccountModel) UpdateBalance(tx *Transaction) error {

// }

func (a *AccountModel) GetAccountBalance(accountId string) (float64, error) {
	var balance float64
	err := a.DB.Raw("select balance from accounts where account_id = ?", accountId).Scan(&balance).Error
	if err != nil {
		return 0, errors.New("failed to get balance's account")
	}
	return balance, nil
}

func (a *AccountModel) SaveNewBalanceWithPositiveAmount(amount float64, accountId string, txs ...*gorm.DB) error {
	var tx = a.DB

	if len(txs) > 0 {
		tx = txs[0]
	}

	err := tx.Exec("update accounts set balance = accounts.balance + ? where account_id = ?", amount, accountId).Error
	if err != nil {
		return fmt.Errorf("failed to save new balance : %v", err)
	}

	return nil
}

func (a *AccountModel) SaveNewBalanceWithNegativeAmount(amount float64, accountId string, txs ...*gorm.DB) error {
	var tx = a.DB

	if len(txs) > 0 {
		tx = txs[0]
	}

	result := tx.Exec("update accounts set balance = accounts.balance - ? where account_id = ? and balance > ?", amount, accountId, minimumBalance)
	if err := result.Error; err != nil {
		return fmt.Errorf("failed to save new balance : %v", err)
	}

	rowsAffected := result.RowsAffected
	if rowsAffected == 0 {
		return fmt.Errorf("failed to save new balance : Balance must be greater than %0.f", minimumBalance)
	}

	return nil
}

func (a *AccountModel) SaveNewBalance(newAccountBalance float64, accountId string) error {
	err := a.DB.Exec("update accounts set balance = ? where account_id = ?", newAccountBalance, accountId).Error
	if err != nil {
		return fmt.Errorf("failed to set state's account : %v", err)
	}
	return nil
}

func (a *AccountModel) SetAccountState(accountId string, state int) error {
	err := a.DB.Exec("update accounts set state = ? where account_id = ?", state, accountId).Error
	if err != nil {
		return fmt.Errorf("failed to set state's account : %v", err)
	}
	return nil
}

func (a *AccountModel) GetAccountState(accountId string) (int, error) {
	var state int
	err := a.DB.Raw("select state from accounts where account_id = ?", accountId).Scan(&state).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get state's account : %v", err)
	}
	return state, err
}
