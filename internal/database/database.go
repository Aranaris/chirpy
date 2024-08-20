package database

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
)

type DB struct {
	path string
	mutex  *sync.RWMutex
}

type Chirp struct {
	body string
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
}

func NewDB(path string) (*DB, error) {
	fp := path + "/database.json"
	
	db := DB{path: fp}
	err := db.EnsureDB()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	if err := os.Truncate(fp, 0); err != nil {
		fmt.Println("failed to truncate")
		return nil, err
	}

	return &db, nil
}

func (db *DB) EnsureDB() error {
	f, err := os.Open(db.path)
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
	f.Close()
	return nil
}


