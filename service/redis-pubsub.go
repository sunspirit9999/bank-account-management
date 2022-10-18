package service

import (
	"account-management/controller"
	"account-management/db"
	"account-management/model"
	re "account-management/redis"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"gorm.io/gorm"
)

var (
	// channel                = "mychannel3"
	minimumBalance float64 = 50000
)

type TaskQueue struct {
	UseWorker       bool
	NumOfWorkers    int
	MessageChannels []string
	accountService  *controller.AccountService
}

func NewTaskQueue(useWorker bool, numOfWorkers int, messageChannels []string) *TaskQueue {
	db := db.InitDB()
	accountModel := model.NewAccountModel(db)
	transactionModel := model.NewTransactionModel(db)
	redisClient := re.InitRedisClient()
	accountService := controller.NewAccountService(accountModel, transactionModel, redisClient, messageChannels)

	return &TaskQueue{
		UseWorker:       useWorker,
		NumOfWorkers:    numOfWorkers,
		MessageChannels: messageChannels,
		accountService:  accountService,
	}
}

func (t *TaskQueue) Start() {
	rdb := t.accountService.RedisClient
	accountModel := t.accountService.AccountModel
	transactionModel := t.accountService.TransactionModel

	subscriber := rdb.Subscribe(t.MessageChannels...)
	defer subscriber.Close()

	if err := subscriber.Ping(); err != nil {
		log.Fatalln("Redis server is busy !")
	}

	messageChannel := subscriber.Channel()

	if t.UseWorker {
		fmt.Printf("Started task queue with %d workers !\n", t.NumOfWorkers)
		for i := 1; i <= t.NumOfWorkers; i++ {
			go ProcessWithWorkers(rdb, messageChannel, accountModel, transactionModel, i)
		}
	} else {
		fmt.Printf("Started task queue without worker !\n")

		for message := range messageChannel {
			ProcessWithoutWorker(rdb, message, accountModel, transactionModel)
		}

		// for {
		// 	message, err := subscriber.ReceiveMessage()
		// 	ProcessWithoutWorker(rdb, message, accountModel, transactionModel)
		// 	if err != nil {
		// 		log.Fatal(err)
		// 	}
		// }

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

	err := accountModel.DB.Transaction(func(dbTx *gorm.DB) error {

		if tx.Type != "Deposit" {
			err := accountModel.SaveNewBalanceWithNegativeAmount(tx.Amount, tx.Sender, dbTx)
			if err != nil {
				return err
			}

			if tx.Type == "Transfer" {
				err = accountModel.SaveNewBalanceWithPositiveAmount(tx.Amount, tx.Receiver, dbTx)
				if err != nil {
					return err
				}
			}

		} else {
			err := accountModel.SaveNewBalanceWithPositiveAmount(tx.Amount, tx.Sender, dbTx)
			if err != nil {
				return err
			}
		}

		err := transactionModel.Save(tx)
		if err != nil {
			return err
		}

		return nil
	})

	return err
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
