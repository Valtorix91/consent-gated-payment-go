package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: go run . <authorize|revoke> <user_id> <amount_cents>")
		os.Exit(2)
	}
	client, err := NewConsentClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ctx := context.Background()
	if os.Args[1] == "revoke" {
		err = client.Revoke(ctx, os.Args[2])
		if err == nil {
			fmt.Printf("consent revoked for %s\n", os.Args[2])
		}
	} else if os.Args[1] == "authorize" {
		event := PaymentEvent{UserID: os.Args[2], Currency: "USD"}
		if _, err = fmt.Sscan(os.Args[3], &event.Amount); err == nil {
			var notice AuditNotice
			notice, err = authorizePayment(ctx, client, event)
			if err == nil {
				err = json.NewEncoder(os.Stdout).Encode(notice)
			}
		}
	} else {
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
