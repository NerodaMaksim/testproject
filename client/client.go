package client

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testproject/config"
	"testproject/types"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
)

type Client struct {
	queue    *sqs.SQS
	queueUrl string
}

func NewClient(conf *config.Config) (*Client, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:      aws.String(conf.Aws.Region),
		Credentials: credentials.NewStaticCredentials(conf.Aws.ClientId, conf.Aws.ClientSecret, conf.Aws.ClientToken),
	}))

	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, fmt.Errorf("Cannot assign session with credentials\n%s", err)
	}

	return &Client{
		queue:    sqs.New(sess),
		queueUrl: conf.Aws.QueueUrl,
	}, nil
}

func (c *Client) SendMessage(item *types.Item) {
	id, _ := uuid.NewRandom()
	req, _ := json.Marshal(item)
	c.queue.SendMessage(&sqs.SendMessageInput{
		DelaySeconds:           aws.Int64(0),
		MessageBody:            aws.String(string(req)),
		QueueUrl:               &c.queueUrl,
		MessageGroupId:         aws.String("test"),
		MessageDeduplicationId: aws.String(id.String()),
	})
}

func (c *Client) AddItem(key, value string) {
	item := &types.Item{Action: "AddItem", Key: key, Value: value}
	go c.SendMessage(item)
}

func (c *Client) GetItem(key string) {
	item := &types.Item{Action: "GetItem", Key: key}
	go c.SendMessage(item)
}

func (c *Client) GetAllItems() {
	item := &types.Item{Action: "GetAllItems"}
	go c.SendMessage(item)
}

func (c *Client) RemoveItem(key string) {
	item := &types.Item{Action: "RemoveItem", Key: key}
	go c.SendMessage(item)
}

type ClientsManager struct {
	clients   map[string]*ClientUsage
	input     *os.File
	clientCfg *config.Config
	mux       sync.Mutex
	ctx       context.Context
	Cancel    context.CancelFunc
}

type ClientUsage struct {
	client   *Client
	lastUsed time.Time
}

func NewClientsManager(cfg *config.Config) (manager *ClientsManager, err error) {
	input := os.Stdin
	if len(cfg.ClientsInputPath) != 0 {
		input, err = os.Open(cfg.ClientsInputPath)
		if err != nil {
			return nil, err
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &ClientsManager{
		clients:   make(map[string]*ClientUsage),
		mux:       sync.Mutex{},
		input:     input,
		clientCfg: cfg,
		ctx:       ctx,
		Cancel:    cancel,
	}, nil
}

func (cm *ClientsManager) ListenClientActions() error {
	if cm.input == os.Stdin {
		fmt.Println("Write clients tasks here in format <clientId> <item>")
	}

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			<-ticker.C
			cm.removeUnusedClients()
		}
	}()

	lines, errChan := SubscribeToFileInput(cm.input)

	for {
		select {
		case <-cm.ctx.Done():
			return nil
		case line := <-lines:
			if len(line) != 0 {
				go cm.processClientAction(line)
			}
		case err := <-errChan:
			if err != nil {
				return err
			}
		}
	}
}

func (cm *ClientsManager) removeUnusedClients() {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	for clientId, clientUsage := range cm.clients {
		if time.Since(clientUsage.lastUsed) > time.Second*10 {
			delete(cm.clients, clientId)
		}
	}

}

func (cm *ClientsManager) processClientAction(inputStr string) error {
	cm.mux.Lock()
	defer cm.mux.Unlock()
	if len(inputStr) <= 1 {
		return fmt.Errorf("Wrong input string. Should be in format <clientId> <item>")
	}

	splittedInput := strings.Split(inputStr, " ")

	clientId := splittedInput[0]
	itemStr := strings.Join(splittedInput[1:], " ")

	var item *types.Item
	err := json.Unmarshal([]byte(itemStr), &item)
	if err != nil {
		return err
	}
	if client, ok := cm.clients[clientId]; ok {
		go client.client.SendMessage(item)
		client.lastUsed = time.Now()
		return nil
	}
	client, err := NewClient(cm.clientCfg)
	if err != nil {
		return err
	}
	cm.clients[clientId] = &ClientUsage{client, time.Now()}
	go client.SendMessage(item)
	return nil
}
