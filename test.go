// package main

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"net/url"
// )

// func testUrl() {

// 	u := &url.URL{
// 		Scheme:   "env",
// 		Fragment: "Victor França Lopes",
// 	}

// 	// q := u.Query()

// 	// q.Set("k", "Victor França Lopes")

// 	// u.RawQuery = q.Encode()

// 	fmt.Println(u.String())

// 	u2, err := url.Parse(u.String())

// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}

// 	data, _ := json.Marshal(u2)
// 	fmt.Printf("%s\n", data)
// 	fmt.Println(u.Fragment, "=", u2.Fragment)
// 	fmt.Println(u.Fragment, "=", u2.Fragment)
// }

// type anystruct struct {
// 	Buff *bytes.Buffer
// }

// func testMarshalJsonBytesBuffer() {
// 	b := bytes.NewBuffer(nil)
// 	b.WriteString("asdas/123974dasdqwe123")

// 	fmt.Println(b.Len())

// 	data, _ := json.Marshal(&anystruct{b})

// 	fmt.Printf("%s\n", data)
// }

// func main() {
// 	testMarshalJsonBytesBuffer()
// }
