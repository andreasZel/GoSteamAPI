package Models

import "gopkg.in/mgo.v2/bson"

type GameDeals struct {
	Id				bson.ObjectId		 	`json:"id" bson:"_id"`
	Game 			bson.ObjectId		 	`json:"game" bson:"game"`
	GameId			string					`json:"gameId" bson:"gameId"`
	Deals			[]map[string] string 	`json:"deals" bson:"deals"`
}