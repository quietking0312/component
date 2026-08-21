package mmongo

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestClientOptions_URI(t *testing.T) {
	cfg := defaultOptions()
	SetURI("mongodb://localhost:27017/?replicaSet=rs0")(cfg)
	SetAuth("user", "pass")(cfg)

	clientOpts := cfg.clientOptions()
	if clientOpts.Auth != nil {
		t.Fatalf("expected auth to be ignored when URI is set, got %+v", clientOpts.Auth)
	}
}

func TestClientOptions_Hosts(t *testing.T) {
	cfg := defaultOptions()
	SetHosts([]string{"127.0.0.1:27017", "127.0.0.1:27018"})(cfg)
	SetAuth("user", "pass")(cfg)
	SetAuthSource("test")(cfg)
	SetReplicaSet("rs0")(cfg)
	SetMaxPoolSize(50)(cfg)
	SetMinPoolSize(2)(cfg)

	clientOpts := cfg.clientOptions()

	if got := clientOpts.Hosts; len(got) != 2 || got[0] != "127.0.0.1:27017" {
		t.Fatalf("unexpected hosts: %+v", got)
	}
	if clientOpts.Auth == nil || clientOpts.Auth.Username != "user" || clientOpts.Auth.AuthSource != "test" {
		t.Fatalf("unexpected auth: %+v", clientOpts.Auth)
	}
	if clientOpts.ReplicaSet == nil || *clientOpts.ReplicaSet != "rs0" {
		t.Fatalf("unexpected replica set: %+v", clientOpts.ReplicaSet)
	}
	if clientOpts.MaxPoolSize == nil || *clientOpts.MaxPoolSize != 50 {
		t.Fatalf("unexpected max pool size: %+v", clientOpts.MaxPoolSize)
	}
}

func TestNewClient(t *testing.T) {
	cli, err := NewClient(
		SetHosts([]string{"127.0.0.1:27017"}),
		SetDatabase("test"),
		SetConnectTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close(context.Background())

	coll := cli.Database().Collection("mmongo_test")
	if _, err := coll.InsertOne(context.Background(), bson.M{"k": "v"}); err != nil {
		t.Fatal(err)
	}
}
