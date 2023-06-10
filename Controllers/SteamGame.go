package Controllers

import (
	//"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/tidwall/gjson"

	"go.mongodb.org/mongo-driver/mongo"
	//"go.mongodb.org/mongo-driver/mongo/options"
	//"go.mongodb.org/mongo-driver/mongo/readpref"
	"context"

	"github.com/andreasZel/GoSteamAPI/Models"
	"github.com/julienschmidt/httprouter"

	//"gopkg.in/mgo.v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/mgo.v2/bson"
)

type GameController struct {
	client *mongo.Client
}

type ResponseGameId struct {
	GameId	string	`json:"GameId"`
}

func NewGameController(client *mongo.Client) *GameController {
	return &GameController{client}
}

//[GET] Get all Steam Games that are saved
func (GC GameController) AllGames (
	writer http.ResponseWriter, 
	request *http.Request, 
	_ httprouter.Params){

	steam_games := Models.SteamGame{} 

	ctx := context.Background()

	cursor, err := GC.client.Database("SteamPriceDB").Collection("SteamGames").Find(ctx, bson.D{}) 

	if err != nil{
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	cursor.Decode(&steam_games)

	steam_gamesjson, err := json.Marshal(steam_games)	

	if err != nil {
		fmt.Println(err)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	fmt.Println(writer, "%s\n", steam_gamesjson)
}

// [GET] GetsteamGame (id)
func (GC GameController) GetSteamGame (
	writer http.ResponseWriter, 
	request *http.Request, 
	params httprouter.Params) {

	//Get id from parameters of get request 
	id := params.ByName("id")

	if !bson.IsObjectIdHex(id) {
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	//Transform it to bson object id, because we use mongo
	oid := bson.ObjectIdHex(id)
	filter := bson.M{"_id": bson.M{"$eq": oid}}

	//Get our SteamGame model
	steam_games := Models.SteamGame{} 

	//Find a file that has that bson object id and pass 
	//it to steam_games model
	ctx := context.Background()

	if err := GC.client.Database("SteamPriceDB").Collection("SteamGames").FindOne(ctx, filter).Decode(&steam_games); err != nil {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	
	//Transform the results to json
	steam_Gamejson, err := json.Marshal(steam_games)
	
	if err != nil {
		fmt.Println(err)
	}

	//Display Ok if everything worked out
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	fmt.Println(writer, "%s\n", steam_Gamejson)
}


// [POST] CreateGame
func (GC GameController) CreateGame (
	writer http.ResponseWriter, 
	request *http.Request, 
	_ httprouter.Params) {

	//Get the values from postman 
	steamGameId := ResponseGameId{} 

	err := json.NewDecoder(request.Body).Decode(&steamGameId)
	
	if err != nil {
		fmt.Println(err)
	}

	if steamGameId.GameId == "" {
		fmt.Println("Provide a GameId in Body, example { GameId : 1203220 }")
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	//Send GET Request to steam API  
	apiUrl := `https://store.steampowered.com/api/appdetails/?appids=` + steamGameId.GameId + ``
	response, err := http.Get(apiUrl)
   	
	if err != nil {
      	fmt.Println("err")
   	}
	
	//Close Body on return of function
	defer response.Body.Close()
	
	//Read the response body
	body, err := io.ReadAll(response.Body)

	if err != nil {
		fmt.Println(err)
   	}

	//Get only specific fields using gjson package
	value := gjson.GetBytes(body, `` + steamGameId.GameId + `.success`)

	if value.String() == "false" {
		fmt.Println("Game does not exist in steam API")
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	//Get our SteamGame model
	steam_games := Models.SteamGame{} 

	steam_games.Name = gjson.GetBytes(body, `` + steamGameId.GameId + `.data.name`).String()
	steam_games.Steam_appid = gjson.GetBytes(body, `` + steamGameId.GameId + `.data.steam_appid`).String()
	steam_games.Header_image = gjson.GetBytes(body, `` + steamGameId.GameId + `.data.header_image`).String()
	steam_games.Capsule_image = gjson.GetBytes(body, `` + steamGameId.GameId + `.data.capsule_image`).String()
	json.Unmarshal([]byte(gjson.GetBytes(body, `` + steamGameId.GameId + `.data.developers`).String()), &steam_games.Developers)
	json.Unmarshal([]byte(gjson.GetBytes(body, `` + steamGameId.GameId + `.data.publishers`).String()), &steam_games.Publishers)
	
	if gjson.GetBytes(body, `` + steamGameId.GameId + `.data.is_free`).String() == "false" {	
		steam_games.Price = gjson.GetBytes(body, `` + steamGameId.GameId + `.data.price_overview.final_formatted`).String()
	} else {
		steam_games.Price = "free"
	}
	
	steam_games.Platforms = append(steam_games.Platforms, gjson.GetBytes(body, `` + steamGameId.GameId + `.data.platforms.windows`).String())
	steam_games.Platforms = append(steam_games.Platforms, gjson.GetBytes(body, `` + steamGameId.GameId + `.data.platforms.mac`).String())
	
	if gjson.GetBytes(body, `` + steamGameId.GameId + `.data.metacritic.score`).String() != "" {
		steam_games.Metacritic = append(steam_games.Metacritic, gjson.GetBytes(body, `` + steamGameId.GameId + `.data.metacritic.score`).String())
		steam_games.Metacritic = append(steam_games.Metacritic, gjson.GetBytes(body, `` + steamGameId.GameId + `.data.metacritic.url`).String())
	} else {
		steam_games.Metacritic = append(steam_games.Metacritic, "false")
	}

	//Screenshot and Genres is an array of objects
	//we Unmarshal it to an array of maps 
	//and get only the "path_thumbnail" value
	var ScreenshotDat []map[string]string
	var GenreDat []map[string]string

	//Screenshot Information
	json.Unmarshal([]byte(gjson.GetBytes(body, `` + steamGameId.GameId + `.data.screenshots`).String()), &ScreenshotDat)
		
	for idx := range ScreenshotDat {
		steam_games.Screenshots = append(steam_games.Screenshots, ScreenshotDat[idx]["path_thumbnail"])
    }

	//Genre Information
	json.Unmarshal([]byte(gjson.GetBytes(body, `` + steamGameId.GameId + `.data.genres`).String()), &GenreDat)
		
	for idx := range GenreDat {
		steam_games.Genres = append(steam_games.Genres, GenreDat[idx]["description"])
    }

	steam_games.Background = gjson.GetBytes(body, `` + steamGameId.GameId + `.data.background`).String()

	//Create a bson object id, because we use mongo
	steam_games.Id = primitive.NewObjectID()

	ctx := context.Background()

	//Incert the new game to our collection
	result, err := GC.client.Database("SteamPriceDB").Collection("SteamGames").InsertOne(ctx, steam_games)
	
	if err != nil {
		writer.WriteHeader(http.StatusNotModified)
		fmt.Println(err)
		return 
	}

	fmt.Println(result)

	//Get our GameDeal model
	GameDeals := Models.GameDeals{} 

	//Create a bson object id, because we use mongo
	GameDeals.Id = primitive.NewObjectID()

	//Connect the Gamedeal to the specific game
	GameDeals.Game = steam_games.Id
	GameDeals.GameId = steam_games.Steam_appid

	//Send GET Request to cheapshark to get CheapSharkId
	apiUrl2 := `https://www.cheapshark.com/api/1.0/games?steamAppID=` + steamGameId.GameId + ``
	response3, err := http.Get(apiUrl2)
   	
	if err != nil {
      	fmt.Println("err")
   	}
	
	//Close Body on return of function
	defer response3.Body.Close()

	//Read the response body
	body3, err := io.ReadAll(response3.Body)

	if err != nil {
		fmt.Println(err)
   	}

	var CheapSharkDat []map[string]string

	json.Unmarshal(body3, &CheapSharkDat)
	//fmt.Println(CheapSharkDat)

	GameDeals.CheapSharkId = CheapSharkDat[0]["gameID"]
    //GameDeals.Cheapest = CheapSharkDat[0]["cheapest"]

	//Send GET Request to cheapshark to get the Deals for the game
	apiUrl3 := `https://www.cheapshark.com/api/1.0/games?id=` + GameDeals.CheapSharkId + ``
	response4, err := http.Get(apiUrl3)
   	
	if err != nil {
      	fmt.Println("err")
   	}
	
	//Close Body on return of function
	defer response4.Body.Close()

	//Read the response body
	body4, err := io.ReadAll(response4.Body)

	if err != nil {
		fmt.Println(err)
   	}

	//Get only specific fields using gjson package
	GameDeals.Cheapest = append(GameDeals.Cheapest, gjson.GetBytes(body4, `cheapestPriceEver.price`).String())
	GameDeals.Cheapest = append(GameDeals.Cheapest, gjson.GetBytes(body4, `cheapestPriceEver.date`).String())

	//var DealsDat []map[string]string

	json.Unmarshal([]byte(gjson.GetBytes(body4, `deals`).String()), &GameDeals.Deals)
	
	currentTime := time.Now()

	for idx := range GameDeals.Deals {
		GameDeals.Deals[idx].Date = strconv.FormatInt(currentTime.Unix(), 10)
    }

	result2, err := GC.client.Database("SteamPriceDB").Collection("GameDeals").InsertOne(ctx, GameDeals)

	if err != nil {
		writer.WriteHeader(http.StatusNotModified)
		fmt.Println(err)
		return 
	}

	fmt.Println(result2)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
}

// [POST] UpdateGame (id)
func (GC GameController) UpdateGame (
	writer http.ResponseWriter, 
	request *http.Request, 
	params httprouter.Params) {

	//Get id from parameters of get request 
	id := params.ByName("id")

	if !bson.IsObjectIdHex(id) {
		writer.WriteHeader(http.StatusNotFound)
		return 
	}

	//Transform it to bson object id, because we use mongo
	oid := bson.ObjectIdHex(id)

	// Declare an _id filter to get a specific MongoDB document
	filter := bson.M{"_id": bson.M{"$eq": oid}}
	
	//Get our SteamGame model
	steam_games := Models.SteamGame{} 

	//Get the values from postman 
	//? Later change to a call to steamAPI
	json.NewDecoder(request.Body).Decode(&steam_games)

	//Declare a filter that will change field values 
	//according to SteamGame struct
	update := steam_games

	ctx := context.Background()

	//Incert the new deals for our collection
	result3, err := GC.client.Database("SteamPriceDB").Collection("SteamGames").UpdateOne(ctx, filter, update)

	if err != nil {
		writer.WriteHeader(http.StatusNotModified)
		fmt.Println(err)
		return //? ======> DEBUG <======
	}

	fmt.Println(result3)

	steam_gamesjson, err := json.Marshal(steam_games)	

	if err != nil {
		fmt.Println(err)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	fmt.Println(writer, "%s\n", steam_gamesjson)
}

//[DELETE] DeleteGame (id)
func (GC GameController) DeleteGame (
	writer http.ResponseWriter, 
	request *http.Request, 
	params httprouter.Params) {

	//Get id from parameters of get request 
	id := params.ByName("id")

	if !bson.IsObjectIdHex(id) {
		writer.WriteHeader(http.StatusNotFound)
		return
	}
	
	oid := bson.ObjectIdHex(id)
	filter := bson.M{"_id": bson.M{"$eq": oid}}

	if result, err := GC.client.Database("SteamPriceDB").Collection("SteamGames").DeleteOne(context.TODO(), filter); err != nil {
		fmt.Fprintln(writer, "Error Deleting Game\n", result, "\n")
		writer.WriteHeader(http.StatusNotFound)
	}

	writer.WriteHeader(http.StatusOK)
	fmt.Fprintln(writer, "Delete Game\n", oid, "\n")
}