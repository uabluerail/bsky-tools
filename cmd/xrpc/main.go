package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/bluesky-social/indigo/xrpc"

	"github.com/uabluerail/bsky-tools/pagination"
	"github.com/uabluerail/bsky-tools/xrpcauth"
)

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "host",
				Usage: "API host name",
				Value: "bsky.social"},
			&cli.PathFlag{
				Name:  "auth-file",
				Usage: "path to the file with auth info"},
		},
		Commands: []*cli.Command{
			{
				Name:    "query",
				Aliases: []string{"get"},
				Action:  runGet,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "all",
						Usage: "automatically re-issue the request with a returned cursor to fetch all the results",
						Value: true},
				},
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func createClient(cCtx *cli.Context) *xrpc.Client {
	f := cCtx.Path("auth-file")
	if f == "" {
		return xrpcauth.NewAnonymousClient(context.Background())
	}
	return xrpcauth.NewClient(context.Background(), f)
}

func runGet(cCtx *cli.Context) error {
	method := cCtx.Args().First()

	params := map[string]interface{}{}

	for _, arg := range cCtx.Args().Tail() {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("argument must be a =-separated key-value pair")
		}
		params[parts[0]] = parts[1]
	}

	client := createClient(cCtx)

	encoder := json.NewEncoder(os.Stdout)

	_, err := pagination.Collect(func(cursor string) (interface{}, string, error) {
		if cursor != "" {
			params["cursor"] = cursor
		}

		var resp interface{}
		err := client.Do(context.Background(), xrpc.Query, "", method, params, nil, &resp)
		if err != nil {
			return nil, "", err
		}

		if err := encoder.Encode(resp); err != nil {
			return nil, "", err
		}

		if !cCtx.Bool("all") {
			return nil, "", nil
		}

		cursor = ""
		if m, ok := resp.(map[string]interface{}); ok && m["cursor"] != nil {
			if s, ok := m["cursor"].(string); ok {
				cursor = s
			}
		}

		return nil, cursor, nil
	})
	return err
}
