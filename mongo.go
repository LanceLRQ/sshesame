package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
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

func LogEventToMongo(mongoRecorder *MongoRecorder, eventType string, logRecord *bson.M, entry logEntry) {
	var err error
	collect := mongoRecorder.sshLogCollect
	switch eventType {
	case "no_auth":
		if entry.(noAuthLog).User != "" {
			mergeBSONM(*logRecord, bson.M{
				"user":     entry.(noAuthLog).User,
				"accepted": entry.(noAuthLog).Accepted,
			})
			collect = mongoRecorder.authLogCollect
			break
		} else {
			return
		}
	case "password_auth":
		mergeBSONM(*logRecord, bson.M{
			"password": entry.(passwordAuthLog).Password,
			"user":     entry.(passwordAuthLog).User,
			"accepted": entry.(passwordAuthLog).Accepted,
		})
		collect = mongoRecorder.authLogCollect
		break
	case "public_key_auth":
		mergeBSONM(*logRecord, bson.M{
			"public_key": entry.(publicKeyAuthLog).PublicKeyFingerprint,
			"user":       entry.(publicKeyAuthLog).User,
			"accepted":   entry.(publicKeyAuthLog).Accepted,
		})
		collect = mongoRecorder.authLogCollect
		break
	case "keyboard_interactive_auth":
		logRecord = mergeBSONM(*logRecord, bson.M{
			"answers":  entry.(keyboardInteractiveAuthLog).Answers,
			"user":     entry.(keyboardInteractiveAuthLog).User,
			"accepted": entry.(keyboardInteractiveAuthLog).Accepted,
		})

		collect = mongoRecorder.authLogCollect
		break
	case "session_input":
		logRecord = mergeBSONM(*logRecord, bson.M{
			"content":    entry.(sessionInputLog).Input,
			"channel_id": entry.(sessionInputLog).ChannelID,
		})
		collect = mongoRecorder.shellLogCollect
		break
	default:
		logRecord = mergeBSONM(*logRecord, bson.M{
			"payload": entry,
		})
		break
	}
	_, err = collect.InsertOne(context.Background(), logRecord)
	if err != nil {
		warningLogger.Printf("[mongo] Failed to insert log event: %v", err)
	}
}
