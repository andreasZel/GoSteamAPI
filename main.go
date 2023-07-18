package main

import (
	"context"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"fmt"
	"log"
	"net/http"
	"github.com/andreasZel/GoSteamAPI/Controllers"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
)

func main() {

	router := httprouter.New()

	//Get client to pass it to GameController 
	//by connecting to Mongo Db
	client := getClient()

	GameController := Controllers.NewGameController(client)
	DealsController := Controllers.NewGameDealsController(client)

	//For SteamGames Document
	router.GET("/SteamGames/", GameController.AllGames)
	router.POST("/SteamGames/GetGameDeals/", DealsController.GetGameDeals)
	router.POST("/SteamGames/GetSteamGame/", GameController.GetSteamGame)
	router.GET("/SteamGames/GetSteamGamesNameList/", GameController.GetSteamGames)
	router.POST("/SteamGames/CreateGame/", GameController.CreateGame)
	router.POST("/SteamGames/UpdateGame/", GameController.UpdateGame)
	router.DELETE("/SteamGames/DeleteGame/", GameController.DeleteGame) 
	
	//Disconect from db when exiting program
	defer client.Disconnect(context.Background())

	//Enable CORS, only for our steam-price server
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"HOST URL"},
		   AllowedMethods: []string{"GET", "POST", "DELETE", "PUT", "OPTIONS"},
	})
	
	err := http.ListenAndServe(":PORT", c.Handler(router))

	if err != nil {	
		fmt.Println(err)
	}
}

//Connects to Mongo Cluster and returns client
func getClient() *mongo.Client{

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()
	
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("ATLAS_URI"))

	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	err = client.Ping(ctx, readpref.Primary())
	
	if err != nil {
		log.Fatal(err)
	}

	return client
} 

