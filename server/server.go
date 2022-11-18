package server

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testproject/config"
	"testproject/types"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/inconshreveable/log15"
)

type Server struct {
	data     *types.OrderedMap[string, string]
	queue    *sqs.SQS
	queueUrl string
	waitTime int64
	logFile  os.File
	ctx      context.Context
	Cancel   context.CancelFunc
	logger   log15.Logger
	dataMux  sync.Mutex
	logsMux  sync.Mutex
}

func NewServer(conf *config.Config) (*Server, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(conf.Aws.Region),
		Credentials: credentials.NewStaticCredentials(conf.Aws.ClientId, conf.Aws.ClientSecret, conf.Aws.ClientToken),
	}))

	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, fmt.Errorf("Cannot assign session with credentials\n%s", err)
	}

	logFile, err := os.OpenFile(conf.LogFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	logger := log15.New("service", "server")
	logger.SetHandler(log15.LvlFilterHandler(log15.LvlDebug, log15.StdoutHandler))

	return &Server{
		data:     types.NewOrderedMap[string, string](),
		queue:    sqs.New(sess),
		logFile:  *logFile,
		ctx:      ctx,
		Cancel:   cancel,
		queueUrl: conf.Aws.QueueUrl,
		logger:   logger,
		dataMux:  sync.Mutex{},
		logsMux:  sync.Mutex{},
		waitTime: conf.ServerWaitTimeSeconds,
	}, nil
}

func (s *Server) StartServer() error {
	s.logger.Debug("Listening queue!")
	messagesChan := make(chan *sqs.Message)
	go s.listenMessages(messagesChan)
	err := s.processMessages(messagesChan)
	return err
}

func (s *Server) listenMessages(messagesChan chan *sqs.Message) error {
	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
			msgResult, err := s.queue.ReceiveMessage(&sqs.ReceiveMessageInput{
				AttributeNames: []*string{
					aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
				},
				MessageAttributeNames: []*string{
					aws.String(sqs.QueueAttributeNameAll),
				},
				QueueUrl:            &s.queueUrl,
				MaxNumberOfMessages: aws.Int64(10),
				WaitTimeSeconds:     aws.Int64(s.waitTime),
			})
			if err != nil {
				s.logger.Error("Error while receiving messages", "error", err.Error())
				return err
			}
			for _, message := range msgResult.Messages {
				messagesChan <- message
			}
		}
	}
}

func (s *Server) processMessages(messagesChan chan *sqs.Message) error {
	for {
		select {
		case <-s.ctx.Done():
			return nil
		case message := <-messagesChan:
			if message == nil {
				continue
			}
			go func(message string) {
				var item *types.Item
				err := json.Unmarshal([]byte(message), &item)
				if err != nil {
					s.logger.Error("Cannot unmarhsal message", "error", err.Error())
				}
				if item != nil {
					log := s.processItem(item)
					s.logsMux.Lock()
					defer s.logsMux.Unlock()
					// s.logger.Info(log)
					s.logFile.WriteString(fmt.Sprintf("%s || %s\n", time.Now().Format(time.RFC822), log))
				}
			}(*message.Body)
			_, err := s.queue.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      &s.queueUrl,
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				s.logger.Error("Error while deleting message", "error", err)
				return err
			}
		}
	}
}

func (s *Server) processItem(item *types.Item) (logMessage string) {
	s.dataMux.Lock()
	defer s.dataMux.Unlock()
	switch item.Action {
	case types.AddItem:
		ok := s.data.Set(item.Key, item.Value)
		return fmt.Sprintf("SetItem() done. Item(key: %s, value: %s) created: %t", item.Key, item.Value, ok)
	case types.GetItem:
		v, _ := s.data.Get(item.Key)
		return fmt.Sprintf("GetItem() done. Item(key: %s, value: %s)", item.Key, v)
	case types.GetAllItems:
		resp := ""
		for _, key := range s.data.Keys() {
			v, _ := s.data.Get(key)
			resp += fmt.Sprintf(" Item(key: %s, value: %s)", key, v)
		}
		return fmt.Sprintf("GetAllTimes() done.%s", resp)
	case types.RemoveItem:
		ok := s.data.Delete(item.Key)
		return fmt.Sprintf("DeleteItem() done. Item(key: %s) deleted: %t", item.Key, ok)
	default:
		return "Unknown action!"
	}
}
