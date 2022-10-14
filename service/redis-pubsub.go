package service

import (
	"account-management/db"
	"account-management/model"
	re "account-management/redis"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

var (
	// channel                = "mychannel3"
	channelList            = []string{"deposit", "withdraw", "transfer"}
	numOfWorkers           = 10
	minimumBalance float64 = 50000
)

func StartTaskQueue(useWorker bool) {
	db := db.InitDB()
	accountModel := model.NewAccountModel(db)
	transactionModel := model.NewTransactionModel(db)
	rdb := re.InitRedisClient()

	subscriber := rdb.Subscribe(channelList...)
	defer subscriber.Close()

	if err := subscriber.Ping(); err != nil {
		log.Fatalln("Redis server is busy !")
	}

	messageChannel := subscriber.Channel()

	if useWorker {
		for i := 1; i <= numOfWorkers; i++ {
			go ProcessWithWorkers(rdb, messageChannel, accountModel, transactionModel, i)
		}
	} else {
		for message := range messageChannel {
			ProcessWithoutWorker(rdb, message, accountModel, transactionModel)
		}
	}

	time.Sleep(time.Hour * 10000)

}

func ProcessWithWorkers(rdb *redis.Client, messageChan <-chan *redis.Message, accountModel *model.AccountModel,
	transactionModel *model.TransactionModel, workerId int) {

	for message := range messageChan {
		payload := message.Payload

		var tx model.Transaction

		err := json.Unmarshal([]byte(payload), &tx)
		if err != nil {
			fmt.Println(err)
			continue
		}

		err = ProcessTransactionWithWorkers(accountModel, transactionModel, &tx)
		if err != nil {
			fmt.Println(err)
			continue
		}

		switch tx.Type {
		case "Transfer":
			fmt.Printf("Worker %d : %s transfered %0.f$ to %s\n", workerId, tx.Sender, tx.Amount, tx.Receiver)
		case "Deposit":
			fmt.Printf("Worker %d : %s deposited %0.f$ to account\n", workerId, tx.Sender, tx.Amount)
		case "Withdraw":
			fmt.Printf("Worker %d : %s withdrew %0.f$ from account\n", workerId, tx.Sender, tx.Amount)
		}

	}

}

func ProcessWithoutWorker(rdb *redis.Client, message *redis.Message, accountModel *model.AccountModel, transactionModel *model.TransactionModel) error {

	payload := message.Payload

	var tx model.Transaction

	err := json.Unmarshal([]byte(payload), &tx)
	if err != nil {
		return err
	}

	err = ProcessTransactionWithoutWorker(accountModel, transactionModel, &tx)
	if err != nil {
		return err
	}

	switch tx.Type {
	case "Transfer":
		fmt.Printf("%s transfered %0.f$ to %s\n", tx.Sender, tx.Amount, tx.Receiver)
	case "Deposit":
		fmt.Printf("%s deposited %0.f$ to account\n", tx.Sender, tx.Amount)
	case "Withdraw":
		fmt.Printf("%s withdrew %0.f$ from account\n", tx.Sender, tx.Amount)
	}

	return nil

}

func ProcessTransactionWithWorkers(accountModel *model.AccountModel, transactionModel *model.TransactionModel, tx *model.Transaction) error {

	// var senderBalance float64
	var err error

	if tx.Type != "Deposit" {

		err = accountModel.SaveNewBalanceWithNegativeAmount(tx.Amount, tx.Sender)
		if err != nil {
			return err
		}

		if tx.Type == "Transfer" {
			err = accountModel.SaveNewBalanceWithPositiveAmount(tx.Amount, tx.Receiver)
			if err != nil {
				return err
			}
		}

	} else {
		err = accountModel.SaveNewBalanceWithPositiveAmount(tx.Amount, tx.Sender)
		if err != nil {
			return err
		}
	}

	err = transactionModel.Save(tx)
	if err != nil {
		return err
	}

	return nil
}

func ProcessTransactionWithoutWorker(accountModel *model.AccountModel, transactionModel *model.TransactionModel, tx *model.Transaction) error {

	var senderBalance float64
	var err error

	senderBalance, err = accountModel.GetAccountBalance(tx.Sender)
	if err != nil {
		return err
	}

	if tx.Type != "Deposit" {

		if senderBalance-tx.Amount < minimumBalance {
			return fmt.Errorf("your balance is not enough to %s", strings.ToLower(tx.Type))
		}

		newSenderBalance := senderBalance - tx.Amount

		err = accountModel.SaveNewBalance(newSenderBalance, tx.Sender)
		if err != nil {
			return err
		}

		if tx.Type == "Transfer" {
			receiverBalance, err := accountModel.GetAccountBalance(tx.Receiver)
			if err != nil {
				return err
			}

			newReceiverBalance := receiverBalance + tx.Amount
			err = accountModel.SaveNewBalance(newReceiverBalance, tx.Receiver)
			if err != nil {
				return err
			}
		}

	} else {
		newSenderBalance := senderBalance + tx.Amount

		err = accountModel.SaveNewBalance(newSenderBalance, tx.Sender)
		if err != nil {
			return err
		}
	}

	err = transactionModel.Save(tx)
	if err != nil {
		return err
	}

	return nil
}
