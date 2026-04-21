package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/sewnie/otoko/bandcamp"
)

type options struct {
	// Pretty unconventional way of setting the client and evaluating the identity cookie.
	Client *Client `kong:"name=identity,help='Bandcamp identity cookie value, fetched from browser if empty',required,env=BANDCAMP_IDENTITY"`

	Value valueCmd `kong:"cmd,help='Calculate the total value of your Bandcamp collection'"`
	Sync  syncCmd  `kong:"cmd,help='Download and synchronize your collection to a local directory'"`
	List  listCmd  `kong:"cmd,help='Display detailed metadata for tracks and albums in your collection'"`
}

func main() {
	var o options
	app := kong.Parse(&o,
		kong.UsageOnError())

	err := app.Run(o.Client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func (o *options) AfterApply() error {
	f, err := o.Client.GetFan()
	if err != nil {
		return err
	}
	o.Client.Fan = f
	return nil
}

type Client struct {
	Fan *bandcamp.Fan

	*bandcamp.Client
}

func (c *Client) UnmarshalText(b []byte) error {
	identity := string(b)
	*c = Client{Client: bandcamp.New(identity)}
	return nil
}
