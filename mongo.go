package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"time"
)

type MongoRecorder struct {
	cfg             *config
	client          *mongo.Client
	sshLogCollect   *mongo.Collection
	authLogCollect  *mongo.Collection
	shellLogCollect *mongo.Collection
	stopWatchDog    chan bool
	isConnected     bool
}

func (mr *MongoRecorder) init() error {
	var err error
	mr.client, err = mongo.Connect(options.Client().ApplyURI(fmt.Sprintf(
		"mongodb://%s:%d",
		mr.cfg.MongoDBConfig.Host,
		mr.cfg.MongoDBConfig.Port,
	)).SetAuth(options.Credential{
		Username:   mr.cfg.MongoDBConfig.User,
		Password:   mr.cfg.MongoDBConfig.Password,
		AuthSource: mr.cfg.MongoDBConfig.Auth,
	}))
	if err != nil {
		return err
	}
	err = mr.client.Ping(context.Background(), nil)
	if err != nil {
		return err
	}
	infoLogger.Printf("Successfully connected to MongoDB")
	mr.sshLogCollect = mr.client.Database(mr.cfg.MongoDBConfig.DB).Collection(mr.cfg.MongoDBConfig.SSHLogCollect)
	mr.authLogCollect = mr.client.Database(mr.cfg.MongoDBConfig.DB).Collection(mr.cfg.MongoDBConfig.AuthLogCollect)
	mr.shellLogCollect = mr.client.Database(mr.cfg.MongoDBConfig.DB).Collection(mr.cfg.MongoDBConfig.ShellLogCollect)
	mr.isConnected = true
	return nil
}

func (mr *MongoRecorder) WatchDog() {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := mr.client.Ping(context.Background(), nil); err != nil {
				mr.isConnected = false
				warningLogger.Println("Connection lost, attempting to reconnect...")
				for {
					err := mr.init()
					if err != nil {
						warningLogger.Println("Reconnect failed:", err)
					} else {
						break
					}
				}
			}
		case <-mr.stopWatchDog:
			return
		}
	}
}

func (mr *MongoRecorder) Disconnect() {
	close(mr.stopWatchDog)
	_ = mr.client.Disconnect(context.Background())
}

func NewMongoRecorder(cfg *config) *MongoRecorder {
	mongoRecorder := &MongoRecorder{cfg: cfg}
	_ = mongoRecorder.init()
	go mongoRecorder.WatchDog()
	return mongoRecorder
}
