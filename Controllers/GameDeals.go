package Controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/andreasZel/GoSteamAPI/Models"
	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/mgo.v2/bson"
)

type GameDealId struct {
	Id string `json:"GameId"`
}

type GameDealsController struct {
	client *mongo.Client
}

func NewGameDealsController(client *mongo.Client) *GameDealsController {
	return &GameDealsController{client}
}

// [GET] GetGameDeals 
func (GDC GameDealsController) GetGameDeals(
	writer http.ResponseWriter,
	request *http.Request,
	params httprouter.Params) {
	
	//Get the steamGameId from response body
	steamGameId :=  GameDealId{}

	err := json.NewDecoder(request.Body).Decode(&steamGameId)

	fmt.Println(steamGameId.Id)	

	if err != nil {
		fmt.Println(err)
	}

	if steamGameId.Id == "" {
		fmt.Println("Provide a GameId in Body, example { GameId : 1203220 }")
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		return
	}
	
	//Make the input Id to Object Id for mongo
	DealId, err := primitive.ObjectIDFromHex(steamGameId.Id)

	if err != nil {
		fmt.Println(err)
	}

	//Create filter to find if the steamappId exist in db
	filter := bson.M{"game": DealId}

	//Get our GameDeals model
	steam_game_deals := Models.GameDeals{}

	ctx := context.Background()

	if err := GDC.client.Database("SteamPriceDB").Collection("GameDeals").FindOne(ctx, filter).Decode(&steam_game_deals); err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	//Transform the results to json
	steam_GameDealsjson, err := json.Marshal(steam_game_deals)

	if err != nil {
		fmt.Println(err)
	}

	//Display Ok if everything worked out
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	fmt.Fprintf(writer, "%s\n", steam_GameDealsjson)
}