package main

import (
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

func isValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func isInUsers(token string) (bool, error) {
	rCount, err := loginData.CountDocuments(ctx, bson.M{"_id": token})
	return rCount != 0, err
}
