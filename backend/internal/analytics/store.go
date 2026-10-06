package analytics

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/bestows-Z/dev-hub/backend/internal/config"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type ViewEvent struct {
	ID   string    `json:"id" bson:"_id"`
	Kind string    `json:"kind" bson:"kind"`
	Slug string    `json:"slug" bson:"slug"`
	At   time.Time `json:"at" bson:"at"`
}

type DayCount struct {
	Day   string `json:"day" bson:"_id"`
	Count int64  `json:"count" bson:"count"`
}

type PageCount struct {
	Kind  string `json:"kind" bson:"kind"`
	Slug  string `json:"slug" bson:"slug"`
	Count int64  `json:"count" bson:"count"`
}

type Summary struct {
	Today     int64       `json:"today"`
	Last7Days int64       `json:"last_7_days"`
	Daily     []DayCount  `json:"daily"`
	TopPages  []PageCount `json:"top_pages"`
}

type Store struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewStore(ctx context.Context, cfg config.AnalyticsConfig) (*Store, error) {
	uri := &url.URL{Scheme: "mongodb", Host: cfg.MongoAddr, User: url.UserPassword(cfg.MongoUser, cfg.MongoPassword)}
	client, err := mongo.Connect(options.Client().ApplyURI(uri.String()))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	collection := client.Database("devhub").Collection("page_views")
	index := mongo.IndexModel{Keys: bson.D{{Key: "at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(180 * 24 * 60 * 60)}
	if _, err := collection.Indexes().CreateOne(ctx, index); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("create analytics retention index: %w", err)
	}
	return &Store{client: client, collection: collection}, nil
}

func (s *Store) Close(ctx context.Context) error { return s.client.Disconnect(ctx) }

func (s *Store) Insert(ctx context.Context, event ViewEvent) error {
	_, err := s.collection.InsertOne(ctx, event)
	if mongo.IsDuplicateKeyError(err) {
		return nil
	}
	return err
}

func (s *Store) Summary(ctx context.Context) (Summary, error) {
	zone := time.FixedZone("CST", 8*60*60)
	now := time.Now().In(zone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone)
	since := today.AddDate(0, 0, -6)
	filter := bson.D{{Key: "at", Value: bson.D{{Key: "$gte", Value: since.UTC()}}}}
	last7, err := s.collection.CountDocuments(ctx, filter)
	if err != nil {
		return Summary{}, err
	}
	todayCount, err := s.collection.CountDocuments(ctx, bson.D{{Key: "at", Value: bson.D{{Key: "$gte", Value: today.UTC()}}}})
	if err != nil {
		return Summary{}, err
	}
	dailyPipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{{Key: "format", Value: "%Y-%m-%d"}, {Key: "date", Value: "$at"}, {Key: "timezone", Value: "+08:00"}}}}}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
	cursor, err := s.collection.Aggregate(ctx, dailyPipeline)
	if err != nil {
		return Summary{}, err
	}
	var daily []DayCount
	if err := cursor.All(ctx, &daily); err != nil {
		return Summary{}, err
	}
	topPipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: bson.D{{Key: "kind", Value: "$kind"}, {Key: "slug", Value: "$slug"}}}, {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}}}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
		{{Key: "$limit", Value: 5}},
		{{Key: "$project", Value: bson.D{{Key: "kind", Value: "$_id.kind"}, {Key: "slug", Value: "$_id.slug"}, {Key: "count", Value: 1}, {Key: "_id", Value: 0}}}},
	}
	cursor, err = s.collection.Aggregate(ctx, topPipeline)
	if err != nil {
		return Summary{}, err
	}
	var top []PageCount
	if err := cursor.All(ctx, &top); err != nil {
		return Summary{}, err
	}
	if daily == nil {
		daily = []DayCount{}
	}
	if top == nil {
		top = []PageCount{}
	}
	return Summary{Today: todayCount, Last7Days: last7, Daily: daily, TopPages: top}, nil
}
