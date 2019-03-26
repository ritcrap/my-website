package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	
	"github.com/ritwik310/my-website/server/models"
	"github.com/ritwik310/my-website/server/config"
	
)

// Type for Data return from Google Oauth flow
type googleUserData struct {
	Email string `json:"email"`
	GoogleID    string `json:"id"`
}

var oauthStateString = "pseudo-random" // Oauth State

// GetUserInfo - gets User (Admin) info from Google APIs
func GetUserInfo(state string, code string) ([]byte, error) {
	// Checking Oauth-State
	if state != oauthStateString {
		return nil, fmt.Errorf("Invalid oauth state")
	}

	// Getting Token
	token, err := googleOauthConfig.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, fmt.Errorf("Code exchange failed: %s", err.Error())
	}

	// Getting User info
	response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %s", err.Error())
	}

	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response body: %s", err.Error())
	}

	return content, nil
}

// CheckAuth - ...
func CheckAuth(r *http.Request, client *mongo.Client) (models.Admin, error) {
	var err error
	var admin models.Admin

	var eCookie *http.Cookie // "admin-email" Cookie
	var hCookie *http.Cookie // "admin-id" Cookie

	eCookie, err = r.Cookie("admin-email")
	hCookie, err = r.Cookie("admin-id")
	
	//  Cookie error handleing
	if err != nil {
		return admin, err
	}

	// Mongo Collection
	collection := client.Database("dev_db").Collection("admins")

	// Query User from Database (by Email)
	err = nil
	err = collection.FindOne(context.TODO(), bson.D{bson.E{Key: "email", Value: eCookie.Value}}).Decode(&admin)		
	if err != nil {
		return admin, err
	}

	// Compare Hashed ID
	err = nil
	err = bcrypt.CompareHashAndPassword([]byte(hCookie.Value), []byte(admin.GoogleID))
	if err != nil {
		return admin, err
	}

	return admin, err
}

// CreateOrGetAdmin - queries Admin if it exists else Creates a new Admin in the Database
func CreateOrGetAdmin(content []byte, client *mongo.Client) (models.Admin, error) {
	var err error
	var admin models.Admin

	// Unmarshal Data returned by Google (content []byte)
	var data googleUserData
	err = nil
	data, err = unmarshalAdmin(content)
	if err != nil {
		fmt.Printf("Error: unable to unmarshal %s\n", err)
		return admin, err
	}

	// Checking if Email Allowed or Not
	userUnauth := true // User Unauthorized
	for _, email := range config.Secrets.AdminEmails {
		if email == data.Email {
			userUnauth = false
		}
	}
	// To handle Unauthorized Players
	if userUnauth {
		fmt.Printf("Email %s not Authorized..\n", data.Email)
		return admin, errors.New("Unauthorized")
	}

	// MongoDB collection
	collection := client.Database("dev_db").Collection("admins")

	// Check if user Exists in the Database
	err = nil
	err = collection.FindOne(context.TODO(), bson.D{
		bson.E{Key: "email", Value: data.Email},
		bson.E{Key: "googleid", Value: string(data.GoogleID)},
	}).Decode(&admin)
	
	if err != nil {
		fmt.Printf("Admin not found on Database %s %+v\n", err, admin)
	} else {
		return admin, err
	}

	// Creating a new Admin if none Exist
	newAdmin := models.Admin{
		Email:    data.Email,
		GoogleID: data.GoogleID,
	}

	// Creating Mongo Document
	err = nil
	var result *mongo.InsertOneResult

	result, err = collection.InsertOne(context.TODO(), newAdmin)
	if err != nil {
		fmt.Printf("Error: unable to insert new admin %s\n", err)
		return admin, err
	}

	// Query Created Admin
	err = collection.FindOne(context.TODO(), bson.D{
		// bson.E{Key: "_id", Value: result.InsertedID},
		bson.E{Key: "email", Value: data.Email},
		bson.E{Key: "googleid", Value: string(data.GoogleID)},
	}).Decode(&admin)

	fmt.Printf("DD: data.admin: => %+v", admin)

	fmt.Println("Here: newAdmin", result.InsertedID)

	if err != nil {
		fmt.Println("Error: Saved but couldn't query user")
		return admin, err
	}

	return admin, nil
}

// Unmarshal Byte Slice to Struct
func unmarshalAdmin(content []byte) (googleUserData, error) {
	var admin googleUserData
	err := json.Unmarshal([]byte(content), &admin)
	return admin, err
}

// GenSessionHash ...
func GenSessionHash(id string) (http.Cookie, error) {
	var cookie http.Cookie // Cookie Struct
	fmt.Println("var cookie http.Cookie "+ id)

	// Generating Hash from byteData
	hashedData, hErr := bcrypt.GenerateFromPassword([]byte(id), 14)
	if hErr != nil {
		return cookie, hErr
	}

	cookie.Name = "admin-id"
	cookie.Value = string(hashedData)
	cookie.Expires = time.Now().Add(30 * 24 * time.Hour)
	cookie.Path = "/"
	cookie.Domain = config.Secrets.DomainName

	return cookie, nil
}