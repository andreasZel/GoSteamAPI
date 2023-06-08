package Controllers

import (
	//"encoding/json"
	//"fmt"
	//"net/http"

	//"github.com/andreasZel/GoSteamAPI/GoSteamAPI/Models"
	//"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
	//"gopkg.in/mgo.v2/bson"
)

type GameDealsController struct {
	session *mgo.Session
}

func NewGameDealsController(session *mgo.Session) *GameDealsController {
	return &GameDealsController{session}
}

