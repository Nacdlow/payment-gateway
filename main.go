package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/checkout/session"
	"gopkg.in/macaron.v1"
)

var (
	// port is the port the web server is going to run as, if PORT is not set
	// in the environment variable.
	port   = "4242"
	config Configuration
)

type Configuration struct {
	StripeKey      string    `toml:"stripe_key"`
	PublishableKey string    `toml:"publishable_key"`
	Currency       string    `toml:"currency"`
	Products       []Product `toml:"products"`
}

type Product struct {
	Id          string `toml:"id"`
	Title       string `toml:"title"`
	Description string `toml:"description"`
	Price       string `toml:"price"`
}

func loadConfig() {
	if _, err := toml.DecodeFile("./config.toml", &config); err != nil {
		log.Panicf("Failed to load the config. Make sure it exists! Error: %s", err)
	}
}

func main() {
	loadConfig()
	stripe.Key = config.StripeKey
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	m := macaron.Classic()
	m.Use(macaron.Renderer())

	m.Get("/", func(ctx *macaron.Context) {
		ctx.HTML(200, "hello")
	})

	m.Get("/checkout/:id", func(ctx *macaron.Context) {
		for _, prod := range config.Products {
			if ctx.Params("id") == prod.Id {
				prodURL := fmt.Sprintf("https://market.nacdlow.com/%s", prod.Id)
				succURL := fmt.Sprintf("https://app.nacdlow.com/settings/plugins/%s", prod.Id)
				price, err := strconv.Atoi(strings.ReplaceAll(prod.Price, ".", ""))
				if err != nil {
					ctx.PlainText(200, []byte("Invalid product price!"))
					return
				}
				price64 := int64(price)
				prodImg := fmt.Sprintf("https://market.nacdlow.com/%s.png", prod.Id)
				quantity := int64(1)

				prod := &stripe.CheckoutSessionLineItemParams{
					Description: &prod.Description,
					Name:        &prod.Title,
					Amount:      &price64,
					Images:      []*string{&prodImg},
					Currency:    &config.Currency,
					Quantity:    &quantity,
				}

				payType := "card"

				sess := &stripe.CheckoutSessionParams{
					CancelURL:          &prodURL,
					SuccessURL:         &succURL,
					PaymentMethodTypes: []*string{&payType},
					LineItems:          []*stripe.CheckoutSessionLineItemParams{prod},
				}
				checkout, err := session.New(sess)
				if err != nil {
					log.Println("Failed to create new session! ", err)
					ctx.PlainText(500, []byte("Stripe error while creating a new session."))
					return
				}
				ctx.Data["SessionID"] = checkout.ID
				ctx.Data["PublishableKey"] = config.PublishableKey
				ctx.HTML(200, "checkout")
				return
			}
		}
		ctx.PlainText(200, []byte("Invalid product!"))
	})

	log.Println("Running on 0.0.0.0:" + port)
	log.Println(http.ListenAndServe("0.0.0.0:"+port, m))
}
