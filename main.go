package main

import (
    "workout-tracker/database"
    "workout-tracker/routes"
    "fmt"
    "log"
    "net/http"
)

func main() {
    database.ConnectDB()
    r := routes.SetupRoutes()
    fmt.Println("Server running")
    log.Fatal(http.ListenAndServe(":8080", r))
}
