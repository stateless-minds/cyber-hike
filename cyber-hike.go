package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/foolin/mixer"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/mitchellh/mapstructure"
	shell "github.com/stateless-minds/go-ipfs-api"
)

const dbNameRide = "ride"

const encPassword = "mysecretpassword"

const (
	topicCreateRide = "create-ride"
	topicUpdateRide = "update-ride"
)

type cyberhike struct {
	citizenID     string
	isParticipant bool
	app.Compo
	sh    *shell.Shell
	sub   *shell.PubSubSubscription
	rides map[string]Ride
	alert string
}

// Ride is the struct holding the journey
type Ride struct {
	ID           string   `mapstructure:"_id" json:"_id" validate:"uuid_rfc4122"`
	Route        string   `mapstructure:"route" json:"route" validate:"uuid_rfc4122"`
	DateTime     string   `mapstructure:"datetime" json:"datetime" validate:"uuid_rfc4122"`
	Pickup       string   `mapstructure:"pickup" json:"pickup" validate:"uuid_rfc4122"`
	Dropoff      string   `mapstructure:"dropoff" json:"dropoff" validate:"uuid_rfc4122"`
	Seats        int      `mapstructure:"seats" json:"seats" validate:"uuid_rfc4122"`
	Participants []string `mapstructure:"participants" json:"participants" validate:"uuid_rfc4122"`
}

func (c *cyberhike) OnMount(ctx app.Context) {
	sh := shell.NewShell("localhost:5001")
	c.sh = sh
	myPeer, err := c.sh.ID()
	if err != nil {
		log.Fatal(err)
	}

	c.rides = make(map[string]Ride)
	c.citizenID = myPeer.ID

	c.subscribeToCreateRideTopic(ctx)
	c.subscribeToUpdateRideTopic(ctx)

	ctx.Async(func() {
		// c.DeleteRides(ctx)
		v := c.FetchRides(ctx)
		var vv []interface{}
		err := json.Unmarshal(v, &vv)
		if err != nil {
			log.Fatal(err)
		}

		for _, ii := range vv {
			r := Ride{}
			err = mapstructure.Decode(ii, &r)
			if err != nil {
				log.Fatal(err)
			}

			tf, err := time.Parse("2006-01-02T15:04", r.DateTime)

			if time.Now().After(tf) {
				ctx.Async(func() {
					err = c.sh.OrbitDocsDelete(dbNameRide, r.ID)
					if err != nil {
						ctx.Dispatch(func(ctx app.Context) {
							fmt.Println("Error: could not delete ride")
							log.Fatal(err)
						})
					}
				})
				return
			}

			ctx.Dispatch(func(ctx app.Context) {
				c.rides[r.ID] = r
			})
		}
	})
}

func (c *cyberhike) DeleteRides(ctx app.Context) {
	err := c.sh.OrbitDocsDelete(dbNameRide, "all")
	if err != nil {
		log.Fatal(err)
	}
}

func (c *cyberhike) FetchRides(ctx app.Context) []byte {
	v, err := c.sh.OrbitDocsQuery(dbNameRide, "all", "")
	if err != nil {
		log.Fatal(err)
	}

	return v
}

func (c *cyberhike) subscribeToCreateRideTopic(ctx app.Context) {
	ctx.Async(func() {
		topic := topicCreateRide
		subscription, err := c.sh.PubSubSubscribe(topic)
		if err != nil {
			log.Fatal(err)
		}
		c.sub = subscription
		c.subscriptionCreateRide(ctx)
	})
}

func (c *cyberhike) subscriptionCreateRide(ctx app.Context) {
	ctx.Async(func() {
		defer c.sub.Cancel()
		// wait on pubsub
		res, err := c.sub.Next()
		if err != nil {
			log.Fatal(err)
		}
		// Decode the string data.
		str := string(res.Data)
		// log.Println("Subscriber of topic create-ride received message: " + str)
		ctx.Async(func() {
			c.subscribeToCreateRideTopic(ctx)
		})

		r := Ride{}
		err = json.Unmarshal([]byte(str), &r)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.rides[r.ID] = r
		})
	})
}

func (c *cyberhike) subscribeToUpdateRideTopic(ctx app.Context) {
	ctx.Async(func() {
		topic := topicUpdateRide
		subscription, err := c.sh.PubSubSubscribe(topic)
		if err != nil {
			log.Fatal(err)
		}
		c.sub = subscription
		c.subscriptionUpdateRide(ctx)
	})
}

func (c *cyberhike) subscriptionUpdateRide(ctx app.Context) {
	ctx.Async(func() {
		defer c.sub.Cancel()
		// wait on pubsub
		res, err := c.sub.Next()
		if err != nil {
			log.Fatal(err)
		}
		// Decode the string data.
		str := string(res.Data)
		// log.Println("Subscriber of topic update-ride received message: " + str)
		ctx.Async(func() {
			c.subscribeToUpdateRideTopic(ctx)
		})

		r := Ride{}
		err = json.Unmarshal([]byte(str), &r)
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.rides[r.ID] = r
		})
	})
}

func (c *cyberhike) Render() app.UI {
	return app.Div().Class("app page-wrap xyz-in").Body(
		app.If(len(c.alert) > 0, func() app.UI {
			return app.Div().Class("container alert").Body(
				app.Text(c.alert),
			)
		}),
		app.Div().Class("col center-x space-y-0 pt-50 page-hero").Attr("xyz", "fade small stagger ease-out-back").Body(
			app.H1().Class("title hero-logo xyz-nested").Text("CyberHike"),
			app.H2().Class("subtitle hero-text xyz-nested pt-5").Text("P2P ride share matching"),
		),
		app.Div().ID("what-is").Class("col center-x").Body(
			app.Div().Class("container").Body(
				app.Div().Class("page-section").Attr("xyz", "fade small stagger delay-4 ease-in-out").Body(
					app.Div().Class("section row space-10 xyz-nested").Attr("xyz", "fade left stagger").Body(
						app.Div().Class("card section-item xyz-nested").Body(
							app.Header().Text("Sociable"),
							app.Text("Local-first global community organized around sharing economy."),
							app.Footer().Body(app.Strong().Text("Every shared journey is free.")),
						),
						app.Div().Class("card with-leaves section-item xyz-nested").Body(
							app.Header().Text("Enjoyable"),
							app.Text("Post the coordinates, p2p will do the matching."),
							app.Footer().Body(app.Strong().Text("Because travelling together is fun.")),
						),
					),
					app.Div().Class("section row space-10 xyz-nested").Attr("xyz", "fade left stagger").Body(
						app.Div().Class("card with-scales section-item xyz-nested").Body(
							app.Header().Text("Transparent"),
							app.Text("Free, no registration, no ads, no tracking, no data collection."),
							app.Footer().Body(app.Strong().Text("No strings attached.")),
						),
						app.Div().Class("card with-wood section-item xyz-nested").Body(
							app.Header().Text("Sustainable"),
							app.Text("By sharing a journey we reduce unnecessary traffic."),
							app.Footer().Body(app.Strong().Text("Promote sharing economy.")),
						),
					),
				),
			),
		),
		app.Div().Class("col center-x space-y-0 pt-25 page-hero").Attr("xyz", "fade small stagger ease-out-back").Body(
			app.H1().Class("title hero-logo xyz-nested").Text("Ride"),
			app.H2().Class("subtitle hero-text xyz-nested").Text("With companion!"),
		),
		app.Div().ID("give").Class("col center-x space-y-0 page-hero").Attr("xyz", "fade small stagger ease-out-back").Body(
			app.Div().Class("col container").Body(
				app.Form().Class("col center-x space-10").Body(
					app.Label().ID("label-route").For("route").Class("xyz-nested").Body(
						app.Text("Route: "),
						app.Input().ID("route").Class("input ml-55 is-success xyz-nested").Size(23).Placeholder("Varna-Sofia").Required(true),
					),
					app.Label().ID("label-datetime").For("datetime").Class("xyz-nested").Body(
						app.Text("Departure: "),
						app.Input().ID("datetime").Class("input ml-25 is-success xyz-nested").Type("datetime-local").Min(time.Now().Format("2006-01-02T15:04")).Required(true),
					),
					app.Label().ID("label-pickup").For("pickup").Class("xyz-nested").Body(
						app.Text("Pickup: "),
						app.Input().ID("pickup").Class("input ml-50 is-success xyz-nested").Size(23).Placeholder("coordinates: 42.1396, 24.7616").Required(true),
					),
					app.Label().ID("label-dropoff").For("dropoff").Class("xyz-nested").Body(
						app.Text("Dropoff: "),
						app.Input().ID("dropoff").Class("input ml-40 is-success xyz-nested").Size(23).Placeholder("coordinates: 42.1396, 24.7616").Required(true),
					),
					app.Label().ID("label-seats").For("seats").Class("xyz-nested").Body(
						app.Text("Seats: "),
						app.Input().ID("seats").Class("input ml-55 is-success xyz-nested").Placeholder("5").Type("number").Min(1).Required(true),
					),
					app.Button().Class("button mt-30 is-info xyz-nested").Type("submit").Text("Submit"),
				).OnSubmit(c.onSubmitRide),
			),
		),
		app.Div().Class("col center-x space-y-0 pt-25 page-hero").Attr("xyz", "fade small stagger ease-out-back").Body(
			app.H1().Class("title hero-logo xyz-nested").Text("Share"),
			app.H2().Class("subtitle hero-text xyz-nested").Text("The journey!"),
		),
		app.Div().ID("gallery").Class("col center-x space-y-0 page-hero").Attr("xyz", "fade small stagger ease-out-back").Body(
			app.Div().Class("col container gallery-container").Body(
				app.Div().Class("gallery-section row space-10 xyz-nested").Attr("xyz", "fade left stagger").Body(
					app.Range(c.rides).Map(func(i string) app.UI {
						return app.Div().ID(i).Class("card card-gallery section-item xyz-nested").Body(
							app.Div().Class("row p-10").Body(
								app.Header().Text(c.rides[i].Route),
							),
							app.Div().Class("row p-10").Body(
								app.Label().ID("label-datetime").Class("xyz-nested").Body(
									app.Span().Class("p-10").Body(app.Strong().Body(app.Text("Departure: "))),
									app.Span().Body(app.Text(c.rides[i].DateTime)),
								),
							),
							app.Div().Class("row p-10").Body(
								app.Label().ID("label-pickup").Class("xyz-nested").Body(
									app.Span().Class("p-10").Body(app.Strong().Body(app.Text("Pickup: "))),
									app.Span().Body(app.Text(c.rides[i].Pickup)),
								),
							),
							app.Div().Class("row p-10").Body(
								app.Label().ID("label-dropoff").Class("xyz-nested").Body(
									app.Span().Class("p-10").Body(app.Strong().Body(app.Text("Dropoff: "))),
									app.Span().Body(app.Text(c.rides[i].Dropoff)),
								),
							),
							app.Div().Class("row p-10").Body(
								app.Label().ID("label-seats").Class("xyz-nested").Body(
									app.Span().Class("p-10").Body(app.Strong().Body(app.Text("Seats left: "))),
									app.Span().Body(app.Text(c.rides[i].Seats)),
								),
							),
							app.If(len(c.rides[i].Participants) > 0, func() app.UI {
								app.Range(c.rides[i].Participants).Slice(func(n int) app.UI {
									if mixer.DecodeString(encPassword, c.rides[i].Participants[n]) == c.citizenID {
										c.isParticipant = true
									}
									
									return app.If(mixer.DecodeString(encPassword, c.rides[i].Participants[n]) == c.citizenID,
										func() app.UI {
											return app.Span().Class("badge mt-30 is-success xyz-nested").Text("Confirmed")
									})
								})

								return app.If(!c.isParticipant && c.rides[i].Seats > 0, func() app.UI {
									return app.Button().Class("button mt-30 is-success xyz-nested").Text("Join").OnClick(c.onJoinRide)
								})
							}).Else(func() app.UI {
								return app.Button().Class("button mt-30 is-success xyz-nested").Text("Join").OnClick(c.onJoinRide)
							}),
						)
					}),
				).Style("--carousel-start", "-"+strconv.Itoa(len(c.rides)*250)+"px").Style("--carousel-end", strconv.Itoa(len(c.rides)*250)+"px"),
			),
		),
		app.Div().Class("row center-x pb-20").Body(
			app.Span().Body(
				app.Text("Made with "),
				app.I().Class("fa fa-heart pulse").Style("color", "red"),
				app.Text(" by "),
				app.A().Href("https://github.com/stateless-minds").Target("_blank").Text("Stateless Minds"),
			),
		),
	)
}

func (c *cyberhike) Alert(ctx app.Context, msg string) {
	c.alert = msg
	ctx.Async(func() {
		time.Sleep(3 * time.Second)
		c.alert = ""
		ctx.Dispatch(func(ctx app.Context) {})
	})
}

func (c *cyberhike) onSubmitRide(ctx app.Context, e app.Event) {
	e.PreventDefault()
	route := app.Window().GetElementByID("route").Get("value").String()
	datetime := app.Window().GetElementByID("datetime").Get("value").String()
	dt, err := time.Parse("2006-01-02T15:04", datetime)

	if err != nil || time.Now().After(dt) {
		c.Alert(ctx, "Error: time not valid")
	}

	pickup := app.Window().GetElementByID("pickup").Get("value").String()
	dropoff := app.Window().GetElementByID("dropoff").Get("value").String()
	s := app.Window().GetElementByID("seats").Get("value").String()
	seats, err := strconv.Atoi(s)
	if err != nil {
		c.Alert(ctx, "Error: seats not valid")
	}
	if seats < 1 {
		c.Alert(ctx, "Error: seats can not be less than 1")
	}

	var id int

	if len(c.rides) > 0 {
		id = len(c.rides)
	} else {
		id = 0
	}

	r := Ride{
		ID:           strconv.Itoa(id),
		Route:        route,
		DateTime:     datetime,
		Pickup:       pickup,
		Dropoff:      dropoff,
		Seats:        seats,
		Participants: []string{},
	}

	ride, err := json.Marshal(r)
	if err != nil {
		log.Fatal(err)
	}

	ctx.Async(func() {
		err = c.sh.OrbitDocsPut(dbNameRide, ride)
		if err != nil {
			ctx.Dispatch(func(ctx app.Context) {
				c.Alert(ctx, "Error: could not save ride.")
				log.Fatal(err)
			})
		}
		err = c.sh.PubSubPublish(topicCreateRide, string(ride))
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.Alert(ctx, "Success: Ride submited.")
		})
	})
}

func (c *cyberhike) onJoinRide(ctx app.Context, e app.Event) {
	e.PreventDefault()
	pid := ctx.JSSrc().Get("parentElement").Get("id").String()

	citizenID := mixer.EncodeString(encPassword, c.citizenID)
	participants := append(c.rides[pid].Participants, citizenID)

	r := Ride{
		ID:           pid,
		Route:        c.rides[pid].Route,
		DateTime:     c.rides[pid].DateTime,
		Pickup:       c.rides[pid].Pickup,
		Dropoff:      c.rides[pid].Dropoff,
		Seats:        c.rides[pid].Seats - 1,
		Participants: participants,
	}

	rd, err := json.Marshal(r)
	if err != nil {
		log.Fatal(err)
	}

	ctx.Async(func() {
		err = c.sh.OrbitDocsPut(dbNameRide, rd)
		if err != nil {
			ctx.Dispatch(func(ctx app.Context) {
				c.Alert(ctx, "Error: could not update ride.")
				log.Fatal(err)
			})
		}
		err = c.sh.PubSubPublish(topicUpdateRide, string(rd))
		if err != nil {
			log.Fatal(err)
		}

		ctx.Dispatch(func(ctx app.Context) {
			c.Alert(ctx, "Success: ride updated.")
		})
	})
}

func main() {
	app.Route("/", func() app.Composer{
		return &cyberhike{}
	})
	app.RunWhenOnBrowser()
	http.Handle("/", &app.Handler{
		Name:        "cyberhike",
		Description: "P2P ride sharing community",
		Styles: []string{
			"https://cdnjs.cloudflare.com/ajax/libs/normalize/8.0.1/normalize.min.css",
			"web/app.css",
			"https://cdn.jsdelivr.net/npm/retro.css@1.0.0/css/index.min.css",
			"https://unpkg.com/pattern.css",
			"https://cdn.jsdelivr.net/npm/@animxyz/core",
			"https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css",
		},
	})

	if err := http.ListenAndServe(":8000", nil); err != nil {
		log.Fatal(err)
	}
}
