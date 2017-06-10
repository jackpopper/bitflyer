# bitflyer
bitFlyer Lightningのapiクライアント

# Example
```Go
package main

import (
    bitflyer "../src/bitflyer"
    "context"
    "fmt"
    "log"
    "time"
)

func main() {
    c := bitflyer.NewClient("", "")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    b, err := c.GetBoard(ctx, "BTC_JPY")
    if err != nil {
        log.Fatalln(err)
    }
    mp := b.MidPrice
    fmt.Printf("%#v : %#v\n", productCode, mp)
}
```
