package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testproject/config"
	"testproject/types"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/inconshreveable/log15"
)

type Server struct {
	data     *types.OrderedMap
	queue    *sqs.SQS
	queueUrl string
	waitTime int64
	logFile  log15.Logger
	ctx      context.Context
	Cancel   context.CancelFunc
	logger   log15.Logger
	dataMux  sync.RWMutex
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

	logger := log15.New("service", "server")
	logger.SetHandler(log15.LvlFilterHandler(log15.LvlDebug, log15.StdoutHandler))

	logFile := log15.New()
	logfileHandler, err := log15.FileHandler(conf.LogFilePath, log15.LogfmtFormat())
	if err != nil {
		return nil, err
	}

	logFile.SetHandler(logfileHandler)

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		data:     types.NewOrderedMap(),
		queue:    sqs.New(sess),
		ctx:      ctx,
		Cancel:   cancel,
		queueUrl: conf.Aws.QueueUrl,
		logger:   logger,
		dataMux:  sync.RWMutex{},
		logsMux:  sync.Mutex{},
		waitTime: conf.ServerWaitTimeSeconds,
		logFile:  logFile,
	}, nil
}

func (s *Server) StartServer() error {
	s.logger.Debug("Listening queue!")
	messagesChan := make(chan *sqs.Message)
	go s.listenMessages(messagesChan)
	err := s.processMessages(messagesChan)
	return err
}

func (s *Server) listenMessages(messagesChan chan *sqs.Message) {
	receivedMessages := make(chan *sqs.ReceiveMessageOutput)
	errChan := s.receiveMessages(receivedMessages)
	for {
		select {
		case <-s.ctx.Done():
			return
		case msgResult := <-receivedMessages:
			if msgResult != nil {
				for _, message := range msgResult.Messages {
					messagesChan <- message
				}
			}
		case err := <-errChan:
			s.logger.Error("Error while receiving messages", "error", err.Error())
		}
	}
}

func (s *Server) receiveMessages(messages chan *sqs.ReceiveMessageOutput) chan error {
	errChan := make(chan error)

	go func() {
		for {
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
				errChan <- err
				continue
			}
			messages <- msgResult
		}
	}()

	return errChan
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
			go func(messageData string) {
				var item *types.Item
				err := json.Unmarshal([]byte(messageData), &item)
				if err != nil {
					s.logger.Error("Cannot unmarhsal message", "error", err.Error())
				}
				if item != nil {
					log := s.processItem(item)
					s.logsMux.Lock()
					defer s.logsMux.Unlock()
					s.logFile.Info(log)
				}
				_, err = s.queue.DeleteMessage(&sqs.DeleteMessageInput{
					QueueUrl:      &s.queueUrl,
					ReceiptHandle: message.ReceiptHandle,
				})
				if err != nil {
					s.logger.Error("Error while deleting message", "error", err)
				}
			}(*message.Body)
		}
	}
}

func (s *Server) processItem(item *types.Item) (logMessage string) {
	switch item.Action {
	case types.AddItem:
		s.dataMux.Lock()
		defer s.dataMux.Unlock()
		ok := s.data.Set(item.Key, item.Value)
		return fmt.Sprintf("SetItem() done. Item(key: %s, value: %s) created: %t", item.Key, item.Value, ok)
	case types.GetItem:
		s.dataMux.RLock()
		defer s.dataMux.RUnlock()
		v, _ := s.data.Get(item.Key)
		return fmt.Sprintf("GetItem() done. Item(key: %s, value: %s)", item.Key, v)
	case types.GetAllItems:
		s.dataMux.RLock()
		defer s.dataMux.RUnlock()
		resp := ""
		for _, key := range s.data.Keys() {
			v, _ := s.data.Get(key)
			resp += fmt.Sprintf(" Item(key: %s, value: %s)", key, v)
		}
		return fmt.Sprintf("GetAllTimes() done.%s", resp)
	case types.RemoveItem:
		s.dataMux.Lock()
		defer s.dataMux.Unlock()
		ok := s.data.Delete(item.Key)
		return fmt.Sprintf("DeleteItem() done. Item(key: %s) deleted: %t", item.Key, ok)
	default:
		return "Unknown action!"
	}
}
