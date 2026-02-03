package main

import (
    "database/sql"
    "fmt"
    _ "github.com/lib/pq"
)

func mainTest() {
    db, err := sql.Open("postgres", "host=localhost user=fominyhdenis password=1533 dbname=usda_db sslmode=disable")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    var version string
    err = db.QueryRow("SELECT version()").Scan(&version)
    if err != nil {
        panic(err)
    }
    fmt.Println("✅ PostgreSQL подключён:", version)
}
