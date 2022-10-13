package controller

import (
	"account-management/model"
	"account-management/utils.go"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

const (
	minimumBalance  = 50000
	depositChannel  = "deposit"
	withdrawChannel = "withdraw"
	transferChannel = "transfer"
)

type AccountService struct {
	AccountModel     *model.AccountModel
	TransactionModel *model.TransactionModel
	RedisClient      *redis.Client
}

func NewAccountService(accountModel *model.AccountModel, transactionModel *model.TransactionModel, rdb *redis.Client) *AccountService {
	return &AccountService{
		AccountModel:     accountModel,
		TransactionModel: transactionModel,
		RedisClient:      rdb,
	}
}

func (a *AccountService) Register(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	var account model.Account
	err = json.Unmarshal(body, &account)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	if account.Username == "" || account.Password == "" {
		c.JSON(500, gin.H{
			"messages": errors.New("username or password must not be blank").Error(),
			"status":   500,
		})
		return
	}

	account.Password, err = utils.HashPassword(account.Password)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	err = a.AccountModel.Register(&account)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	c.JSON(200, gin.H{
		"messages": "Created successfully !",
		"status":   200,
	})

}

func (a *AccountService) Login(c *gin.Context) {
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	var account model.Account
	err = json.Unmarshal(body, &account)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	username := account.Username
	password := account.Password

	if username == "" || password == "" {
		c.JSON(500, gin.H{
			"messages": errors.New("username or password must not be blank").Error(),
			"status":   500,
		})
		return
	}

	if !a.AccountModel.UserNameExist(account.Username) {
		c.JSON(500, gin.H{
			"messages": errors.New("username doesn't exist").Error(),
			"status":   500,
		})
		return
	}

	hashedPassword := a.AccountModel.GetHashedPasswordByUsername(account.Username)

	if !utils.CheckPasswordHash(password, hashedPassword) {
		c.JSON(500, gin.H{
			"messages": errors.New("password is wrong").Error(),
			"status":   500,
		})
		return
	}

	token, err := utils.GenerateToken(username)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": errors.New("fail to generate jwt-token").Error(),
			"status":   500,
		})
		return
	}

	err = a.AccountModel.SaveToken(token, username)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	c.JSON(200, gin.H{
		"messages": "Logged in !",
		"status":   200,
		"token":    token,
	})

}

func (a *AccountService) GetAllAccounts(c *gin.Context) {

	accounts, err := a.AccountModel.GetList()
	if err != nil {
		c.JSON(500, gin.H{
			"messages": "Failed to get all accounts from DB",
			"status":   500,
		})
		return
	}

	if len(accounts) == 0 {
		c.JSON(500, gin.H{
			"messages": "There no accounts found in DB",
			"status":   500,
		})
		return
	}

	c.JSON(200, gin.H{
		"messages": accounts,
		"status":   200,
	})

}

func (a *AccountService) Deposit(c *gin.Context) {

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	var transaction model.Transaction
	err = json.Unmarshal(body, &transaction)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	err = a.checkValidTransaction(&transaction)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	bearerToken := c.Request.Header.Get("Authorization")

	var token string
	if len(strings.Split(bearerToken, " ")) == 2 {
		token = strings.Split(bearerToken, " ")[1]
	}

	accountId, err := a.AccountModel.GetAccountIdByToken(token)
	if accountId == "" || err != nil {
		c.JSON(500, gin.H{
			"messages": errors.New("failed to get accountId with given token").Error(),
			"status":   500,
		})
		return
	}

	transaction.Sender = accountId
	transaction.Type = "Deposit"
	transaction.TransactionId = uuid.NewString()

	payload, err := json.Marshal(&transaction)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": errors.New("failed to marshal request").Error(),
			"status":   500,
		})
		return
	}

	err = a.RedisClient.Publish(depositChannel, payload).Err()
	if err != nil {
		c.JSON(500, gin.H{
			"messages": errors.New("failed to send request to task queue").Error(),
			"status":   500,
		})
		return
	}

	// err = a.ProcessTransaction(&transaction)
	// if err != nil {
	// 	c.JSON(500, gin.H{
	// 		"messages": err.Error(),
	// 		"status":   500,
	// 	})
	// 	return
	// }

	c.JSON(200, gin.H{
		"messages":       "your deposit request is processing !",
		"transaction_id": transaction.TransactionId,
		"status":         200,
	})

}

func (a *AccountService) Withdraw(c *gin.Context) {

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	var transaction model.Transaction
	err = json.Unmarshal(body, &transaction)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	err = a.checkValidTransaction(&transaction)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	bearerToken := c.Request.Header.Get("Authorization")

	var token string
	if len(strings.Split(bearerToken, " ")) == 2 {
		token = strings.Split(bearerToken, " ")[1]
	}

	accountId, err := a.AccountModel.GetAccountIdByToken(token)
	if accountId == "" || err != nil {
		c.JSON(500, gin.H{
			"messages": errors.New("failed to get accountId with given token").Error(),
			"status":   500,
		})
		return
	}

	transaction.Sender = accountId
	transaction.Type = "Withdraw"
	transaction.TransactionId = uuid.NewString()

	payload, err := json.Marshal(&transaction)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": errors.New("failed to marshal request").Error(),
			"status":   500,
		})
		return
	}

	err = a.RedisClient.Publish(withdrawChannel, payload).Err()
	if err != nil {
		c.JSON(500, gin.H{
			"messages": errors.New("failed to send request to task queue").Error(),
			"status":   500,
		})
		return
	}

	// err = a.ProcessTransaction(&transaction)
	// if err != nil {
	// 	c.JSON(500, gin.H{
	// 		"messages": err.Error(),
	// 		"status":   500,
	// 	})
	// 	return
	// }

	c.JSON(200, gin.H{
		"messages":       "your withdraw request is processing !",
		"transaction_id": transaction.TransactionId,
		"status":         200,
	})

}

func (a *AccountService) Transfer(c *gin.Context) {

	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	var transaction model.Transaction
	err = json.Unmarshal(body, &transaction)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	err = a.checkValidTransaction(&transaction)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": err.Error(),
			"status":   500,
		})
		return
	}

	bearerToken := c.Request.Header.Get("Authorization")

	var token string
	if len(strings.Split(bearerToken, " ")) == 2 {
		token = strings.Split(bearerToken, " ")[1]
	}

	accountId, err := a.AccountModel.GetAccountIdByToken(token)
	if accountId == "" || err != nil {
		c.JSON(500, gin.H{
			"messages": errors.New("failed to get accountId with given token").Error(),
			"status":   500,
		})
		return
	}

	transaction.Sender = accountId
	transaction.Type = "Transfer"
	transaction.TransactionId = uuid.NewString()

	payload, err := json.Marshal(&transaction)
	if err != nil {
		c.JSON(500, gin.H{
			"messages": errors.New("failed to marshal request").Error(),
			"status":   500,
		})
		return
	}

	err = a.RedisClient.Publish(transferChannel, payload).Err()
	if err != nil {
		c.JSON(500, gin.H{
			"messages": errors.New("failed to send request to task queue").Error(),
			"status":   500,
		})
		return
	}

	// err = a.ProcessTransaction(&transaction)
	// if err != nil {
	// 	c.JSON(500, gin.H{
	// 		"messages": err.Error(),
	// 		"status":   500,
	// 	})
	// 	return
	// }

	c.JSON(200, gin.H{
		"messages":       "your transfer request is processing !",
		"transaction_id": transaction.TransactionId,
		"status":         200,
	})

}

func (a *AccountService) checkValidTransaction(tx *model.Transaction) error {

	if tx.Type == "Transfer" {
		if tx.Receiver == "" {
			return errors.New("receiver_id must not be blank")
		}
	}

	if tx.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}

	return nil
}

func (a *AccountService) ProcessTransaction(tx *model.Transaction) error {

	senderAmount, err := a.AccountModel.GetAccountBalance(tx.Sender)
	if err != nil {
		return err
	}

	var newSenderBalance float64
	var newReceiverBalance float64

	if tx.Type == "Deposit" {
		newSenderBalance = senderAmount + tx.Amount
	} else {
		if senderAmount-tx.Amount < minimumBalance {
			return fmt.Errorf("your balance is not enough to %s", strings.ToLower(tx.Type))
		}

		newSenderBalance = senderAmount - tx.Amount
	}

	if tx.Type == "Transfer" {
		receiverAmount, err := a.AccountModel.GetAccountBalance(tx.Receiver)
		if err != nil {
			return err
		}

		newReceiverBalance = receiverAmount + tx.Amount
	}

	err = a.AccountModel.SaveNewBalance(newSenderBalance, tx.Sender)
	if err != nil {
		return err
	}

	if tx.Type == "Transfer" {
		err = a.AccountModel.SaveNewBalance(newReceiverBalance, tx.Receiver)
		if err != nil {
			return err
		}
	}

	err = a.TransactionModel.Save(tx)
	if err != nil {
		return err
	}

	return nil
}