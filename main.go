package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/unrolled/render"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CustomerValue struct {
	CusName        string `json:"cusName"`
	Age            string `json:"age"`
	Email          string `json:"email"`
	TransactionNum string `json:"transactionNum"`
	RedCoins       string `json:"redCoins"`
}

type UserData struct {
	CusId    string        `json:"cusId"`
	CusValue CustomerValue `json:"cusValue"`
}

type BaseFare struct {
	Price string `json:"baseFare"`
}

var Collection *mongo.Collection

func MongoConnect() *mongo.Client {

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	userName := "teamxenon"
	password := "XenonXenonX"
	opts := options.Client().ApplyURI(fmt.Sprintf("mongodb+srv://%s:%s@cluster0.vvpkath.mongodb.net/?retryWrites=true&w=majority", userName, password)).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		panic(err)
	}
	if err := client.Database("Xenon").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Err(); err != nil {
		panic(err)
	}
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
	Collection = client.Database("Xenon").Collection("redCoin")
	return client
}

// var client *mongo.Client

func main() {
	client := MongoConnect()
	defer func() {
		err := client.Disconnect(context.TODO())
		if err != nil {
			panic(err)
		}
	}()
	m := http.NewServeMux()
	m.Handle("/getRedCoin/", http.HandlerFunc(GetRedcoin))
	m.Handle("/getNewRedCoinCount/", http.HandlerFunc(handlePayment))
	http.Handle("/getRedCoin/", m)
	http.Handle("/getNewRedCoinCount/", m)
	http.ListenAndServe(":3030", nil)
}

func GetRedcoin(w http.ResponseWriter, r *http.Request) {

	response := GetResponse()
	cusID := r.URL.Query().Get("cusId")

	var userData UserData
	filter := bson.M{"cusId": cusID}
	err := Collection.FindOne(context.TODO(), filter).Decode(&userData)
	if err != nil {
		panic(err)
	}

	redCoinValue := userData.CusValue.RedCoins

	response.JSON(w, 200, redCoinValue)
}

func GetResponse() *render.Render {
	response := render.New(render.Options{StreamingJSON: true})
	return response
}

func handlePayment(w http.ResponseWriter, r *http.Request) {

	response := GetResponse()

	client := MongoConnect()
	defer func() {
		err := client.Disconnect(context.TODO())
		if err != nil {
			panic(err)
		}
	}()
	var data BaseFare
	cusId := r.URL.Query().Get("cusId")
	fmt.Println(cusId)
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Failed to decode JSON data from Request", http.StatusBadRequest)
		return
	}
	// fmt.Println(data.Price)
	var userData UserData

	filt := bson.M{"cusId": cusId}
	projection := bson.M{"cusValue.redCoins": 1}
	// if err != nil {
	// panic(err)
	// }
	// defer cursor.Close(context.Background())

	err := Collection.FindOne(context.Background(), filt, options.FindOne().SetProjection(projection)).Decode(&userData)

	if err != nil {
		panic(err)
	}

	fmt.Println(userData)

	fare, err := strconv.ParseFloat(data.Price, 64)
	if err != nil {
		http.Error(w, "Failed to convert JSON data from Request", http.StatusInternalServerError)
		return
	}

	if fare < 0 {
		http.Error(w, "Fare Price is not positive", http.StatusExpectationFailed)
		return
	}

	numRedCoin := fare * 0.02
	fmt.Println("Number of redCoins is: ", numRedCoin)
	var existingRedCoinFloat float64

	existingRedCoin := userData.CusValue.RedCoins

	existingRedCoinFloat, err = strconv.ParseFloat(existingRedCoin, 64)
	// }
	if err != nil {
		log.Fatal("Problem in parsing existing redcoin")
	}

	newRedCoinCount := numRedCoin + existingRedCoinFloat

	newRedCointCountString := strconv.FormatFloat(newRedCoinCount, 'f', -1, 64)

	filter := bson.M{"cusId": cusId}

	update := bson.M{"$set": bson.M{"cusValue.redCoins": newRedCointCountString}}

	_, er := Collection.UpdateOne(context.Background(), filter, update)
	if er != nil {
		http.Error(w, "Failed to insert data into MongoDB", http.StatusInternalServerError)
		return
	}

	response.JSON(w, 200, newRedCointCountString)

	w.WriteHeader(http.StatusCreated)
}
