package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"sync"
)

type DB struct {
	path string
	mutex sync.RWMutex
}

type Chirp struct {
	ID int `json:"id"`
	Body string `json:"body"`
}

type User struct {
	ID int `json:"id"`
	Email string `json:"email"`
	Password string `json:"password"`
}

type UserNoPassword struct {
	ID int `json:"id"`
	Email string `json:"email"`
	Token string `json:"token"`
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users map[int]User `json:"users"`
}

var ErrChirpID = errors.New("chirp id out of range")

func NewDB(path string) (*DB, error) {
	fp := path + "/database.json"
	
	db := DB{path: fp}

	err := db.EnsureDB()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if err := os.Truncate(fp, 0); err != nil {
		fmt.Println("failed to clear file")
		return nil, err
	}

	empty := DBStructure{
		Chirps: make(map[int]Chirp),
		Users: make(map[int]User),
	}

	err = db.writeDB(empty)

	return &db, nil
}

func (db *DB) EnsureDB() error {
	f, err := os.Open(db.path)
	defer f.Close()

	if errors.Is(err, fs.ErrNotExist) {
		fmt.Println("file does not exist: creating...")
		_, err := os.Create(db.path)
		if err != nil {
			fmt.Println(err)
			return err
		}
	} else if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (db *DB) LoadDB() (*DBStructure, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	
	f, err := os.ReadFile(db.path)
	if err != nil {
		return nil, err
	}

	
	dbStructure := DBStructure{}
	if err := json.Unmarshal(f, &dbStructure); err != nil {
		fmt.Println("Error decoding db json file", err)
		return nil, err
	}

	return &dbStructure, nil
}

func (db *DB) CreateChirp(body string) (Chirp, error) {
	dbStructure, err := db.LoadDB()
	if err != nil {
		fmt.Println("error loading db")
		return Chirp{}, err
	}

	chirps := dbStructure.Chirps

	var maxNum int
	for n := range chirps {
		maxNum = n
		break
	}

	for n := range chirps {
		if n > maxNum {
			maxNum = n
		}
	}

	id := maxNum + 1

	newChirp := Chirp{
		ID: id,
		Body: body,
	}

	dbStructure.Chirps[id] = newChirp
	db.writeDB(*dbStructure)

	return newChirp, nil
}

func (db *DB) GetChirps() ([]Chirp, error) {

	dbStructure, err := db.LoadDB()
	if err != nil {
		fmt.Println("Error loading db structure")
		return nil, err
	}

	chirps := dbStructure.Chirps

	v := make([]Chirp, 0, len(chirps))

	for _, value := range chirps {
		v = append(v, value)
	}

	sort.Slice(v, func(i, j int) bool {
		return v[i].ID < v[j].ID
	})

	return v, nil
}

func (db *DB) writeDB(dbStructure DBStructure) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	f, err := json.Marshal(dbStructure)
	if err != nil {
		fmt.Println("error marshalling json: ", err)
		return err
	}

	err = os.WriteFile(db.path, f, 0644)
	if err != nil {
		fmt.Println("error writing to db: ", err)
		return err
	}
	return nil
}

func (db *DB) GetChirpByID(idString string) (Chirp, error) {
	dbStructure, err := db.LoadDB()
	if err != nil {
		fmt.Println("Error loading db structure")
		return Chirp{}, err
	}

	id, err := strconv.Atoi(idString)
	if err != nil {
		fmt.Println("Error converting id")
		return Chirp{}, err
	}

	if id > len(dbStructure.Chirps) {
		return Chirp{}, ErrChirpID
	}

	return dbStructure.Chirps[id], nil
}

func (db *DB) CreateUser(email string, hashed string) (User, error) {
	dbStructure, err := db.LoadDB()
	if err != nil {
		fmt.Println("Error loading db structure")
		return User{}, err
	}

	users := dbStructure.Users

	var maxNum int
	for n := range users {
		maxNum = n
		break
	}

	for n := range users {
		if n > maxNum {
			maxNum = n
		}
	}

	id := maxNum + 1

	newUser := User{
		ID: id,
		Email: email,
		Password: hashed,
	}

	dbStructure.Users[id] = newUser
	db.writeDB(*dbStructure)

	return newUser, nil
}

func (db *DB) GetUserByEmail(email string) (User, error) {
	dbStructure, err := db.LoadDB()
	if err != nil {
		fmt.Println("Error loading db structure")
		return User{}, err
	}

	users := dbStructure.Users
	user := User{}

	for n := range users {
		if users[n].Email == email {
			user = users[n]
		} else if n == len(users) {
			return User{}, errors.New("User not found")
		}
	}

	return user, nil
}
