package Controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/andreasZel/GoSteamAPI/Models"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type GameController struct {
	session *mgo.Session
}

func NewGameController(session *mgo.Session) *GameController {
	return &GameController{session}
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

	//Get our SteamGame model
	steam_games := Models.SteamGame{} 

	//Find a file that has that bson object id and pass 
	//it to steam_games model
	if err := GC.session.DB("SteamPriceDB").C("SteamGames").FindId(oid).One(&steam_games); err != nil {
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
	params httprouter.Params) {

	//Get our SteamGame model
	steam_games := Models.SteamGame{} 

	steamId := params.ByName("steamId")

	//Get the values from postman 
	//? Later change to a call to steamAPI
	json.NewDecoder(request.Body).Decode(&steam_games)

	//?============================================\

	apiUrl := `"https://store.steampowered.com/api/appdetails/?appids=` + steamId + `"`

	// create new http request
	request2, err := http.NewRequest("GET", apiUrl, nil)
	request2.Header.Set("Content-Type", "application/json; charset=utf-8")

	// send the request
	client := &http.Client{}
	response, err := client.Do(request2)

	if err != nil {
		fmt.Println(err)
	}

	responseBody, err := io.ReadAll(response.Body)

	if err != nil {
		fmt.Println(err)
	}

	formattedData, err := json.Marshal(responseBody)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Status: ", response.Status)
	fmt.Println("Response body: ", formattedData)

	defer response.Body.Close()
	//?==============================================\

	//Create a bson object id, because we use mongo
	steam_games.Id = bson.NewObjectId()

	//Incert the new game to our collection
	GC.session.DB("SteamPriceDB").C("SteamGames").Insert(steam_games)
	
	steam_gamesjson, err := json.Marshal(steam_games)	

	if err != nil {
		fmt.Println(err)
	}

	//?==============================================\
	//Get our GameDeal model
	GameDeals := Models.GameDeals{} 

	//Create a bson object id, because we use mongo
	GameDeals.Id = bson.NewObjectId()
	//Connect the Gamedeal to the specific game
	GameDeals.Game = steam_games.Id

	GC.session.DB("SteamPriceDB").C("GameDeals").Insert(GameDeals)

	gameDealsjson, err := json.Marshal(GameDeals)	

	if err != nil {
		fmt.Println(err)
	}
	//?==============================================\

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	fmt.Println(writer, "%s\n%s\n", steam_gamesjson, gameDealsjson)
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

	//Incert the new game to our collection
	GC.session.DB("SteamPriceDB").C("SteamGames").Update(filter, update)

	steam_gamesjson, err := json.Marshal(steam_games)	

	if err != nil {
		fmt.Println(err)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	fmt.Println(writer, "%s\n", steam_gamesjson)
}

//[DELETE] DeleteGame(id)
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

	if err := GC.session.DB("SteamPriceDB").C("SteamGames").Remove(oid); err != nil {
		writer.WriteHeader(http.StatusNotFound)
	}

	writer.WriteHeader(http.StatusOK)
	fmt.Fprintln(writer, "Delete Game\n", oid, "\n")
}