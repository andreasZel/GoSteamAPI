package Models

import "gopkg.in/mgo.v2/bson"

type SteamGame struct {
	Id				bson.ObjectId		`json:"id" bson:"_id"`
	Name			string				`json:"name" bson:"name"`
	Steam_appid		string				`json:"steam_appid" bson:"steam_appid"`
	Header_image	string				`json:"header_image" bson:"header_image"`
	Capsule_image	string				`json:"capsule_image" bson:"capsule_image"`
	Developers		string				`json:"developers" bson:"developers"`
	Publishers		string				`json:"publishers" bson:"publishers"`
	Price			[]map[string] int 	`json:"price" bson:"price"`
	Platforms		[]bool				`json:"platforms" bson:"platforms"`
	Metacritic		[]string			`json:"metacritic" bson:"metacritic"`
	Genres			[]string			`json:"genres" bson:"genres"`
	Screenshots		[]string			`json:"screenshots" bson:"screenshots"`
	Background		string				`json:"background" bson:"background"`
}