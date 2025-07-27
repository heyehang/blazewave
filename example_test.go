package blazewave_test

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/heyehang/blazewave"
	"github.com/heyehang/blazewave/internal/wsjson"
)

func ExampleAccept() {

	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := blazewave.Accept(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer c.CloseNow()

		ctx, cancel := context.WithTimeout(r.Context(), time.Second*10)
		defer cancel()

		var v any
		err = wsjson.Read(ctx, c, &v)
		if err != nil {
			log.Println(err)
			return
		}

		c.Close(blazewave.StatusNormalClosure, "")
	})

	err := http.ListenAndServe("localhost:8080", fn)
	log.Fatal(err)
}

func ExampleDial() {

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, _, err := blazewave.Dial(ctx, "ws://localhost:8080", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.CloseNow()

	err = wsjson.Write(ctx, c, "hi")
	if err != nil {
		log.Fatal(err)
	}

	c.Close(blazewave.StatusNormalClosure, "")
}

func ExampleCloseStatus() {

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, _, err := blazewave.Dial(ctx, "ws://localhost:8080", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.CloseNow()

	_, _, err = c.Reader(ctx)
	if blazewave.CloseStatus(err) != blazewave.StatusNormalClosure {
		log.Fatalf("expected to be disconnected with StatusNormalClosure but got: %v", err)
	}
}

func Example_writeOnly() {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := blazewave.Accept(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}
		defer c.CloseNow()

		ctx, cancel := context.WithTimeout(r.Context(), time.Minute*10)
		defer cancel()

		ctx = c.CloseRead(ctx)

		t := time.NewTicker(time.Second * 30)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				c.Close(blazewave.StatusNormalClosure, "")
				return
			case <-t.C:
				err = wsjson.Write(ctx, c, "hi")
				if err != nil {
					log.Println(err)
					return
				}
			}
		}
	})

	err := http.ListenAndServe("localhost:8080", fn)
	log.Fatal(err)
}

func Example_crossOrigin() {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := blazewave.Accept(w, r, &blazewave.AcceptOptions{
			OriginPatterns: []string{"example.com"},
		})
		if err != nil {
			log.Println(err)
			return
		}
		c.Close(blazewave.StatusNormalClosure, "cross origin blazewave accepted")
	})

	err := http.ListenAndServe("localhost:8080", fn)
	log.Fatal(err)
}

func ExampleConn_Ping() {

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	c, _, err := blazewave.Dial(ctx, "ws://localhost:8080", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.CloseNow()

	ctx = c.CloseRead(ctx)

	for range 5 {
		err = c.Ping(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}

	c.Close(blazewave.StatusNormalClosure, "")
}

func Example_fullStackChat() {
}

func Example_echo() {
}
