package Controllers

import (
	//"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
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
	GameId string `json:"GameId"`
}

func NewGameController(client *mongo.Client) *GameController {
	return &GameController{client}
}

// [GET] Get all Steam Games that are saved
func (GC GameController) AllGames(
	writer http.ResponseWriter,
	request *http.Request,
	_ httprouter.Params) {

	steam_games := Models.SteamGame{}

	ctx := context.Background()

	cursor, err := GC.client.Database("SteamPriceDB").Collection("SteamGames").Find(ctx, bson.D{})

	if err != nil {
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

// [GET] GetsteamGame
func (GC GameController) GetSteamGame(
	writer http.ResponseWriter,
	request *http.Request,
	params httprouter.Params) {

	//Get the steamGameId from response body
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

	//Create filter to find if the steamappId exist in db
	filter := bson.M{"steam_appid": steamGameId.GameId}

	//Get our SteamGame model
	steam_games := Models.SteamGame{}

	//Find a file that has that bson object id and pass
	//it to steam_games model
	ctx := context.Background()

	if err := GC.client.Database("SteamPriceDB").Collection("SteamGames").FindOne(ctx, filter).Decode(&steam_games); err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusBadRequest)
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
	fmt.Fprintf(writer, "%s\n", steam_Gamejson)
}

// [POST] CreateGame
func (GC GameController) CreateGame(
	writer http.ResponseWriter,
	request *http.Request,
	_ httprouter.Params) {

	//Get the steamGameId from response body
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
	value := gjson.GetBytes(body, ``+steamGameId.GameId+`.success`)

	if value.String() == "false" {
		fmt.Println("Game does not exist in steam API")
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	//Get our SteamGame model
	steam_games := Models.SteamGame{}

	steam_games.Name = gjson.GetBytes(body, ``+steamGameId.GameId+`.data.name`).String()
	steam_games.Steam_appid = gjson.GetBytes(body, ``+steamGameId.GameId+`.data.steam_appid`).String()
	steam_games.Header_image = gjson.GetBytes(body, ``+steamGameId.GameId+`.data.header_image`).String()
	steam_games.Capsule_image = gjson.GetBytes(body, ``+steamGameId.GameId+`.data.capsule_image`).String()
	json.Unmarshal([]byte(gjson.GetBytes(body, ``+steamGameId.GameId+`.data.developers`).String()), &steam_games.Developers)
	json.Unmarshal([]byte(gjson.GetBytes(body, ``+steamGameId.GameId+`.data.publishers`).String()), &steam_games.Publishers)

	currentTime := time.Now()

	fmt.Println(len(steam_games.Price))

	game_price_struct := `[{
							"priceOnDate" : "",
							"Date" : ""
						}]`

	json.Unmarshal([]byte(game_price_struct), &steam_games.Price)

	if gjson.GetBytes(body, ``+steamGameId.GameId+`.data.is_free`).String() == "false" {
		steam_games.Price[0].PriceOnDate = gjson.GetBytes(body, ``+steamGameId.GameId+`.data.price_overview.final_formatted`).String()
		steam_games.Price[0].Date = strconv.FormatInt(currentTime.Unix(), 10)
	} else {
		steam_games.Price[0].PriceOnDate = "free"
		steam_games.Price[0].Date = strconv.FormatInt(currentTime.Unix(), 10)
	}

	steam_games.Platforms = append(steam_games.Platforms, gjson.GetBytes(body, ``+steamGameId.GameId+`.data.platforms.windows`).String())
	steam_games.Platforms = append(steam_games.Platforms, gjson.GetBytes(body, ``+steamGameId.GameId+`.data.platforms.mac`).String())

	if gjson.GetBytes(body, ``+steamGameId.GameId+`.data.metacritic.score`).String() != "" {
		steam_games.Metacritic = append(steam_games.Metacritic, gjson.GetBytes(body, ``+steamGameId.GameId+`.data.metacritic.score`).String())
		steam_games.Metacritic = append(steam_games.Metacritic, gjson.GetBytes(body, ``+steamGameId.GameId+`.data.metacritic.url`).String())
	} else {
		steam_games.Metacritic = append(steam_games.Metacritic, "false")
	}

	//Screenshot and Genres is an array of objects
	//we Unmarshal it to an array of maps
	//and get only the "path_thumbnail" value
	var ScreenshotDat []map[string]string
	var GenreDat []map[string]string

	//Screenshot Information
	json.Unmarshal([]byte(gjson.GetBytes(body, ``+steamGameId.GameId+`.data.screenshots`).String()), &ScreenshotDat)

	for idx := range ScreenshotDat {
		steam_games.Screenshots = append(steam_games.Screenshots, ScreenshotDat[idx]["path_thumbnail"])
	}

	//Genre Information
	json.Unmarshal([]byte(gjson.GetBytes(body, ``+steamGameId.GameId+`.data.genres`).String()), &GenreDat)

	for idx := range GenreDat {
		steam_games.Genres = append(steam_games.Genres, GenreDat[idx]["description"])
	}

	steam_games.Background = gjson.GetBytes(body, ``+steamGameId.GameId+`.data.background`).String()

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
	fmt.Println(CheapSharkDat)

	no_cheapest_price := false

	//============================================= ADED AS DEBUG =============================================
	if len(CheapSharkDat) > 0 {

		GameDeals.CheapSharkId = CheapSharkDat[0]["gameID"]

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

		json.Unmarshal([]byte(gjson.GetBytes(body4, `deals`).String()), &GameDeals.Deals)
	} else {
		no_cheapest_price = true
		GameDeals.CheapSharkId = ""
		GameDeals.Cheapest = append(GameDeals.Cheapest, "")
		GameDeals.Cheapest = append(GameDeals.Cheapest, "")
	}

	//============================================= ADED AS DEBUG =============================================

	//Create a collector from colly package
	//to scrap google for eneba, kinguin deals
	collector := colly.NewCollector()

	game_name_url_format := strings.ReplaceAll(steam_games.Name, " ", "+")

	url_eneba := `http://www.google.com/search?q=eneba+` + game_name_url_format + `+price+pc/`
	url_Allkeyshop := `http://www.google.com/search?q=Allkeyshop+` + game_name_url_format + `+price+pc/`
	url_kinguin := `http://www.google.com/search?q=kinguin+` + game_name_url_format + `+price+pc/`

	collector.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong: ", err)
		return
	})

	price_scrapped := []string{}
	startcount := false
	found := false
	count := 0

	collector.OnHTML("body", func(element *colly.HTMLElement) {

		//Find all span elements on page
		element.ForEach("span", func(_ int, spanelement *colly.HTMLElement) {

			if found == false {
				//If we found a specific word, 7 indexes later is were the
				//price is stored
				if strings.Contains(spanelement.Text, "Αξιολόγηση") {
					startcount = true
				}

				if startcount == true {
					count++
				}

				if count == 7 {
					//If it is a price, we save it
					//if not, we restart
					if strings.Contains(spanelement.Text, "€") {
						price_scrapped = append(price_scrapped, spanelement.Text)
						found = true
					} else {
						count = 0
					}
				}
			}
		})

	})

	//Visit Eneba, kinguin and allkeyshop and
	//get pc price if it exist
	collector.Visit(url_eneba) //id = 25
	startcount = false
	count = 0
	collector.Visit(url_kinguin) //id = 26
	startcount = false
	count = 0
	collector.Visit(url_Allkeyshop) //id = 27

	collector.Wait()

	var more_deals struct {
		StoreId     string `json:"storeId" bson:"storeId"`
		RetailPrice string `json:"retailPrice" bson:"retailPrice"`
		Date        string `json:"date" bson:"date"`
	}
	//? DEBUG ==============================================
	fmt.Println("price_scrapped")
	fmt.Println(price_scrapped)

	for idx := range price_scrapped {
		if price_scrapped[idx] != "" {
			//Remove redundant data
			if strings.Contains(price_scrapped[idx], "Από") {
				price_scrapped[idx] = price_scrapped[idx][6:12]
			}

			tmp := strings.ReplaceAll(price_scrapped[idx], "€", "")
			tmp = strings.ReplaceAll(tmp, " ", "")
			more_deals.RetailPrice = strings.ReplaceAll(tmp, ",", ".")
			more_deals.StoreId = strconv.Itoa(idx + 25)
			more_deals.Date = ""

			//Add values to struct array
			GameDeals.Deals = append(GameDeals.Deals, more_deals)
		}
	}

	fmt.Println("More deals")
	fmt.Println(more_deals)

	cheapest_price, _ := strconv.ParseFloat("1100", 32)
	var current_value float64

	fmt.Println(current_value) //DEBUG!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

	for idx := range GameDeals.Deals {

		if no_cheapest_price == true {

			current_value, _ := strconv.ParseFloat(GameDeals.Deals[idx].RetailPrice, 32)

			if current_value < cheapest_price {
				cheapest_price = current_value
			}
		}
		GameDeals.Deals[idx].Date = strconv.FormatInt(currentTime.Unix(), 10)
	}

	//? Might need tweaking
	if no_cheapest_price == true && cheapest_price != 1100 {
		GameDeals.Cheapest[0] = strconv.FormatFloat(cheapest_price, 'E', -1, 32)
		GameDeals.Cheapest[1] = strconv.FormatInt(currentTime.Unix(), 10)
	} else {
		GameDeals.Cheapest[0] = ""
		GameDeals.Cheapest[1] = ""
	}

	result2, err := GC.client.Database("SteamPriceDB").Collection("GameDeals").InsertOne(ctx, GameDeals)

	if err != nil {
		writer.WriteHeader(http.StatusNotModified)
		fmt.Println(err)
		return
	}

	fmt.Println(result2)

	//Transform the results to json
	steam_Gamejson, err := json.Marshal(steam_games)

	if err != nil {
		fmt.Println(err)
	}

	//Display Ok if everything worked out
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	fmt.Fprintf(writer, "%s\n", steam_Gamejson)
}

// [POST] UpdateGame
func (GC GameController) UpdateGame(
	writer http.ResponseWriter,
	request *http.Request,
	params httprouter.Params) {

	//Get the steamGameId from response body
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

	//Create filter to find if the steamappId exist in db
	filter := bson.M{"steam_appid": steamGameId.GameId}

	//Get our SteamGame model
	steam_games := Models.SteamGame{}

	//Find a file that has that bson object id and pass
	//it to steam_games model
	ctx := context.Background()

	if err := GC.client.Database("SteamPriceDB").Collection("SteamGames").FindOne(ctx, filter).Decode(&steam_games); err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusNotFound)
		return
	}

	if err != nil {
		fmt.Println(err)
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
	value := gjson.GetBytes(body, ``+steamGameId.GameId+`.success`)

	if value.String() == "false" {
		fmt.Println("Game does not exist in steam API")
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	currentTime := time.Now()
	current_price := gjson.GetBytes(body, ``+steamGameId.GameId+`.data.price_overview.final_formatted`).String()
	current_time_unix := strconv.FormatInt(currentTime.Unix(), 10)

	if steam_games.Price[len(steam_games.Price)-1].PriceOnDate != current_price &&
		steam_games.Price[len(steam_games.Price)-1].Date != current_time_unix {

		//Define a temporary Price with the current values
		var price_to_add struct {
			PriceOnDate string `json:"priceOnDate" bson:"priceOnDate"`
			Date        string `json:"date" bson:"date"`
		}

		price_to_add.PriceOnDate = current_price
		price_to_add.Date = current_time_unix

		//Append the struct to the previous array of structs
		steam_games.Price = append(steam_games.Price, price_to_add)

		//Declare a filter that will change field values
		//according to SteamGame struct
		update := bson.M{"$set": bson.M{"price": steam_games.Price}}

		//Incert the new Price to our collection
		result3, err := GC.client.Database("SteamPriceDB").Collection("SteamGames").UpdateOne(ctx, filter, update)

		if err != nil {
			writer.WriteHeader(http.StatusNotModified)
			fmt.Println(err)
			return
		}

		fmt.Println(result3)
	}

	//Get our GameDeals Model
	GameDeals := Models.GameDeals{}

	//Create filter to find if the steamappId exist in db
	filter2 := bson.M{"game": steam_games.Id}

	if err := GC.client.Database("SteamPriceDB").Collection("GameDeals").FindOne(ctx, filter2).Decode(&GameDeals); err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	//Create a temporary array of Deal structs
	var current_deals []struct {
		StoreId     string `json:"storeId" bson:"storeId"`
		RetailPrice string `json:"retailPrice" bson:"retailPrice"`
		Date        string `json:"date" bson:"date"`
	}

	//============================================= ADED AS DEBUG =============================================
	if GameDeals.CheapSharkId != "" {
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

		//Get current values
		current_cheapes := gjson.GetBytes(body4, `cheapestPriceEver.price`).String()

		if GameDeals.Cheapest[0] != current_cheapes {
			GameDeals.Cheapest[0] = gjson.GetBytes(body4, `cheapestPriceEver.price`).String()
			GameDeals.Cheapest[1] = gjson.GetBytes(body4, `cheapestPriceEver.date`).String()
		}

		json.Unmarshal([]byte(gjson.GetBytes(body4, `deals`).String()), &current_deals)

		fmt.Println("current_deals")
		fmt.Println(current_deals)
	}

	//============================================= ADED AS DEBUG =============================================

	//Create a collector from colly package
	//to scrap google for eneba, kinguin deals
	collector := colly.NewCollector()

	game_name_url_format := strings.ReplaceAll(steam_games.Name, " ", "+")

	url_eneba := `http://www.google.com/search?q=eneba+` + game_name_url_format + `+price+pc/`
	url_Allkeyshop := `http://www.google.com/search?q=Allkeyshop+` + game_name_url_format + `+price+pc/`
	url_kinguin := `http://www.google.com/search?q=kinguin+` + game_name_url_format + `+price+pc/`

	collector.OnError(func(_ *colly.Response, err error) {
		fmt.Println("Something went wrong: ", err)
		return
	})

	price_scrapped := []string{}
	startcount := false
	found := false
	count := 0

	collector.OnHTML("body", func(element *colly.HTMLElement) {

		//Find all span elements on page
		element.ForEach("span", func(_ int, spanelement *colly.HTMLElement) {

			if found == false {
				//If we found a specific word, 7 indexes later is were the
				//price is stored
				if strings.Contains(spanelement.Text, "Αξιολόγηση") {
					startcount = true
				}

				if startcount == true {
					count++
				}

				if count == 7 {
					//If it is a price, we save it
					//if not, we restart
					if strings.Contains(spanelement.Text, "€") {
						found = true
						price_scrapped = append(price_scrapped, spanelement.Text)
					} else {
						count = 0
					}
				}
			}
		})

	})

	//Visit Eneba, kinguin and allkeyshop and
	//get pc price if it exist
	collector.Visit(url_eneba) //id = 25
	startcount = false
	count = 0
	collector.Visit(url_kinguin) //id = 26
	startcount = false
	count = 0
	collector.Visit(url_Allkeyshop) //id = 27

	collector.Wait()

	var more_deals struct {
		StoreId     string `json:"storeId" bson:"storeId"`
		RetailPrice string `json:"retailPrice" bson:"retailPrice"`
		Date        string `json:"date" bson:"date"`
	}
	//? DEBUG ==================================================
	fmt.Println("price_scrapped")
	fmt.Println(price_scrapped)
	fmt.Println(len(price_scrapped))

	for idx := range price_scrapped {
		if price_scrapped[idx] != "" {
			//Remove redundant data
			if strings.Contains(price_scrapped[idx], "Από") {
				price_scrapped[idx] = price_scrapped[idx][6:12]
			}

			tmp := strings.ReplaceAll(price_scrapped[idx], "€", "")
			more_deals.RetailPrice = strings.ReplaceAll(tmp, ",", ".")
			more_deals.StoreId = strconv.Itoa(idx + 25)
			fmt.Println(more_deals.StoreId)
			more_deals.Date = ""

			//Add values to struct array
			current_deals = append(current_deals, more_deals)
		}
	}

	fmt.Println("More deals")
	fmt.Println(more_deals)

	for idx := range current_deals {

		err_sameVal := false
		current_price, _ := strconv.ParseFloat(current_deals[idx].RetailPrice, 32)

		//Check if the deal already exist
		for idy := range GameDeals.Deals {

			gamedeals_price, _ := strconv.ParseFloat(GameDeals.Deals[idy].RetailPrice, 32)

			if GameDeals.Deals[idy].Date[0:6] == current_time_unix[0:6] &&
				GameDeals.Deals[idy].StoreId == current_deals[idx].StoreId {

				//if the price was lowered in a day, update the price
				if gamedeals_price < current_price {
					GameDeals.Deals[idy].RetailPrice = current_deals[idx].RetailPrice
				}
				err_sameVal = true
				break
			}
		}

		//If the price does not exist in db
		//append the struct to the previous array of structs
		if err_sameVal != true {
			current_deals[idx].Date = current_time_unix
			GameDeals.Deals = append(GameDeals.Deals, current_deals[idx])
		}
	}

	//Declare a filter that will change field values
	//according to SteamGame struct
	update2 := bson.M{"$set": bson.M{"cheapest": GameDeals.Cheapest, "deals": GameDeals.Deals}}

	//Incert the new deals for our collection
	result4, err := GC.client.Database("SteamPriceDB").Collection("GameDeals").UpdateOne(ctx, filter2, update2)

	if err != nil {
		writer.WriteHeader(http.StatusNotModified)
		fmt.Println(err)
		return
	}

	fmt.Println(result4)

	steam_gamesjson, err := json.Marshal(steam_games)

	if err != nil {
		fmt.Println(err)
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusCreated)
	fmt.Fprintf(writer, "%s\n", steam_gamesjson)
}

// [DELETE] DeleteGame (id)
func (GC GameController) DeleteGame(
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
