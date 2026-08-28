package main

import (
"fmt"
"overshare-backend/metadata"
)

func main() {
gps, err := metadata.ExtractGPSMetadata("..\\test-real-gps.jpg")
if err != nil {
fmt.Println("ERROR:", err)
return
}
if gps == nil {
fmt.Println("No GPS data found (nil)")
return
}
fmt.Printf("Latitude:  %f\n", gps.Latitude)
fmt.Printf("Longitude: %f\n", gps.Longitude)
fmt.Printf("Timestamp: %s\n", gps.Timestamp)
}
