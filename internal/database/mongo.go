package database

import (
	"context"
	"crypto/tls"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return nil, errors.New("mongodb: empty connection URI")
	}

	// Atlas แนะนำ mongodb+srv + พารามิเตอร์พื้นฐาน
	if strings.HasPrefix(uri, "mongodb+srv://") && !strings.Contains(uri, "retryWrites=") {
		sep := "?"
		if strings.Contains(uri, "?") {
			sep = "&"
		}
		uri += sep + "retryWrites=true&w=majority"
	}

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().
		ApplyURI(uri).
		SetServerAPIOptions(serverAPI).
		SetConnectTimeout(30 * time.Second).
		SetServerSelectionTimeout(30 * time.Second).
		SetTLSConfig(&tls.Config{MinVersion: tls.VersionTLS12})

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	return client, nil
}

func DB(client *mongo.Client, name string) *mongo.Database {
	return client.Database(name)
}
