package main

import (
"fmt"
"os"

"overshare-backend/imaging"
)

func main() {
files := []string{"demo-as1.jpg", "demo-as3.jpg", "demo-as5.png", "demo-as6.jpg"}
os.MkdirAll("test-output", 0o755)

for _, f := range files {
src := "..\\" + f
dst := "test-output\\stripped-" + f
if err := imaging.StripMetadata(src, dst); err != nil {
fmt.Printf("%s: FAILED - %v\n", f, err)
continue
}
fmt.Printf("%s: OK -> %s\n", f, dst)
}
}
