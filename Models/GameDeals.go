package Models

import (
	//"gopkg.in/mgo.v2/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)
type GameDeals struct {
	Id				primitive.ObjectID		`json:"id" bson:"_id"`
	Game 			primitive.ObjectID		`json:"game" bson:"game"`
	GameId			string					`json:"gameId" bson:"gameId"`
	CheapSharkId	string					`json:"cheapSharkId" bson:"cheapSharkId"`
	Cheapest		[]string				`json:"cheapest" bson:"cheapest"`
	Deals			[]struct {
						StoreId			string  `json:"storeId" bson:"storeId"`
						RetailPrice 	string	`json:"retailPrice" bson:"retailPrice"`
						Date			string	`json:"date" bson:"date"`
					} 	`json:"deals" bson:"deals"`
}